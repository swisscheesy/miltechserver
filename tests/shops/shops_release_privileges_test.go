package shops_test

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"testing"
)

// Mirrors the operational privilege gate, exercised as each effective role.
func TestRequiredPrivilegeIntersection(t *testing.T) {
	for _, privileges := range []string{"SELECT", "INSERT", "UPDATE", "SELECT,INSERT,UPDATE"} {
		t.Run(privileges, func(t *testing.T) {
			shopID := seedShop(t, "release privilege proof")
			role := "release_" + uuid.NewString()
			quoted := pq.QuoteIdentifier(role)
			_, err := testDB.Exec(`CREATE ROLE ` + quoted + ` NOLOGIN NOSUPERUSER NOBYPASSRLS NOINHERIT`)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := testDB.Exec(`DROP OWNED BY ` + quoted + `; DROP ROLE ` + quoted)
				require.NoError(t, err)
			})
			for _, grant := range []string{
				`USAGE ON SCHEMA public`,
				`SELECT,UPDATE ON public.shops`,
				`INSERT,SELECT ON public.shop_messages`,
				privileges + ` ON public.shop_message_counters`,
			} {
				_, err = testDB.Exec(`GRANT ` + grant + ` TO ` + quoted)
				require.NoError(t, err)
			}
			tx, _ := allocatorSession(t)
			_, err = tx.Exec(`SET LOCAL ROLE ` + quoted)
			require.NoError(t, err)
			var actual string
			require.NoError(t, tx.QueryRow(`SELECT current_user`).Scan(&actual))
			require.Equal(t, role, actual)
			var allowed bool
			require.NoError(t, tx.QueryRow(`SELECT has_table_privilege(current_user,'public.shop_message_counters','SELECT') AND has_table_privilege(current_user,'public.shop_message_counters','INSERT') AND has_table_privilege(current_user,'public.shop_message_counters','UPDATE')`).Scan(&allowed))
			full := privileges == "SELECT,INSERT,UPDATE"
			require.Equal(t, full, allowed, "every required privilege must hold independently")
			_, err = insertMessageInTx(tx, shopID)
			if full {
				require.NoError(t, err, "full application role legacy-shaped insert")
			} else {
				var pgErr *pq.Error
				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, pq.ErrorCode("42501"), pgErr.Code)
			}
			require.NoError(t, tx.Rollback())
		})
	}
}

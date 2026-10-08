package shops_test

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"log/slog"
	equipmentcore "miltechserver/api/equipment_services/core"
	"miltechserver/bootstrap"
	"testing"
)

func TestEquipmentDeleteCommitFailureDoesNotLogSuccess(t *testing.T) {
	router, shop, vehicle := atomicFixture(t, "atomic-owner")
	_ = router
	id := createEquipmentService(t, "atomic-owner", shop, vehicle, "", "commit failure", nil, false)
	_, err := testDB.Exec(`CREATE FUNCTION test_infrastructure.reject_equipment_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'commit rejected'; END $$; CREATE CONSTRAINT TRIGGER reject_equipment_commit AFTER DELETE ON equipment_services DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION test_infrastructure.reject_equipment_commit()`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := testDB.Exec(`DROP TRIGGER IF EXISTS reject_equipment_commit ON equipment_services; DROP FUNCTION test_infrastructure.reject_equipment_commit()`)
		require.NoError(t, err)
	})
	var output bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(old)
	repo := equipmentcore.NewRepository(testDB)
	user := &bootstrap.User{UserID: "atomic-owner"}
	require.ErrorContains(t, repo.Delete(context.Background(), user, shop, id), "commit rejected")
	require.Equal(t, 1, atomicRowCount(t, "equipment_services", "id=$1", id))
	require.NotContains(t, output.String(), "Equipment service deleted")
	_, err = testDB.Exec(`DROP TRIGGER reject_equipment_commit ON equipment_services`)
	require.NoError(t, err)
	require.NoError(t, repo.Delete(context.Background(), user, shop, id))
	require.Contains(t, output.String(), "Equipment service deleted")
}
func TestPerformanceSamplerFinishIsIdempotent(t *testing.T) {
	_, db, counter := newPerformanceRouter(t)
	finish := measurePerformanceResources(t, db, counter)
	finish("first")
	finish("second")
}

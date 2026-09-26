package shared

import (
	"context"
	"database/sql"
	"sort"
)

// LockReferencedLists must follow LockShopMutation. Attachments do not mutate
// list contents, so admin-only list editing policy does not apply here.
func LockReferencedLists(ctx context.Context, tx *sql.Tx, shopID string, listIDs ...string) error {
	ids := append([]string(nil), listIDs...)
	sort.Strings(ids)
	previous := ""
	for _, id := range ids {
		if id == "" || id == previous {
			continue
		}
		previous = id
		var owner string
		if err := tx.QueryRowContext(ctx, `SELECT shop_id FROM shop_lists WHERE id=$1 FOR UPDATE`, id).Scan(&owner); err != nil {
			return authorizationQueryError("lock referenced list", ErrListNotFound, err)
		}
		if owner != shopID {
			return ErrListAccessDenied
		}
	}
	return nil
}

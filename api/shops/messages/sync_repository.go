package messages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

// This reader targets the schema from migration 018. Its four reads are served
// only when SHOPS_MESSAGE_SYNC_ENABLED is on (ServiceImpl.syncReader); the
// message_sync capability additionally requires the readiness probe
// (SyncReadiness) to find the counter table and the enabled allocator trigger.
var _ SyncReader = (*RepositoryImpl)(nil)
var _ SyncReader = (*ServiceImpl)(nil)

func (repo *RepositoryImpl) syncRead(ctx context.Context, user *bootstrap.User, shopID string, read func(*sql.Tx) error) error {
	if user == nil || user.UserID == "" {
		return &shared.Failure{Code: "unauthorized", PublicMessage: "Unauthorized", Status: 401}
	}
	if _, err := uuid.Parse(shopID); err != nil {
		return syncInvalid()
	}
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var member bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM shop_members WHERE shop_id = $1 AND user_id = $2)`, shopID, user.UserID).Scan(&member)
	if err != nil {
		return err
	}
	if !member {
		return shared.ErrShopAccessDenied
	}
	if err = read(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func syncWatermark(ctx context.Context, tx *sql.Tx, shopID string) (int64, error) {
	var n int64
	err := tx.QueryRowContext(ctx, `SELECT last_number FROM shop_message_counters WHERE shop_id = $1`, shopID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return watermarkWithoutCounter(ctx, tx, shopID)
	}
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, syncUnavailable()
	}
	return n, nil
}

// The allocator trigger creates a Shop's counter with its first message, so a
// Shop created after migration 018 has none until then and starts at zero.
// Messages without a counter mean numbering is incomplete: never fabricate a
// watermark for them.
func watermarkWithoutCounter(ctx context.Context, tx *sql.Tx, shopID string) (int64, error) {
	var hasMessages bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM shop_messages WHERE shop_id = $1)`, shopID).Scan(&hasMessages)
	if err != nil {
		return 0, err
	}
	if hasMessages {
		return 0, syncUnavailable()
	}
	return 0, nil
}

const syncProjection = `SELECT m.id,m.shop_id,m.user_id,m.message,m.created_at,m.updated_at,m.is_edited,m.parent_id,NULLIF(BTRIM(u.username),''),m.insertion_number
 FROM shop_messages m LEFT JOIN users u ON u.uid = m.user_id WHERE `

type numberedMessage struct {
	row    response.ShopMessageResponse
	number int64
}

func readSyncRows(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]numberedMessage, error) {
	rows, err := tx.QueryContext(ctx, syncProjection+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []numberedMessage{}
	for rows.Next() {
		var value numberedMessage
		// insertion_number is nullable; an unnumbered row is an incomplete
		// migration (503), not a scan failure (500).
		var number sql.NullInt64
		r := &value.row
		if err := rows.Scan(&r.ID, &r.ShopID, &r.UserID, &r.Message, &r.CreatedAt, &r.UpdatedAt, &r.IsEdited, &r.ParentID, &r.AuthorUsername, &number); err != nil {
			return nil, err
		}
		if !number.Valid || number.Int64 < 1 || r.CreatedAt == nil || r.CreatedAt.IsZero() {
			return nil, syncUnavailable()
		}
		value.number = number.Int64
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func syncResponseRows(rows []numberedMessage) []response.ShopMessageResponse {
	result := make([]response.ShopMessageResponse, len(rows))
	for i, row := range rows {
		result[i] = row.row
	}
	return result
}
func historyResult(shopID string, rows []numberedMessage, limit int) (*MessageHistory, error) {
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	page := &MessageHistory{Rows: syncResponseRows(rows), HasMore: more}
	if more {
		cursor, err := encodeMessageCursor(shopID, rows[len(rows)-1].row)
		if err != nil {
			return nil, err
		}
		page.NextCursor = &cursor
	}
	return page, nil
}
func (repo *RepositoryImpl) InitialMessages(ctx context.Context, user *bootstrap.User, shopID string, limit int) (*MessageInitial, error) {
	if !validSyncLimit(limit) {
		return nil, syncInvalid()
	}
	var page *MessageInitial
	err := repo.syncRead(ctx, user, shopID, func(tx *sql.Tx) error {
		watermark, err := syncWatermark(ctx, tx, shopID)
		if err != nil {
			return err
		}
		rows, err := readSyncRows(ctx, tx, `m.shop_id = $1 ORDER BY m.created_at DESC,m.id DESC LIMIT $2`, shopID, limit+1)
		if err != nil {
			return err
		}
		history, err := historyResult(shopID, rows, limit)
		if err != nil {
			return err
		}
		page = &MessageInitial{Rows: history.Rows, OlderCursor: history.NextCursor, HasOlder: history.HasMore, Watermark: strconv.FormatInt(watermark, 10)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}
func (repo *RepositoryImpl) MessageHistory(ctx context.Context, user *bootstrap.User, shopID, cursor string, limit int) (*MessageHistory, error) {
	if !validSyncLimit(limit) {
		return nil, syncInvalid()
	}
	anchor, err := decodeMessageCursor(shopID, cursor)
	if err != nil {
		return nil, err
	}
	var page *MessageHistory
	err = repo.syncRead(ctx, user, shopID, func(tx *sql.Tx) error {
		rows, err := readSyncRows(ctx, tx, `m.shop_id = $1 AND (m.created_at,m.id) < ($2,$3) ORDER BY m.created_at DESC,m.id DESC LIMIT $4`, shopID, anchor.CreatedAt, anchor.ID, limit+1)
		if err != nil {
			return err
		}
		page, err = historyResult(shopID, rows, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}
func (repo *RepositoryImpl) CatchUpMessages(ctx context.Context, user *bootstrap.User, shopID, after string, through *string, limit int) (*MessageCatchUp, error) {
	if !validSyncLimit(limit) {
		return nil, syncInvalid()
	}
	start, err := syncNumber(after)
	if err != nil {
		return nil, err
	}
	var bound int64
	if through != nil {
		bound, err = syncNumber(*through)
		if err != nil || bound < start {
			return nil, syncInvalid()
		}
	}
	var page *MessageCatchUp
	err = repo.syncRead(ctx, user, shopID, func(tx *sql.Tx) error {
		current, err := syncWatermark(ctx, tx, shopID)
		if err != nil {
			return err
		}
		if through == nil {
			bound = current
		}
		if bound > current || bound < start {
			return syncInvalid()
		}
		rows, err := readSyncRows(ctx, tx, `m.shop_id = $1 AND m.insertion_number > $2 AND m.insertion_number <= $3 ORDER BY m.insertion_number ASC LIMIT $4`, shopID, start, bound, limit+1)
		if err != nil {
			return err
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		next := bound
		if more {
			next = rows[len(rows)-1].number
		}
		page = &MessageCatchUp{Rows: syncResponseRows(rows), NextAfter: strconv.FormatInt(next, 10), Through: strconv.FormatInt(bound, 10), HasMore: more}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}
func (repo *RepositoryImpl) ReconcileMessages(ctx context.Context, user *bootstrap.User, shopID string, ids []string) (*MessageReconcile, error) {
	ids, err := normalizeSyncIDs(ids)
	if err != nil {
		return nil, err
	}
	var page *MessageReconcile
	err = repo.syncRead(ctx, user, shopID, func(tx *sql.Tx) error {
		page = &MessageReconcile{Rows: []response.ShopMessageResponse{}, MissingIDs: []string{}}
		if len(ids) == 0 {
			return nil
		}
		args := []any{shopID}
		placeholders := make([]string, len(ids))
		for i, id := range ids {
			args = append(args, id)
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		// Foreign IDs and absent IDs have identical results; never query outside Shop.
		rows, err := readSyncRows(ctx, tx, `m.shop_id = $1 AND m.id IN (`+strings.Join(placeholders, ",")+`) ORDER BY m.created_at DESC,m.id DESC`, args...)
		if err != nil {
			return err
		}
		page.Rows = syncResponseRows(rows)
		found := map[string]bool{}
		for _, row := range rows {
			found[row.row.ID] = true
		}
		for _, id := range ids {
			if !found[id] {
				page.MissingIDs = append(page.MissingIDs, id)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}

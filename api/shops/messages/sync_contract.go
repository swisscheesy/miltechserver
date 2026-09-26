package messages

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

type MessageInitial struct {
	Rows        []response.ShopMessageResponse `json:"rows"`
	OlderCursor *string                        `json:"older_cursor"`
	HasOlder    bool                           `json:"has_older"`
	Watermark   string                         `json:"watermark"`
}
type MessageHistory struct {
	Rows       []response.ShopMessageResponse `json:"rows"`
	NextCursor *string                        `json:"next_cursor"`
	HasMore    bool                           `json:"has_more"`
}
type MessageCatchUp struct {
	Rows      []response.ShopMessageResponse `json:"rows"`
	NextAfter string                         `json:"next_after"`
	Through   string                         `json:"through"`
	HasMore   bool                           `json:"has_more"`
}
type MessageReconcile struct {
	Rows       []response.ShopMessageResponse `json:"rows"`
	MissingIDs []string                       `json:"missing_ids"`
}

// SyncReader is additive: legacy message implementations keep their old contract.
type SyncReader interface {
	InitialMessages(context.Context, *bootstrap.User, string, int) (*MessageInitial, error)
	MessageHistory(context.Context, *bootstrap.User, string, string, int) (*MessageHistory, error)
	CatchUpMessages(context.Context, *bootstrap.User, string, string, *string, int) (*MessageCatchUp, error)
	ReconcileMessages(context.Context, *bootstrap.User, string, []string) (*MessageReconcile, error)
}

func syncInvalid() error {
	return &shared.Failure{Code: "invalid", PublicMessage: "Invalid message synchronization request", Status: 400}
}
func syncUnavailable() error {
	return &shared.Failure{Code: "unsupported_contract", PublicMessage: "Message synchronization is unavailable", Status: 503}
}
func validSyncLimit(limit int) bool { return limit >= 1 && limit <= 100 }
func syncNumber(value string) (int64, error) {
	if value == "" {
		return 0, syncInvalid()
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, syncInvalid()
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, syncInvalid()
	}
	return n, nil
}

type messageCursor struct {
	Version   int       `json:"v"`
	ShopID    string    `json:"shop_id"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeMessageCursor(shopID string, row response.ShopMessageResponse) (string, error) {
	if row.CreatedAt == nil || row.CreatedAt.IsZero() {
		return "", syncUnavailable()
	}
	b, err := json.Marshal(messageCursor{2, shopID, *row.CreatedAt, row.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func decodeMessageCursor(shopID, value string) (messageCursor, error) {
	var cursor messageCursor
	if len(value) > 1024 {
		return cursor, syncInvalid()
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		return cursor, syncInvalid()
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&cursor) != nil || d.Decode(new(any)) != io.EOF || cursor.Version != 2 || cursor.ShopID != shopID || cursor.CreatedAt.IsZero() {
		return cursor, syncInvalid()
	}
	if _, err := uuid.Parse(cursor.ID); err != nil {
		return cursor, syncInvalid()
	}
	return cursor, nil
}
func normalizeSyncIDs(ids []string) ([]string, error) {
	if ids == nil || len(ids) > 100 {
		return nil, syncInvalid()
	}
	result := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, syncInvalid()
		}
		id = parsed.String()
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result, nil
}

// F1 has no verified deployed schema/backfill or all-writer allocation evidence.
// Keep every production sync path closed until that release gate is completed.
func (*ServiceImpl) InitialMessages(context.Context, *bootstrap.User, string, int) (*MessageInitial, error) {
	return nil, syncUnavailable()
}
func (*ServiceImpl) MessageHistory(context.Context, *bootstrap.User, string, string, int) (*MessageHistory, error) {
	return nil, syncUnavailable()
}
func (*ServiceImpl) CatchUpMessages(context.Context, *bootstrap.User, string, string, *string, int) (*MessageCatchUp, error) {
	return nil, syncUnavailable()
}
func (*ServiceImpl) ReconcileMessages(context.Context, *bootstrap.User, string, []string) (*MessageReconcile, error) {
	return nil, syncUnavailable()
}

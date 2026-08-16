package shared

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	communityCursorVersion          = 2
	subscriptionUpdateCursorVersion = 1
)

type CommunityCursor struct {
	Version   int           `json:"v"`
	Sort      CommunitySort `json:"sort"`
	Score     *int64        `json:"score,omitempty"`
	UpdatedAt time.Time     `json:"updated_at"`
	Checklist uuid.UUID     `json:"checklist_id"`
}

type SubscriptionUpdateCursor struct {
	Version   int       `json:"v"`
	Checklist uuid.UUID `json:"checklist_id"`
}

func EncodeCommunityCursor(cursor CommunityCursor) (string, error) {
	if err := validateCommunityCursor(cursor); err != nil {
		return "", err
	}
	return encodeCursor(cursor)
}

func DecodeCommunityCursor(value string) (CommunityCursor, error) {
	var cursor CommunityCursor
	if err := decodeCursor(value, &cursor); err != nil {
		return CommunityCursor{}, err
	}
	if err := validateCommunityCursor(cursor); err != nil {
		return CommunityCursor{}, err
	}
	return cursor, nil
}

func EncodeSubscriptionUpdateCursor(cursor SubscriptionUpdateCursor) (string, error) {
	if err := validateSubscriptionUpdateCursor(cursor); err != nil {
		return "", err
	}
	return encodeCursor(cursor)
}

func DecodeSubscriptionUpdateCursor(value string) (SubscriptionUpdateCursor, error) {
	var cursor SubscriptionUpdateCursor
	if err := decodeCursor(value, &cursor); err != nil {
		return SubscriptionUpdateCursor{}, err
	}
	if err := validateSubscriptionUpdateCursor(cursor); err != nil {
		return SubscriptionUpdateCursor{}, err
	}
	return cursor, nil
}

func validateCommunityCursor(cursor CommunityCursor) error {
	if cursor.Version != communityCursorVersion {
		return fmt.Errorf("unsupported community cursor version %d", cursor.Version)
	}
	switch cursor.Sort {
	case CommunitySortTop:
		if cursor.Score == nil {
			return fmt.Errorf("top community cursor requires score anchor")
		}
	case CommunitySortRecent:
		if cursor.Score != nil {
			return fmt.Errorf("recent community cursor must not include score anchor")
		}
	default:
		return fmt.Errorf("unsupported community cursor sort %q", cursor.Sort)
	}
	if cursor.UpdatedAt.IsZero() || cursor.Checklist == uuid.Nil {
		return fmt.Errorf("community cursor requires updated_at and checklist_id anchors")
	}
	return nil
}

func validateSubscriptionUpdateCursor(cursor SubscriptionUpdateCursor) error {
	if cursor.Version != subscriptionUpdateCursorVersion {
		return fmt.Errorf("unsupported subscription update cursor version %d", cursor.Version)
	}
	if cursor.Checklist == uuid.Nil {
		return fmt.Errorf("subscription update cursor requires checklist_id anchor")
	}
	return nil
}

func encodeCursor(cursor any) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(value string, destination any) error {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("decode cursor base64: %w", err)
	}
	if !utf8.Valid(payload) {
		return fmt.Errorf("cursor JSON must contain valid UTF-8")
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode cursor JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("cursor must contain exactly one JSON value")
	}
	return nil
}

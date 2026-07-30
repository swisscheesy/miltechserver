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

const cursorVersion = 1

type CommunityCursor struct {
	Version   int       `json:"v"`
	UpdatedAt time.Time `json:"updated_at"`
	Checklist uuid.UUID `json:"checklist_id"`
}

type SubscriptionUpdateCursor struct {
	Version   int       `json:"v"`
	Checklist uuid.UUID `json:"checklist_id"`
}

func EncodeCommunityCursor(cursor CommunityCursor) (string, error) {
	if cursor.Version != cursorVersion {
		return "", fmt.Errorf("unsupported community cursor version %d", cursor.Version)
	}
	return encodeCursor(cursor)
}

func DecodeCommunityCursor(value string) (CommunityCursor, error) {
	var cursor CommunityCursor
	if err := decodeCursor(value, &cursor); err != nil {
		return CommunityCursor{}, err
	}
	if cursor.Version != cursorVersion {
		return CommunityCursor{}, fmt.Errorf("unsupported community cursor version %d", cursor.Version)
	}
	return cursor, nil
}

func EncodeSubscriptionUpdateCursor(cursor SubscriptionUpdateCursor) (string, error) {
	if cursor.Version != cursorVersion {
		return "", fmt.Errorf("unsupported subscription update cursor version %d", cursor.Version)
	}
	return encodeCursor(cursor)
}

func DecodeSubscriptionUpdateCursor(value string) (SubscriptionUpdateCursor, error) {
	var cursor SubscriptionUpdateCursor
	if err := decodeCursor(value, &cursor); err != nil {
		return SubscriptionUpdateCursor{}, err
	}
	if cursor.Version != cursorVersion {
		return SubscriptionUpdateCursor{}, fmt.Errorf("unsupported subscription update cursor version %d", cursor.Version)
	}
	return cursor, nil
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

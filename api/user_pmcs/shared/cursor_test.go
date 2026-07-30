package shared_test

import (
	"encoding/base64"
	"testing"
	"time"

	"miltechserver/api/user_pmcs/shared"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCommunityCursorRoundTrip(t *testing.T) {
	want := shared.CommunityCursor{
		Version:   1,
		UpdatedAt: time.Date(2026, time.July, 29, 12, 30, 0, 0, time.UTC),
		Checklist: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
	}

	encoded, err := shared.EncodeCommunityCursor(want)
	require.NoError(t, err)
	got, err := shared.DecodeCommunityCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestDecodeCommunityCursorRejectsMalformedValues(t *testing.T) {
	unknownVersion := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"updated_at":"2026-07-29T12:30:00Z","checklist_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`))
	trailingJSON := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"updated_at":"2026-07-29T12:30:00Z","checklist_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"} {}`))
	missingAnchors := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`))
	zeroAnchors := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"updated_at":"0001-01-01T00:00:00Z","checklist_id":"00000000-0000-0000-0000-000000000000"}`))

	tests := []struct {
		name   string
		cursor string
	}{
		{name: "malformed base64", cursor: "%%%"},
		{name: "unknown version", cursor: unknownVersion},
		{name: "trailing JSON", cursor: trailingJSON},
		{name: "missing anchors", cursor: missingAnchors},
		{name: "zero anchors", cursor: zeroAnchors},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := shared.DecodeCommunityCursor(test.cursor)
			require.Error(t, err)
		})
	}
}

func TestSubscriptionUpdateCursorRoundTrip(t *testing.T) {
	want := shared.SubscriptionUpdateCursor{
		Version:   1,
		Checklist: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
	}

	encoded, err := shared.EncodeSubscriptionUpdateCursor(want)
	require.NoError(t, err)
	got, err := shared.DecodeSubscriptionUpdateCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestDecodeSubscriptionUpdateCursorRejectsMissingOrZeroChecklist(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{name: "missing checklist", cursor: base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`))},
		{name: "zero checklist", cursor: base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"checklist_id":"00000000-0000-0000-0000-000000000000"}`))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := shared.DecodeSubscriptionUpdateCursor(test.cursor)
			require.Error(t, err)
		})
	}
}

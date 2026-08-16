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
	topScore := int64(-3)
	top := shared.CommunityCursor{
		Version:   2,
		Sort:      shared.CommunitySortTop,
		Score:     &topScore,
		UpdatedAt: time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC),
		Checklist: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
	}
	recent := shared.CommunityCursor{
		Version:   2,
		Sort:      shared.CommunitySortRecent,
		UpdatedAt: top.UpdatedAt,
		Checklist: top.Checklist,
	}

	for _, want := range []shared.CommunityCursor{top, recent} {
		t.Run(string(want.Sort), func(t *testing.T) {
			encoded, err := shared.EncodeCommunityCursor(want)
			require.NoError(t, err)
			got, err := shared.DecodeCommunityCursor(encoded)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestDecodeCommunityCursorRejectsMalformedValues(t *testing.T) {
	unsupportedVersion := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"sort":"recent","updated_at":"2026-08-16T12:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001"}`))
	unknownSort := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"new","updated_at":"2026-08-16T12:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001"}`))
	topWithoutScore := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"top","updated_at":"2026-08-16T12:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001"}`))
	recentWithScore := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"recent","score":0,"updated_at":"2026-08-16T12:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001"}`))
	zeroTimestamp := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"recent","updated_at":"0001-01-01T00:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001"}`))
	nilChecklist := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"recent","updated_at":"2026-08-16T12:00:00Z","checklist_id":"00000000-0000-0000-0000-000000000000"}`))
	unknownJSONField := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"recent","updated_at":"2026-08-16T12:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001","extra":true}`))
	trailingJSON := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"sort":"recent","updated_at":"2026-08-16T12:00:00Z","checklist_id":"10000000-0000-4000-8000-000000000001"} {}`))

	tests := []struct {
		name   string
		cursor string
	}{
		{name: "malformed base64", cursor: "%%%"},
		{name: "version 1", cursor: unsupportedVersion},
		{name: "unknown sort", cursor: unknownSort},
		{name: "top without score", cursor: topWithoutScore},
		{name: "recent with score", cursor: recentWithScore},
		{name: "zero timestamp", cursor: zeroTimestamp},
		{name: "nil checklist", cursor: nilChecklist},
		{name: "unknown JSON field", cursor: unknownJSONField},
		{name: "trailing JSON", cursor: trailingJSON},
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

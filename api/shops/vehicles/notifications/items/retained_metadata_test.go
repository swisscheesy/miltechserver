package items

import (
	"context"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"testing"
)

func metadataPtr(s string) *string { return &s }
func TestRetainedMetadataRawCandidateComparison(t *testing.T) {
	for _, tc := range []struct {
		name     string
		a, b     ResolvedMetadata
		conflict bool
	}{
		{"null equal", ResolvedMetadata{}, ResolvedMetadata{}, false},
		{"null versus empty", ResolvedMetadata{}, ResolvedMetadata{Nickname: metadataPtr("")}, true},
		{"null versus default unit", ResolvedMetadata{}, ResolvedMetadata{UnitOfMeasure: metadataPtr("EA")}, true},
		{"raw unit spaces", ResolvedMetadata{UnitOfMeasure: metadataPtr("KT")}, ResolvedMetadata{UnitOfMeasure: metadataPtr(" KT ")}, true},
		{"unknown equal", ResolvedMetadata{UnitOfMeasure: metadataPtr("future-code")}, ResolvedMetadata{UnitOfMeasure: metadataPtr("future-code")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.conflict, candidatesDisagree([]metadataCandidate{{ItemID: "z", Nickname: tc.a.Nickname, UnitOfMeasure: tc.a.UnitOfMeasure}, {ItemID: "a", Nickname: tc.b.Nickname, UnitOfMeasure: tc.b.UnitOfMeasure}}))
		})
	}
	require.False(t, candidatesDisagree(nil))
}
func TestMetadataObservationsKeepDurableVersionAndValues(t *testing.T) {
	ctx := MetadataMutationContext(context.Background())
	row := model.ShopNotificationItemMetadata{NotificationID: "notification", Niin: " raw ", Version: 7}
	initial := []metadataCandidate{{ItemID: "durable", Nickname: metadataPtr("Original"), Version: 8}}
	observeMetadata(ctx, row, initial)
	initial[0].ItemID = "mutated slice"
	row.Version = 11
	observed := observeMetadata(ctx, row, []metadataCandidate{{ItemID: "uncommitted", Nickname: metadataPtr("Rejected")}})
	require.EqualValues(t, 7, observed.version)
	require.Equal(t, "durable", observed.candidates[0].ItemID)
	require.Equal(t, "Original", *observed.candidates[0].Nickname)
	row.Niin = "raw"
	require.EqualValues(t, 11, observeMetadata(ctx, row, nil).version)
	row.Niin = " raw "
	require.EqualValues(t, 11, observeMetadata(MetadataMutationContext(ctx), row, nil).version)
}

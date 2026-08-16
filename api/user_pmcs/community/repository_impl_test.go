package community

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
)

func TestContainsModelPatternEscapesLikeMetacharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ordinary text", input: "m1165a1", want: "%m1165a1%"},
		{name: "percent", input: "m%1165", want: "%m!%1165%"},
		{name: "underscore", input: "m_1165", want: "%m!_1165%"},
		{name: "escape character", input: "m!1165", want: "%m!!1165%"},
		{
			name:  "combined",
			input: "m!_1165%a1",
			want:  "%m!!!_1165!%a1%",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, containsModelPattern(test.input))
		})
	}
}

func TestIsCommunityVoteDirection(t *testing.T) {
	require.True(t, isCommunityVoteDirection(1))
	require.True(t, isCommunityVoteDirection(-1))
	require.False(t, isCommunityVoteDirection(0))
	require.False(t, isCommunityVoteDirection(2))
	require.False(t, isCommunityVoteDirection(-2))
}

func TestCommunityBrowseQueryRanksAndBindsEveryFilterCombination(t *testing.T) {
	updatedAt := time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC)
	checklistID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	score := int64(-3)

	tests := []struct {
		name              string
		filter            shared.CommunityBrowseFilter
		wantArguments     []any
		wantViewerJoin    bool
		wantTopOrder      bool
		wantModelPosition int
	}{
		{
			name: "public top without optional filters",
			filter: shared.CommunityBrowseFilter{
				Limit: 20,
				Sort:  shared.CommunitySortTop,
			},
			wantArguments: []any{21},
			wantTopOrder:  true,
		},
		{
			name: "public top model",
			filter: shared.CommunityBrowseFilter{
				Limit:           20,
				NormalizedModel: "m1165_1",
				Sort:            shared.CommunitySortTop,
			},
			wantArguments:     []any{"%m1165!_1%", 21},
			wantTopOrder:      true,
			wantModelPosition: 1,
		},
		{
			name: "public top cursor model",
			filter: shared.CommunityBrowseFilter{
				After: &shared.CommunityCursor{
					Version:   2,
					Sort:      shared.CommunitySortTop,
					Score:     &score,
					UpdatedAt: updatedAt,
					Checklist: checklistID,
				},
				Limit:           7,
				NormalizedModel: "m1165_1",
				Sort:            shared.CommunitySortTop,
			},
			wantArguments:     []any{score, updatedAt, checklistID, "%m1165!_1%", 8},
			wantTopOrder:      true,
			wantModelPosition: 4,
		},
		{
			name: "public recent cursor",
			filter: shared.CommunityBrowseFilter{
				After: &shared.CommunityCursor{
					Version:   2,
					Sort:      shared.CommunitySortRecent,
					UpdatedAt: updatedAt,
					Checklist: checklistID,
				},
				Limit: 7,
				Sort:  shared.CommunitySortRecent,
			},
			wantArguments: []any{updatedAt, checklistID, 8},
		},
		{
			name: "public recent model",
			filter: shared.CommunityBrowseFilter{
				Limit:           7,
				NormalizedModel: "m1165_1",
				Sort:            shared.CommunitySortRecent,
			},
			wantArguments:     []any{"%m1165!_1%", 8},
			wantModelPosition: 1,
		},
		{
			name: "authenticated top without optional filters",
			filter: shared.CommunityBrowseFilter{
				Limit:     7,
				Sort:      shared.CommunitySortTop,
				ViewerUID: "viewer-1",
			},
			wantArguments:  []any{"viewer-1", 8},
			wantViewerJoin: true,
			wantTopOrder:   true,
		},
		{
			name: "authenticated top cursor model",
			filter: shared.CommunityBrowseFilter{
				After: &shared.CommunityCursor{
					Version:   2,
					Sort:      shared.CommunitySortTop,
					Score:     &score,
					UpdatedAt: updatedAt,
					Checklist: checklistID,
				},
				Limit:           7,
				NormalizedModel: "m1165_1",
				Sort:            shared.CommunitySortTop,
				ViewerUID:       "viewer-1",
			},
			wantArguments:     []any{"viewer-1", score, updatedAt, checklistID, "%m1165!_1%", 8},
			wantViewerJoin:    true,
			wantTopOrder:      true,
			wantModelPosition: 5,
		},
		{
			name: "authenticated recent cursor model",
			filter: shared.CommunityBrowseFilter{
				After: &shared.CommunityCursor{
					Version:   2,
					Sort:      shared.CommunitySortRecent,
					UpdatedAt: updatedAt,
					Checklist: checklistID,
				},
				Limit:           7,
				NormalizedModel: "m1165_1",
				Sort:            shared.CommunitySortRecent,
				ViewerUID:       "viewer-1",
			},
			wantArguments:     []any{"viewer-1", updatedAt, checklistID, "%m1165!_1%", 8},
			wantViewerJoin:    true,
			wantModelPosition: 4,
		},
		{
			name: "authenticated recent cursor",
			filter: shared.CommunityBrowseFilter{
				After: &shared.CommunityCursor{
					Version:   2,
					Sort:      shared.CommunitySortRecent,
					UpdatedAt: updatedAt,
					Checklist: checklistID,
				},
				Limit:     7,
				Sort:      shared.CommunitySortRecent,
				ViewerUID: "viewer-1",
			},
			wantArguments:  []any{"viewer-1", updatedAt, checklistID, 8},
			wantViewerJoin: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, arguments := communityBrowseQuery(test.filter)

			require.Contains(t, query, "COALESCE(SUM(vote.direction), 0)::BIGINT")
			require.Contains(t, query, "LEFT JOIN LATERAL")
			require.Equal(t, test.wantArguments, arguments)
			require.NotContains(t, query, "m1165_1")
			if test.wantModelPosition > 0 {
				require.Contains(t, query, "model.normalized_text LIKE $"+strconv.Itoa(test.wantModelPosition)+" ESCAPE '!'")
			}
			if test.wantViewerJoin {
				require.Contains(t, query, "viewer_vote.voter_uid = $1")
			} else {
				require.NotContains(t, query, "viewer_vote.voter_uid")
			}
			if test.wantTopOrder {
				require.Contains(t, query, "ORDER BY ranked.score DESC, ranked.updated_at DESC, ranked.checklist_id ASC")
			} else {
				require.Contains(t, query, "ORDER BY ranked.updated_at DESC, ranked.checklist_id ASC")
			}
		})
	}
}

package community

import (
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

func TestCommunityBrowseQueryUsesParameterizedContainsPattern(t *testing.T) {
	query, arguments := communityBrowseQuery(shared.CommunityBrowseFilter{
		Limit:           20,
		NormalizedModel: "m1165_1",
	})

	require.Contains(
		t,
		query,
		"model.normalized_text LIKE $1 ESCAPE '!'",
	)
	require.NotContains(t, query, "model.normalized_text = $1")
	require.Equal(t, []any{"%m1165!_1%", 21}, arguments)
}

func TestCommunityBrowseQueryKeepsCursorArgumentPositions(t *testing.T) {
	query, arguments := communityBrowseQuery(shared.CommunityBrowseFilter{
		After: &shared.CommunityCursor{
			Version:   1,
			UpdatedAt: time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
			Checklist: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		},
		Limit:           7,
		NormalizedModel: "m1165",
	})

	require.Contains(
		t,
		query,
		"model.normalized_text LIKE $3 ESCAPE '!'",
	)
	require.Contains(t, query, "LIMIT $4")
	require.Equal(t, "%m1165%", arguments[2])
	require.Equal(t, 8, arguments[3])
}

package ps_mag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseIssueFilename(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIssue int
		wantMonth string
		wantYear  int
		wantOK    bool
	}{
		{
			name:      "standard issue",
			input:     "PS_Magazine_Issue_495_February_1994.pdf",
			wantIssue: 495,
			wantMonth: "February",
			wantYear:  1994,
			wantOK:    true,
		},
		{
			name:      "low issue number",
			input:     "PS_Magazine_Issue_1_January_1951.pdf",
			wantIssue: 1,
			wantMonth: "January",
			wantYear:  1951,
			wantOK:    true,
		},
		{
			name:   "wrong prefix",
			input:  "Magazine_Issue_495_February_1994.pdf",
			wantOK: false,
		},
		{
			name:   "not a pdf",
			input:  "PS_Magazine_Issue_495_February_1994.txt",
			wantOK: false,
		},
		{
			name:   "missing year",
			input:  "PS_Magazine_Issue_495_February.pdf",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "stray file in prefix",
			input:  "README.md",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issueNum, month, year, ok := parseIssueFilename(tc.input)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.Equal(t, tc.wantIssue, issueNum)
				require.Equal(t, tc.wantMonth, month)
				require.Equal(t, tc.wantYear, year)
			}
		})
	}
}

// buildTestIssues creates a deterministic slice of test issues for use in filter/sort/paginate tests.
func buildTestIssues() []PSMagIssueResponse {
	return []PSMagIssueResponse{
		{IssueNumber: 100, Month: "January", Year: 1960, Name: "PS_Magazine_Issue_100_January_1960.pdf"},
		{IssueNumber: 200, Month: "June", Year: 1970, Name: "PS_Magazine_Issue_200_June_1970.pdf"},
		{IssueNumber: 300, Month: "March", Year: 1970, Name: "PS_Magazine_Issue_300_March_1970.pdf"},
		{IssueNumber: 400, Month: "August", Year: 1980, Name: "PS_Magazine_Issue_400_August_1980.pdf"},
		{IssueNumber: 495, Month: "February", Year: 1994, Name: "PS_Magazine_Issue_495_February_1994.pdf"},
	}
}

func TestFilterByYear(t *testing.T) {
	issues := buildTestIssues()
	result := filterByYear(issues, 1970)
	require.Len(t, result, 2)
	require.Equal(t, 200, result[0].IssueNumber)
	require.Equal(t, 300, result[1].IssueNumber)
}

func TestFilterByYearNoMatch(t *testing.T) {
	issues := buildTestIssues()
	result := filterByYear(issues, 2099)
	require.Empty(t, result)
}

func TestFilterByIssueNumber(t *testing.T) {
	issues := buildTestIssues()
	result := filterByIssueNumber(issues, 495)
	require.Len(t, result, 1)
	require.Equal(t, "February", result[0].Month)
}

func TestFilterByIssueNumberNoMatch(t *testing.T) {
	issues := buildTestIssues()
	result := filterByIssueNumber(issues, 999)
	require.Empty(t, result)
}

func TestSortIssuesASC(t *testing.T) {
	issues := buildTestIssues()
	// shuffle order first
	issues[0], issues[4] = issues[4], issues[0]
	sortIssues(issues, "asc")
	require.Equal(t, 100, issues[0].IssueNumber)
	require.Equal(t, 495, issues[4].IssueNumber)
}

func TestSortIssuesDESC(t *testing.T) {
	issues := buildTestIssues()
	sortIssues(issues, "desc")
	require.Equal(t, 495, issues[0].IssueNumber)
	require.Equal(t, 100, issues[4].IssueNumber)
}

func TestPaginateIssuesPage1(t *testing.T) {
	// Build 75 issues to test pagination boundary
	issues := make([]PSMagIssueResponse, 75)
	for i := range issues {
		issues[i] = PSMagIssueResponse{IssueNumber: i + 1}
	}
	page, totalPages := paginateIssues(issues, 1, 50)
	require.Len(t, page, 50)
	require.Equal(t, 2, totalPages)
	require.Equal(t, 1, page[0].IssueNumber)
}

func TestPaginateIssuesPage2(t *testing.T) {
	issues := make([]PSMagIssueResponse, 75)
	for i := range issues {
		issues[i] = PSMagIssueResponse{IssueNumber: i + 1}
	}
	page, totalPages := paginateIssues(issues, 2, 50)
	require.Len(t, page, 25)
	require.Equal(t, 2, totalPages)
	require.Equal(t, 51, page[0].IssueNumber)
}

func TestPaginateIssuesBeyondEnd(t *testing.T) {
	issues := make([]PSMagIssueResponse, 10)
	page, totalPages := paginateIssues(issues, 99, 50)
	require.Empty(t, page)
	require.Equal(t, 1, totalPages)
}

func TestPaginateIssuesEmpty(t *testing.T) {
	page, totalPages := paginateIssues([]PSMagIssueResponse{}, 1, 50)
	require.Empty(t, page)
	require.Equal(t, 1, totalPages)
}

func TestGenerateDownloadURLValidation(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.GenerateDownloadURL(context.Background(), "")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GenerateDownloadURL(context.Background(), "   ")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GenerateDownloadURL(context.Background(), "pmcs/some-file.pdf")
	require.ErrorIs(t, err, ErrInvalidBlobPath)

	_, err = svc.GenerateDownloadURL(context.Background(), "ps-mag/some-file.txt")
	require.ErrorIs(t, err, ErrInvalidFileType)

	// path traversal: "ps-mag/../secret.pdf" cleans to "secret.pdf" which fails prefix check
	_, err = svc.GenerateDownloadURL(context.Background(), "ps-mag/../secret.pdf")
	require.ErrorIs(t, err, ErrInvalidBlobPath)
}

func TestListIssuesValidation(t *testing.T) {
	svc := &ServiceImpl{repo: &repoStub{}, cache: newIssueCache(5 * time.Minute)}

	_, err := svc.ListIssues(context.Background(), 0, "asc", nil, nil)
	require.ErrorIs(t, err, ErrInvalidPage)

	_, err = svc.ListIssues(context.Background(), 1, "sideways", nil, nil)
	require.ErrorIs(t, err, ErrInvalidOrder)
}

func TestFilterMatchingLines_SingleMatch(t *testing.T) {
	summary := "Line one\nlubrication point A\nLine three"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"lubrication point A"}, got)
}

func TestFilterMatchingLines_MultipleMatches(t *testing.T) {
	summary := "lubrication A\nno match\nlubrication B"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"lubrication A", "lubrication B"}, got)
}

func TestFilterMatchingLines_NoMatch(t *testing.T) {
	summary := "Line one\nLine two"
	got := filterMatchingLines(summary, "xyz")
	require.Nil(t, got)
}

func TestFilterMatchingLines_CaseInsensitive(t *testing.T) {
	summary := "Check LUBRICATION schedule"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"Check LUBRICATION schedule"}, got)
}

func TestFilterMatchingLines_EmptyLinesSkipped(t *testing.T) {
	summary := "\n  \nlubrication note\n\n"
	got := filterMatchingLines(summary, "lubrication")
	require.Equal(t, []string{"lubrication note"}, got)
}

func TestFilterMatchingLines_EmptySummary(t *testing.T) {
	got := filterMatchingLines("", "lubrication")
	require.Nil(t, got)
}

// repoStub satisfies Repository for unit testing.
type repoStub struct {
	rows  []summaryRow
	total int
	err   error
}

func (r *repoStub) SearchSummaries(_ string, _, _ int) ([]summaryRow, int, error) {
	return r.rows, r.total, r.err
}

func TestServiceSearchSummaries_ReturnsMatchingLines(t *testing.T) {
	stub := &repoStub{
		rows: []summaryRow{
			{FileName: "PS_Magazine_Issue_495_February_1994.pdf", Summary: "Check oil.\nOil level low.\nNo match here."},
		},
		total: 1,
	}
	svc := &ServiceImpl{repo: stub}

	resp, err := svc.SearchSummaries("oil", 1)

	require.NoError(t, err)
	require.Equal(t, 1, resp.TotalCount)
	require.Equal(t, 1, len(resp.Results))
	require.Equal(t, []string{"Check oil.", "Oil level low."}, resp.Results[0].MatchingLines)
	require.Equal(t, "oil", resp.Query)
}

func TestServiceSearchSummaries_EmptyResults(t *testing.T) {
	stub := &repoStub{rows: nil, total: 0}
	svc := &ServiceImpl{repo: stub}

	resp, err := svc.SearchSummaries("oil", 1)

	require.NoError(t, err)
	require.Equal(t, 0, resp.TotalCount)
	require.Empty(t, resp.Results)
}

func TestServiceSearchSummaries_QueryTooShort(t *testing.T) {
	svc := &ServiceImpl{repo: &repoStub{}}

	_, err := svc.SearchSummaries("ab", 1)

	require.ErrorIs(t, err, ErrQueryTooShort)
}

func TestServiceSearchSummaries_InvalidPage(t *testing.T) {
	svc := &ServiceImpl{repo: &repoStub{}}

	_, err := svc.SearchSummaries("oil", 0)

	require.ErrorIs(t, err, ErrInvalidPage)
}

func TestServiceSearchSummaries_RepoError(t *testing.T) {
	stub := &repoStub{err: errors.New("db down")}
	svc := &ServiceImpl{repo: stub}

	_, err := svc.SearchSummaries("oil", 1)

	require.Error(t, err)
}

func TestServiceSearchSummaries_Pagination(t *testing.T) {
	// 35 total results, page size 30 → 2 total pages
	stub := &repoStub{rows: make([]summaryRow, 5), total: 35}
	for i := range stub.rows {
		stub.rows[i] = summaryRow{FileName: "file.pdf", Summary: "oil note"}
	}
	svc := &ServiceImpl{repo: stub}

	resp, err := svc.SearchSummaries("oil", 2)

	require.NoError(t, err)
	require.Equal(t, 35, resp.TotalCount)
	require.Equal(t, 2, resp.TotalPages)
	require.Equal(t, 2, resp.Page)
}

func TestListIssues_UsesCacheOnSecondCall(t *testing.T) {
	// Build a ServiceImpl with a warm cache and a nil blobClient.
	// If listAllIssues tries to use blobClient it will panic — proving the cache
	// was bypassed. If it succeeds the cache was used.
	cached := []PSMagIssueResponse{
		{Name: "PS_Magazine_Issue_1_January_1951.pdf", IssueNumber: 1, Month: "January", Year: 1951},
	}
	c := newIssueCache(5 * time.Minute)
	c.set(cached)

	svc := &ServiceImpl{
		blobClient: nil, // panics if called
		repo:       &repoStub{},
		cache:      c,
	}

	result, err := svc.ListIssues(context.Background(), 1, "asc", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, "PS_Magazine_Issue_1_January_1951.pdf", result.Issues[0].Name)
}

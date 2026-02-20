package ps_mag

import (
	"testing"

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

package analytics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// repoStub captures the arguments of the last IncrementCounter call.
type repoStub struct {
	capturedEventType   string
	capturedEntityKey   string
	capturedEntityLabel string
	err                 error
}

func (r *repoStub) IncrementCounter(eventType, entityKey, entityLabel string) error {
	r.capturedEventType = eventType
	r.capturedEntityKey = entityKey
	r.capturedEntityLabel = entityLabel
	return r.err
}

func TestFormatPSMagLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard filename",
			input: "PS_Magazine_Issue_004_September_1951.pdf",
			want:  "Issue 004 September 1951",
		},
		{
			name:  "no PS_Magazine_ prefix",
			input: "Some_Other_File.pdf",
			want:  "Some Other File",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  "",
		},
		{
			name:  "filename without extension",
			input: "PS_Magazine_Issue_001_January_1951",
			want:  "Issue 001 January 1951",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatPSMagLabel(tc.input))
		})
	}
}

func TestIncrementPSMagDownload_StandardFilename(t *testing.T) {
	repo := &repoStub{}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("PS_Magazine_Issue_004_September_1951.pdf")

	require.NoError(t, err)
	require.Equal(t, "ps_mag_download", repo.capturedEventType)
	require.Equal(t, "PS_MAGAZINE_ISSUE_004_SEPTEMBER_1951.PDF", repo.capturedEntityKey)
	require.Equal(t, "ISSUE 004 SEPTEMBER 1951", repo.capturedEntityLabel)
}

func TestIncrementPSMagDownload_EmptyFilename(t *testing.T) {
	repo := &repoStub{}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("")

	require.NoError(t, err)
	require.Empty(t, repo.capturedEventType) // repo must not be called for empty input
}

func TestIncrementPSMagDownload_WhitespaceOnlyFilename(t *testing.T) {
	repo := &repoStub{}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("   ")

	require.NoError(t, err)
	require.Empty(t, repo.capturedEventType) // repo must not be called for whitespace-only input
}

func TestIncrementPSMagDownload_RepoError(t *testing.T) {
	repo := &repoStub{err: errors.New("db down")}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("PS_Magazine_Issue_001_January_1951.pdf")

	require.Error(t, err)
}

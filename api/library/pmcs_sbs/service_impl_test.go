package pmcs_sbs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase word", "hmmwv", "HMMWV"},
		{"numeric token", "m1", "M1"},
		{"hyphen separator", "m2-bradley", "M2 BRADLEY"},
		{"underscore separator", "m2_bradley", "M2 BRADLEY"},
		{"mixed separators", "m1a1-abrams_tank", "M1A1 ABRAMS TANK"},
		{"already uppercase", "HMMWV", "HMMWV"},
		{"empty string", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatDisplayName(tc.input))
		})
	}
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"filename only", "file.json", "file.json"},
		{"one folder deep", "hmmwv/file.json", "file.json"},
		{"full blob path", "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json", "hmmwv_up_armor_pmcs.json"},
		{"empty string", "", ""},
		{"trailing slash", "pmcs_sbs/hmmwv/", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractFileName(tc.input))
		})
	}
}

// TestGetFileContentValidation calls NewService(nil) because validation runs before
// any Azure call — a nil blobClient never gets touched.
func TestGetFileContentValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.GetFileContent(context.Background(), "")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GetFileContent(context.Background(), "   ")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GetFileContent(context.Background(), "pmcs/some-file.json")
	require.ErrorIs(t, err, ErrInvalidBlobPath)

	_, err = svc.GetFileContent(context.Background(), "pmcs_sbs/some-file.pdf")
	require.ErrorIs(t, err, ErrInvalidFileType)

	// path.Clean turns "pmcs_sbs/../secret.json" into "secret.json", failing the prefix check.
	_, err = svc.GetFileContent(context.Background(), "pmcs_sbs/../secret.json")
	require.ErrorIs(t, err, ErrInvalidBlobPath)
}

func TestGetFilesValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.GetFiles(context.Background(), "")
	require.ErrorIs(t, err, ErrEmptyFolderName)

	_, err = svc.GetFiles(context.Background(), "../secret")
	require.ErrorIs(t, err, ErrInvalidBlobPath)

	_, err = svc.GetFiles(context.Background(), "hmmwv/subfolder")
	require.ErrorIs(t, err, ErrInvalidBlobPath)
}

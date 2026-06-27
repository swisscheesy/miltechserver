package pmcs_sbs

import (
	"context"
	"errors"
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

func TestBuildImageBlobPath(t *testing.T) {
	imageBlobPath, err := buildImageBlobPath(
		"pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json",
		"Before_12",
	)

	require.NoError(t, err)
	require.Equal(t, "pmcs_sbs/HMMWV/images/HMMWV NoArmor (SEPT13)/Before_12.png", imageBlobPath)
}

func TestBuildImageBlobPathGuidePathValidation(t *testing.T) {
	tests := []struct {
		name          string
		guideBlobPath string
		wantErr       error
	}{
		{"blank guide path", "   ", ErrEmptyBlobPath},
		{"non-json file", "pmcs_sbs/HMMWV/file.pdf", ErrInvalidFileType},
		{"path traversal", "pmcs_sbs/HMMWV/../secret.json", ErrInvalidBlobPath},
		{"windows separators", `pmcs_sbs\HMMWV\file.json`, ErrInvalidBlobPath},
		{"wrong prefix", "pmcs/other/file.json", ErrInvalidBlobPath},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildImageBlobPath(tc.guideBlobPath, "Before_12")

			require.Error(t, err)
			require.True(t, errors.Is(err, tc.wantErr), "expected %v, got %v", tc.wantErr, err)
		})
	}
}

func TestBuildImageBlobPathImageNameValidation(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		wantErr   error
	}{
		{"blank image name", "   ", ErrEmptyImageName},
		{"png extension", "Before_12.png", ErrInvalidImageName},
		{"forward slash", "folder/Before_12", ErrInvalidImageName},
		{"windows separator", `folder\Before_12`, ErrInvalidImageName},
		{"path traversal", "../Before_12", ErrInvalidImageName},
		{"dot in name", "Before.12", ErrInvalidImageName},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildImageBlobPath("pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json", tc.imageName)

			require.Error(t, err)
			require.True(t, errors.Is(err, tc.wantErr), "expected %v, got %v", tc.wantErr, err)
		})
	}
}

func TestGetImageValidationDoesNotRequireBlobClient(t *testing.T) {
	svc := NewService(nil)

	tests := []struct {
		name          string
		guideBlobPath string
		imageName     string
		wantErr       error
	}{
		{
			name:          "invalid guide path",
			guideBlobPath: "pmcs_sbs/HMMWV/../secret.json",
			imageName:     "Before_12",
			wantErr:       ErrInvalidBlobPath,
		},
		{
			name:          "invalid image name",
			guideBlobPath: "pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json",
			imageName:     "Before_12.png",
			wantErr:       ErrInvalidImageName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetImage(context.Background(), tc.guideBlobPath, tc.imageName)

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func (s *serviceStub) GetImage(_ context.Context, _ string, _ string) (*ImageDownload, error) {
	return nil, nil
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

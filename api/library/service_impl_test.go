package library

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatDisplayName(t *testing.T) {
	require.Equal(t, "M1151", formatDisplayName("m1151"))
	require.Equal(t, "M2 BRADLEY", formatDisplayName("m2-bradley"))
	require.Equal(t, "M2 BRADLEY", formatDisplayName("m2_bradley"))
}

func TestExtractFileName(t *testing.T) {
	require.Equal(t, "m1-abrams.pdf", extractFileName("pmcs/TRACK/m1-abrams.pdf"))
	require.Equal(t, "file.pdf", extractFileName("file.pdf"))
}

func TestExtractPMCSEquipmentName(t *testing.T) {
	name, ok := extractPMCSEquipmentName("pmcs/m1-abrams/manual.pdf")
	require.True(t, ok)
	require.Equal(t, "m1-abrams", name)

	_, ok = extractPMCSEquipmentName("bii/m1-abrams/manual.pdf")
	require.False(t, ok)

	_, ok = extractPMCSEquipmentName("pmcs/")
	require.False(t, ok)
}

func TestGenerateDownloadURLValidation(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.GenerateDownloadURL("")
	require.ErrorIs(t, err, ErrEmptyBlobPath)

	_, err = svc.GenerateDownloadURL("invalid/path.pdf")
	require.ErrorIs(t, err, ErrInvalidBlobPath)

	_, err = svc.GenerateDownloadURL("pmcs/vehicle/file.txt")
	require.ErrorIs(t, err, ErrInvalidFileType)
}

func TestGetPMCSDocumentsValidation(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.GetPMCSDocuments("")
	require.ErrorIs(t, err, ErrEmptyVehicleName)
}

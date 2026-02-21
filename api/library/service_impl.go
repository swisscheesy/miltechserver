package library

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"miltechserver/api/analytics"
	"miltechserver/api/library/shared"
	"miltechserver/bootstrap"
)

const (
	LibraryContainerName = "library"
	PMCSPrefix           = "pmcs/"
)

type ServiceImpl struct {
	blobClient *azblob.Client
	env        *bootstrap.Env
	analytics  analytics.Service
}

func NewService(
	blobClient *azblob.Client,
	env *bootstrap.Env,
	analyticsService analytics.Service,
) Service {
	return &ServiceImpl{
		blobClient: blobClient,
		env:        env,
		analytics:  analyticsService,
	}
}

// GetPMCSVehicles retrieves all vehicle folders from the PMCS library in Azure Blob Storage.
func (s *ServiceImpl) GetPMCSVehicles() (*PMCSVehiclesResponse, error) {
	ctx := context.Background()

	slog.Info("Fetching PMCS vehicles from Azure Blob Storage",
		"container", LibraryContainerName,
		"prefix", PMCSPrefix)

	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)
	prefix := PMCSPrefix
	pager := containerClient.NewListBlobsHierarchyPager(
		"/",
		&container.ListBlobsHierarchyOptions{
			Prefix: &prefix,
		},
	)

	vehicles := []VehicleFolderResponse{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS vehicles from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"prefix", PMCSPrefix)
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}

		for _, prefix := range page.Segment.BlobPrefixes {
			if prefix.Name == nil {
				continue
			}

			fullPath := *prefix.Name
			vehicleName := strings.TrimPrefix(fullPath, PMCSPrefix)
			vehicleName = strings.TrimSuffix(vehicleName, "/")
			if vehicleName == "" {
				continue
			}

			displayName := formatDisplayName(vehicleName)

			vehicles = append(vehicles, VehicleFolderResponse{
				Name:        vehicleName,
				FullPath:    fullPath,
				DisplayName: displayName,
			})
		}
	}

	slog.Info("Successfully fetched PMCS vehicles",
		"count", len(vehicles),
		"container", LibraryContainerName)

	return &PMCSVehiclesResponse{
		Vehicles: vehicles,
		Count:    len(vehicles),
	}, nil
}

// GetPMCSDocuments retrieves all PDF documents from a vehicle folder in Azure Blob Storage.
func (s *ServiceImpl) GetPMCSDocuments(vehicleName string) (*DocumentsListResponse, error) {
	ctx := context.Background()

	if strings.TrimSpace(vehicleName) == "" {
		return nil, ErrEmptyVehicleName
	}

	vehiclePrefix := fmt.Sprintf("%s%s/", PMCSPrefix, vehicleName)

	slog.Info("Fetching PMCS documents from Azure Blob Storage",
		"container", LibraryContainerName,
		"vehiclePrefix", vehiclePrefix,
		"vehicleName", vehicleName)

	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &vehiclePrefix,
	})

	documents := []DocumentResponse{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS documents from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"vehiclePrefix", vehiclePrefix)
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}

			blobPath := *blob.Name
			if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
				slog.Debug("Skipping non-PDF file", "blobPath", blobPath)
				continue
			}

			fileName := extractFileName(blobPath)

			var sizeBytes int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				sizeBytes = *blob.Properties.ContentLength
			}

			var lastModified string
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = blob.Properties.LastModified.Format(time.RFC3339)
			}

			documents = append(documents, DocumentResponse{
				Name:         fileName,
				BlobPath:     blobPath,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}

	slog.Info("Successfully fetched PMCS documents",
		"count", len(documents),
		"vehicleName", vehicleName,
		"container", LibraryContainerName)

	return &DocumentsListResponse{
		VehicleName: vehicleName,
		Documents:   documents,
		Count:       len(documents),
	}, nil
}

// formatDisplayName converts vehicle folder names to human-readable display names.
// Examples: "m1151" -> "M1151", "m2-bradley" -> "M2 BRADLEY", "m2_bradley" -> "M2 BRADLEY"
func formatDisplayName(name string) string {
	display := strings.ToUpper(name)
	display = strings.ReplaceAll(display, "-", " ")
	display = strings.ReplaceAll(display, "_", " ")
	return display
}

// GenerateDownloadURL creates a time-limited SAS URL for secure blob downloads.
// ctx should be the request context so Azure calls are cancelled if the client disconnects.
func (s *ServiceImpl) GenerateDownloadURL(ctx context.Context, blobPath string) (*DownloadURLResponse, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}

	// Sanitise the path to prevent directory traversal (e.g. "pmcs/../secret.pdf").
	blobPath = path.Clean(blobPath)

	if !strings.HasPrefix(blobPath, "pmcs/") && !strings.HasPrefix(blobPath, "bii/") {
		return nil, ErrInvalidBlobPath
	}

	if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
		return nil, ErrInvalidFileType
	}

	slog.Info("Generating download URL for blob",
		"container", LibraryContainerName,
		"blobPath", blobPath)

	// Verify the blob exists before signing a token for it.
	blobClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName).NewBlobClient(blobPath)
	if _, err := blobClient.GetProperties(ctx, nil); err != nil {
		slog.Error("Blob not found or not accessible",
			"error", err,
			"blobPath", blobPath)
		return nil, fmt.Errorf("%w: %v", ErrDocumentNotFound, err)
	}

	sasResult, err := shared.GenerateBlobSASURL(ctx, s.blobClient, LibraryContainerName, blobPath)
	if err != nil {
		slog.Error("Failed to generate SAS token",
			"error", err,
			"blobPath", blobPath)
		return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
	}

	slog.Info("Successfully generated download URL",
		"blobPath", blobPath,
		"expiresAt", sasResult.ExpiresAt.Format(time.RFC3339))

	if analyticsErr := s.trackPMCSDownload(blobPath); analyticsErr != nil {
		slog.Warn("Failed to increment analytics for PMCS download", "blobPath", blobPath, "error", analyticsErr)
	}

	return &DownloadURLResponse{
		BlobPath:    blobPath,
		DownloadURL: sasResult.URL,
		ExpiresAt:   sasResult.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// extractFileName returns the file name from a blob path.
// Example: "pmcs/TRACK/m1-abrams.pdf" -> "m1-abrams.pdf"
func extractFileName(blobPath string) string {
	parts := strings.Split(blobPath, "/")
	if len(parts) == 0 {
		return blobPath
	}
	return parts[len(parts)-1]
}

func (s *ServiceImpl) trackPMCSDownload(blobPath string) error {
	if s.analytics == nil {
		return nil
	}

	equipmentName, ok := extractPMCSEquipmentName(blobPath)
	if !ok {
		return nil
	}

	fileName := extractFileName(blobPath)
	if fileName == "" {
		return nil
	}
	baseName := strings.TrimSuffix(fileName, path.Ext(fileName))
	if strings.TrimSpace(baseName) == "" {
		return nil
	}

	displayName := formatDisplayName(equipmentName)
	if displayName == "" {
		displayName = baseName
	}

	return s.analytics.IncrementPMCSManualDownload(baseName, displayName)
}

func extractPMCSEquipmentName(blobPath string) (string, bool) {
	if !strings.HasPrefix(blobPath, PMCSPrefix) {
		return "", false
	}

	trimmed := strings.TrimPrefix(blobPath, PMCSPrefix)
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", false
	}

	equipmentName := strings.TrimSpace(parts[0])
	if equipmentName == "" {
		return "", false
	}

	return equipmentName, true
}

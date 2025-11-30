package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

const (
	LibraryContainerName = "library"
	PMCSPrefix           = "pmcs/"
)

type LibraryServiceImpl struct {
	blobClient *azblob.Client
	credential *azblob.SharedKeyCredential // Needed for SAS token generation
	env        *bootstrap.Env
}

func NewLibraryServiceImpl(
	blobClient *azblob.Client,
	credential *azblob.SharedKeyCredential,
	env *bootstrap.Env,
) LibraryService {
	return &LibraryServiceImpl{
		blobClient: blobClient,
		credential: credential,
		env:        env,
	}
}

// GetPMCSVehicles retrieves all vehicle folders from the PMCS library in Azure Blob Storage
func (s *LibraryServiceImpl) GetPMCSVehicles() (*response.PMCSVehiclesResponse, error) {
	ctx := context.Background()

	slog.Info("Fetching PMCS vehicles from Azure Blob Storage",
		"container", LibraryContainerName,
		"prefix", PMCSPrefix)

	// Get container client
	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)

	// Create prefix variable for options
	prefix := PMCSPrefix

	// Create pager with delimiter to get hierarchical listing
	pager := containerClient.NewListBlobsHierarchyPager(
		"/", // delimiter for folder simulation
		&container.ListBlobsHierarchyOptions{
			Prefix: &prefix,
		},
	)

	vehicles := []response.VehicleFolderResponse{}

	// Iterate through pages
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS vehicles from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"prefix", PMCSPrefix)
			return nil, fmt.Errorf("failed to list PMCS vehicles: %w", err)
		}

		// BlobPrefixes represent "folders"
		for _, prefix := range page.Segment.BlobPrefixes {
			if prefix.Name == nil {
				continue
			}

			fullPath := *prefix.Name
			// Extract vehicle name from path (e.g., "pmcs/m1151/" -> "m1151")
			vehicleName := strings.TrimPrefix(fullPath, PMCSPrefix)
			vehicleName = strings.TrimSuffix(vehicleName, "/")

			// Skip empty names
			if vehicleName == "" {
				continue
			}

			// Create display name (capitalize, replace hyphens and underscores with spaces)
			displayName := formatDisplayName(vehicleName)

			vehicles = append(vehicles, response.VehicleFolderResponse{
				Name:        vehicleName,
				FullPath:    fullPath,
				DisplayName: displayName,
			})
		}
	}

	slog.Info("Successfully fetched PMCS vehicles",
		"count", len(vehicles),
		"container", LibraryContainerName)

	return &response.PMCSVehiclesResponse{
		Vehicles: vehicles,
		Count:    len(vehicles),
	}, nil
}

// GetPMCSDocuments retrieves all PDF documents from a vehicle folder in Azure Blob Storage
func (s *LibraryServiceImpl) GetPMCSDocuments(vehicleName string) (*response.DocumentsListResponse, error) {
	ctx := context.Background()

	// Input validation
	if vehicleName == "" {
		return nil, fmt.Errorf("vehicle name cannot be empty")
	}

	// Construct the prefix for this vehicle's folder
	vehiclePrefix := fmt.Sprintf("%s%s/", PMCSPrefix, vehicleName)

	slog.Info("Fetching PMCS documents from Azure Blob Storage",
		"container", LibraryContainerName,
		"vehiclePrefix", vehiclePrefix,
		"vehicleName", vehicleName)

	// Get container client
	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)

	// Create pager for flat listing (actual files, not folders)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &vehiclePrefix,
	})

	documents := []response.DocumentResponse{}

	// Iterate through pages
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS documents from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"vehiclePrefix", vehiclePrefix)
			return nil, fmt.Errorf("failed to list PMCS documents: %w", err)
		}

		// BlobItems represent actual files
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}

			blobPath := *blob.Name

			// Filter: Only include PDF files (case-insensitive)
			if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
				slog.Debug("Skipping non-PDF file", "blobPath", blobPath)
				continue
			}

			// Extract file name from path (e.g., "pmcs/TRACK/m1-abrams.pdf" -> "m1-abrams.pdf")
			fileName := extractFileName(blobPath)

			// Extract metadata
			var sizeBytes int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				sizeBytes = *blob.Properties.ContentLength
			}

			var lastModified string
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = blob.Properties.LastModified.Format(time.RFC3339)
			}

			documents = append(documents, response.DocumentResponse{
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

	return &response.DocumentsListResponse{
		VehicleName: vehicleName,
		Documents:   documents,
		Count:       len(documents),
	}, nil
}

// formatDisplayName converts vehicle folder names to human-readable display names
// Examples: "m1151" -> "M1151", "m2-bradley" -> "M2 BRADLEY", "m2_bradley" -> "M2 BRADLEY"
func formatDisplayName(name string) string {
	display := strings.ToUpper(name)
	display = strings.ReplaceAll(display, "-", " ")
	display = strings.ReplaceAll(display, "_", " ")
	return display
}

// GenerateDownloadURL creates a time-limited SAS URL for secure blob downloads
func (s *LibraryServiceImpl) GenerateDownloadURL(blobPath string) (*response.DownloadURLResponse, error) {
	ctx := context.Background()

	// Input validation
	if blobPath == "" {
		return nil, fmt.Errorf("blob path cannot be empty")
	}

	// Validate blob path format (should be within library container)
	if !strings.HasPrefix(blobPath, "pmcs/") && !strings.HasPrefix(blobPath, "bii/") {
		return nil, fmt.Errorf("invalid blob path: must start with pmcs/ or bii/")
	}

	// Validate file extension (only PDFs allowed)
	if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
		return nil, fmt.Errorf("invalid file type: only PDF files can be downloaded")
	}

	slog.Info("Generating download URL for blob",
		"container", LibraryContainerName,
		"blobPath", blobPath)

	// Get blob client for the specific file
	blobClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName).NewBlobClient(blobPath)

	// Check if blob exists before generating SAS
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		slog.Error("Blob not found or not accessible",
			"error", err,
			"blobPath", blobPath)
		return nil, fmt.Errorf("document not found: %w", err)
	}

	// Set SAS token expiry time (1 hour from now)
	expiryTime := time.Now().UTC().Add(1 * time.Hour)

	// Create SAS permissions (read-only)
	permissions := sas.BlobPermissions{
		Read: true,
	}

	// Create SAS signature values
	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,                          // HTTPS only for security
		StartTime:     time.Now().UTC().Add(-5 * time.Minute),     // Start 5 min ago to handle clock skew
		ExpiryTime:    expiryTime,
		Permissions:   permissions.String(),
		ContainerName: LibraryContainerName,
		BlobName:      blobPath,
	}.SignWithSharedKey(s.credential)

	if err != nil {
		slog.Error("Failed to generate SAS token",
			"error", err,
			"blobPath", blobPath)
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	// Construct full download URL with SAS token
	downloadURL := fmt.Sprintf("%s?%s", blobClient.URL(), sasQueryParams.Encode())

	slog.Info("Successfully generated download URL",
		"blobPath", blobPath,
		"expiresAt", expiryTime.Format(time.RFC3339))

	return &response.DownloadURLResponse{
		BlobPath:    blobPath,
		DownloadURL: downloadURL,
		ExpiresAt:   expiryTime.Format(time.RFC3339),
	}, nil
}

// extractFileName returns the file name from a blob path
// Example: "pmcs/TRACK/m1-abrams.pdf" -> "m1-abrams.pdf"
func extractFileName(blobPath string) string {
	parts := strings.Split(blobPath, "/")
	if len(parts) == 0 {
		return blobPath
	}
	return parts[len(parts)-1]
}

package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

const (
	LibraryContainerName = "library"
	PMCSPrefix           = "pmcs/"
)

type LibraryServiceImpl struct {
	blobClient *azblob.Client
	env        *bootstrap.Env
}

func NewLibraryServiceImpl(
	blobClient *azblob.Client,
	env *bootstrap.Env,
) LibraryService {
	return &LibraryServiceImpl{
		blobClient: blobClient,
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

// formatDisplayName converts vehicle folder names to human-readable display names
// Examples: "m1151" -> "M1151", "m2-bradley" -> "M2 BRADLEY", "m2_bradley" -> "M2 BRADLEY"
func formatDisplayName(name string) string {
	display := strings.ToUpper(name)
	display = strings.ReplaceAll(display, "-", " ")
	display = strings.ReplaceAll(display, "_", " ")
	return display
}

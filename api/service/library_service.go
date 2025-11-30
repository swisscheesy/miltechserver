package service

import (
	"miltechserver/api/response"
)

// LibraryService provides methods for accessing PMCS and BII library documents
type LibraryService interface {
	// GetPMCSVehicles returns a list of all vehicle folders in the PMCS library
	GetPMCSVehicles() (*response.PMCSVehiclesResponse, error)

	// GetPMCSDocuments returns all PDF documents for a specific vehicle folder
	// Returns empty array if vehicle folder has no PDFs or doesn't exist
	// Returns error only if Azure Blob Storage operation fails
	GetPMCSDocuments(vehicleName string) (*response.DocumentsListResponse, error)

	// GenerateDownloadURL creates a time-limited SAS URL for downloading a blob
	// blobPath: Full blob path (e.g., "pmcs/TRACK/m1-abrams.pdf")
	// Returns SAS URL valid for 1 hour with read-only permission
	// Returns error if blob doesn't exist or SAS generation fails
	GenerateDownloadURL(blobPath string) (*response.DownloadURLResponse, error)

	// Future endpoints:
	// GetBIICategories() (*response.PMCSVehiclesResponse, error)
	// GetBIIDocuments(category string) (*response.DocumentsListResponse, error)
}

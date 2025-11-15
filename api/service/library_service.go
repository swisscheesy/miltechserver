package service

import (
	"miltechserver/api/response"
)

// LibraryService provides methods for accessing PMCS and BII library documents
type LibraryService interface {
	// GetPMCSVehicles returns a list of all vehicle folders in the PMCS library
	GetPMCSVehicles() (*response.PMCSVehiclesResponse, error)

	// Future endpoints:
	// GetPMCSDocuments(vehicleName string) (*response.DocumentsListResponse, error)
	// GetBIICategories() (*response.PMCSVehiclesResponse, error)
	// GetBIIDocuments(category string) (*response.DocumentsListResponse, error)
	// GenerateDownloadURL(blobPath string) (string, error)
}

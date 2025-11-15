package response

// VehicleFolderResponse represents a vehicle folder in the PMCS library
type VehicleFolderResponse struct {
	Name        string `json:"name"`          // Vehicle name (folder name without prefix)
	FullPath    string `json:"full_path"`     // Full blob prefix (e.g., "pmcs/m1151/")
	DisplayName string `json:"display_name"`  // Human-readable name (e.g., "M1151")
}

// PMCSVehiclesResponse is the response for listing available PMCS vehicles
type PMCSVehiclesResponse struct {
	Vehicles []VehicleFolderResponse `json:"vehicles"`
	Count    int                     `json:"count"`
}

// DocumentResponse represents a document file in the library
// Future: Used for listing documents within a vehicle folder
type DocumentResponse struct {
	Name         string `json:"name"`           // File name
	BlobPath     string `json:"blob_path"`      // Full blob path
	SizeBytes    int64  `json:"size_bytes"`     // File size
	LastModified string `json:"last_modified"`  // ISO 8601 timestamp
	DownloadURL  string `json:"download_url"`   // Temporary download URL
}

// DocumentsListResponse is the response for listing documents in a category
// Future: Used for both PMCS vehicle documents and BII category documents
type DocumentsListResponse struct {
	VehicleName string             `json:"vehicle_name"`
	Documents   []DocumentResponse `json:"documents"`
	Count       int                `json:"count"`
}

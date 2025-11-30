package response

// VehicleFolderResponse represents a vehicle folder in the PMCS library
type VehicleFolderResponse struct {
	Name        string `json:"name"`         // Vehicle name (folder name without prefix)
	FullPath    string `json:"full_path"`    // Full blob prefix (e.g., "pmcs/m1151/")
	DisplayName string `json:"display_name"` // Human-readable name (e.g., "M1151")
}

// PMCSVehiclesResponse is the response for listing available PMCS vehicles
type PMCSVehiclesResponse struct {
	Vehicles []VehicleFolderResponse `json:"vehicles"`
	Count    int                     `json:"count"`
}

// DocumentResponse represents a document file in the library
type DocumentResponse struct {
	Name         string `json:"name"`          // File name (e.g., "m1-abrams-pmcs.pdf")
	BlobPath     string `json:"blob_path"`     // Full blob path (e.g., "pmcs/TRACK/m1-abrams-pmcs.pdf")
	SizeBytes    int64  `json:"size_bytes"`    // File size in bytes
	LastModified string `json:"last_modified"` // ISO 8601 timestamp
}

// DocumentsListResponse is the response for listing documents in a vehicle folder
type DocumentsListResponse struct {
	VehicleName string             `json:"vehicle_name"` // Vehicle name from URL parameter
	Documents   []DocumentResponse `json:"documents"`    // List of PDF files in the folder
	Count       int                `json:"count"`        // Number of documents found
}

// DownloadURLResponse contains a time-limited download URL for a document
type DownloadURLResponse struct {
	BlobPath    string `json:"blob_path"`    // Original blob path requested (e.g., "pmcs/TRACK/m1-abrams.pdf")
	DownloadURL string `json:"download_url"` // Time-limited SAS URL valid for 1 hour
	ExpiresAt   string `json:"expires_at"`   // ISO 8601 timestamp when URL expires
}

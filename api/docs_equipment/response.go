package docs_equipment

import "miltechserver/.gen/miltech_ng/public/model"

// EquipmentDetailsPageResponse — paginated equipment list (matches EIC pattern).
type EquipmentDetailsPageResponse struct {
	Items      []model.DocsEquipmentDetails `json:"items"`
	Count      int                          `json:"count"`
	Page       int                          `json:"page"`
	TotalPages int                          `json:"total_pages"`
	IsLastPage bool                         `json:"is_last_page"`
}

// FamiliesResponse — unique family values from the DB.
type FamiliesResponse struct {
	Families []string `json:"families"`
	Count    int      `json:"count"`
}

// ImageFamilyFolder — a family folder in blob storage.
type ImageFamilyFolder struct {
	Name        string `json:"name"`
	FullPath    string `json:"full_path"`
	DisplayName string `json:"display_name"`
}

// ImageFamiliesResponse — list of image family folders.
type ImageFamiliesResponse struct {
	Families []ImageFamilyFolder `json:"families"`
	Count    int                 `json:"count"`
}

// ImageItem — a single image blob.
type ImageItem struct {
	Name         string `json:"name"`
	BlobPath     string `json:"blob_path"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
}

// FamilyImagesResponse — images in a family folder.
type FamilyImagesResponse struct {
	Family string      `json:"family"`
	Images []ImageItem `json:"images"`
	Count  int         `json:"count"`
}

// ImageDownloadResponse — SAS download URL.
type ImageDownloadResponse struct {
	BlobPath    string `json:"blob_path"`
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
}

// ImageURLItem — a single image with its SAS download URL.
type ImageURLItem struct {
	Name         string `json:"name"`
	BlobPath     string `json:"blob_path"`
	DownloadURL  string `json:"download_url"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
}

// FamilyImageURLsResponse — all images in a family with pre-generated SAS URLs.
type FamilyImageURLsResponse struct {
	Family    string         `json:"family"`
	Images    []ImageURLItem `json:"images"`
	Count     int            `json:"count"`
	ExpiresAt string         `json:"expires_at"`
}

package pmcs_sbs

import "io"

// FolderResponse represents a top-level folder in the PMCS SBS library.
type FolderResponse struct {
	Name        string `json:"name"`
	FullPath    string `json:"full_path"`
	DisplayName string `json:"display_name"`
}

// FoldersListResponse is the response for listing available PMCS SBS folders.
type FoldersListResponse struct {
	Folders []FolderResponse `json:"folders"`
	Count   int              `json:"count"`
}

// FileResponse represents a JSON file in a PMCS SBS folder.
type FileResponse struct {
	Name         string `json:"name"`
	BlobPath     string `json:"blob_path"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
}

// FilesListResponse is the response for listing files in a PMCS SBS folder.
type FilesListResponse struct {
	FolderName string         `json:"folder_name"`
	Files      []FileResponse `json:"files"`
	Count      int            `json:"count"`
}

type ImageDownload struct {
	Body          io.ReadCloser
	ContentLength int64
	ContentType   string
	FileName      string
	BlobPath      string
}

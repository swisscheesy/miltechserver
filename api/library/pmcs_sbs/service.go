package pmcs_sbs

import (
	"context"
	"encoding/json"
)

// Service provides access to PMCS Step-by-Step JSON files in Azure Blob Storage.
type Service interface {
	// GetFolders returns all top-level folders under pmcs_sbs/.
	GetFolders(ctx context.Context) (*FoldersListResponse, error)

	// GetFiles returns all JSON files within the given folder.
	// Returns an empty slice (not an error) if the folder has no JSON files.
	GetFiles(ctx context.Context, folderName string) (*FilesListResponse, error)

	// GetFileContent fetches a JSON blob and returns its raw content.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	GetFileContent(ctx context.Context, blobPath string) (json.RawMessage, error)

	// GetImage fetches a PMCS SBS guide item image.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	GetImage(ctx context.Context, guideBlobPath string, imageName string) (*ImageDownload, error)
}

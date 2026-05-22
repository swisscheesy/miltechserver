package pmcs_sbs

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// implementation is the concrete implementation of the Service interface.
type implementation struct {
	blobClient *azblob.Client
}

// NewService creates a new PMCS SBS service with Azure Blob Storage client.
func NewService(blobClient *azblob.Client) Service {
	return &implementation{
		blobClient: blobClient,
	}
}

// GetFolders returns all top-level folders under pmcs_sbs/.
func (i *implementation) GetFolders(ctx context.Context) (*FoldersListResponse, error) {
	// TODO: Implement blob listing
	_ = ctx
	return nil, ErrBlobListFailed
}

// GetFiles returns all JSON files within the given folder.
func (i *implementation) GetFiles(ctx context.Context, folderName string) (*FilesListResponse, error) {
	// TODO: Implement blob listing with folder filter
	_ = ctx
	_ = folderName
	return nil, ErrBlobListFailed
}

// GetFileContent fetches a JSON blob and returns its raw content.
func (i *implementation) GetFileContent(ctx context.Context, blobPath string) (json.RawMessage, error) {
	// TODO: Implement blob reading
	_ = ctx
	_ = blobPath
	return nil, ErrBlobReadFailed
}

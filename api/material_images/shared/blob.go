package shared

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

const ContainerName = "material-images"

// BlobStorage provides utilities for Azure Blob Storage operations.
type BlobStorage struct {
	client *azblob.Client
}

func NewBlobStorage(client *azblob.Client) *BlobStorage {
	return &BlobStorage{client: client}
}

// Upload stores image data and returns the blob name.
func (b *BlobStorage) Upload(blobName string, imageData []byte) error {
	if b.client == nil {
		return nil
	}

	ctx := context.Background()

	_, err := b.client.UploadBuffer(ctx, ContainerName, blobName, imageData, nil)
	if err != nil {
		return fmt.Errorf("failed to upload to blob storage: %w", err)
	}

	return nil
}

// Delete removes an image from blob storage.
func (b *BlobStorage) Delete(blobName string) error {
	if b.client == nil {
		return nil
	}

	ctx := context.Background()
	_, err := b.client.DeleteBlob(ctx, ContainerName, blobName, nil)
	return err
}

// Download retrieves the blob data.
func (b *BlobStorage) Download(blobName string) ([]byte, error) {
	if b.client == nil {
		return []byte{}, nil
	}

	ctx := context.Background()
	response, err := b.client.DownloadStream(ctx, ContainerName, blobName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download image from blob storage: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return data, nil
}

// GetURL returns the full URL for a blob.
func (b *BlobStorage) GetURL(blobName string, accountName string) string {
	if blobName == "" {
		return ""
	}

	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", accountName, ContainerName, blobName)
}

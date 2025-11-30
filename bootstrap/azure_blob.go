package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func NewAzureBlobClient(env *Env) (*azblob.Client, *azblob.SharedKeyCredential) {
	slog.Info("Creating Azure Blob Client")

	credential, err := azblob.NewSharedKeyCredential(env.BlobAccountName, env.BlobAccountKey)
	if err != nil {
		slog.Error("Error creating Blob credential", "error", err)
		panic(err)
	}

	accountUrl := fmt.Sprintf("https://%s.blob.core.windows.net", env.BlobAccountName)
	blobClient, err := azblob.NewClientWithSharedKeyCredential(accountUrl, credential, nil)
	if err != nil {
		slog.Error("Error creating Blob Client", "error", err)
		panic(err)
	}

	return blobClient, credential
}

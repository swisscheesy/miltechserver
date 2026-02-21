package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// NewAzureBlobClient creates an Azure Blob Storage client authenticated via
// DefaultAzureCredential (Managed Identity in production, Azure CLI locally).
// The server must have the Storage Blob Delegator role to generate User Delegation SAS tokens.
func NewAzureBlobClient(env *Env) *azblob.Client {
	slog.Info("Creating Azure Blob client using DefaultAzureCredential")

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		slog.Error("Failed to create Azure credential", "error", err)
		panic(err)
	}

	accountURL := fmt.Sprintf("https://%s.blob.core.windows.net", env.BlobAccountName)
	blobClient, err := azblob.NewClient(accountURL, credential, nil)
	if err != nil {
		slog.Error("Failed to create Azure Blob client", "error", err)
		panic(err)
	}

	return blobClient
}

// Package shared provides common utilities for the library feature packages.
package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// SASResult holds a generated SAS URL and its expiry time.
type SASResult struct {
	URL       string
	ExpiresAt time.Time
}

// strPtr is a convenience helper to get a *string from a string literal.
func strPtr(s string) *string { return &s }

// GenerateBlobSASURL creates a 1-hour, read-only, HTTPS-only User Delegation SAS URL
// for a specific blob. The server must have a Managed Identity with the
// Storage Blob Delegator role assigned on the storage account.
//
// SAS parameters:
//   - No StartTime set — valid immediately (avoids clock-skew per Azure best practices).
//   - ExpiryTime: 1 hour from now.
//   - Permissions: read-only (sp=r).
//   - Protocol: HTTPS only (spr=https).
//   - Scope: blob-level (sr=b), not container-level.
//
// The ctx parameter should be the request context so the Azure credential call
// is cancelled if the client disconnects.
func GenerateBlobSASURL(
	ctx context.Context,
	blobClient *azblob.Client,
	containerName string,
	blobPath string,
) (*SASResult, error) {
	expiresAt := time.Now().UTC().Add(1 * time.Hour)

	// Obtain a User Delegation Key from Azure AD via the server's Managed Identity.
	// The key validity window starts 15 minutes ago (clock-skew buffer) and covers the SAS expiry.
	svcClient := blobClient.ServiceClient()
	udk, err := svcClient.GetUserDelegationCredential(
		ctx,
		service.KeyInfo{
			Start:  strPtr(time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339)),
			Expiry: strPtr(expiresAt.Format(time.RFC3339)),
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user delegation credential: %w", err)
	}

	permissions := sas.BlobPermissions{Read: true}
	bc := svcClient.NewContainerClient(containerName).NewBlobClient(blobPath)

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		ExpiryTime:    expiresAt,
		Permissions:   permissions.String(),
		ContainerName: containerName,
		BlobName:      blobPath,
	}.SignWithUserDelegation(udk)
	if err != nil {
		return nil, fmt.Errorf("failed to sign SAS token: %w", err)
	}

	downloadURL := fmt.Sprintf("%s?%s", bc.URL(), sasQueryParams.Encode())

	return &SASResult{
		URL:       downloadURL,
		ExpiresAt: expiresAt,
	}, nil
}

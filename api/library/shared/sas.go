// Package shared provides common utilities for the library feature packages.
package shared

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// udkEntry caches a User Delegation Key and its expiry.
// The key is valid for up to 45 minutes and shared across all callers.
type udkEntry struct {
	mu        sync.Mutex
	key       *service.UserDelegationCredential
	expiresAt time.Time
}

// packageUDKCache is the module-level UDK cache shared across all callers.
var packageUDKCache udkEntry

// SASResult holds a generated SAS URL and its expiry time.
type SASResult struct {
	URL       string
	ExpiresAt time.Time
}

// strPtr is a convenience helper to get a *string from a string literal.
func strPtr(s string) *string { return &s }

// getOrRefreshUDK returns a cached User Delegation Key if still valid, or fetches
// a new one from Azure AD and caches it for 45 minutes.
// expiresAt is the expiry of the SAS token being signed — the UDK must cover it.
func getOrRefreshUDK(ctx context.Context, svcClient *service.Client, expiresAt time.Time) (*service.UserDelegationCredential, error) {
	packageUDKCache.mu.Lock()
	defer packageUDKCache.mu.Unlock()

	// Reuse the cached key if it covers the requested expiry with 5 minutes of margin.
	if packageUDKCache.key != nil && packageUDKCache.expiresAt.After(expiresAt.Add(5*time.Minute)) {
		return packageUDKCache.key, nil
	}

	// Request a key valid for 45 minutes from now.
	keyExpiry := time.Now().UTC().Add(45 * time.Minute)
	udk, err := svcClient.GetUserDelegationCredential(
		ctx,
		service.KeyInfo{
			Start:  strPtr(time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339)),
			Expiry: strPtr(keyExpiry.Format(time.RFC3339)),
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user delegation credential: %w", err)
	}

	packageUDKCache.key = udk
	packageUDKCache.expiresAt = keyExpiry
	return udk, nil
}

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

	svcClient := blobClient.ServiceClient()
	udk, err := getOrRefreshUDK(ctx, svcClient, expiresAt)
	if err != nil {
		return nil, err
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

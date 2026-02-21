# Azure Blob Storage: File Download Link Analysis

**Author:** Antigravity (AI Code Analysis)  
**Date:** 2026-02-20  
**Scope:** `bootstrap/azure_blob.go`, `api/library/service_impl.go`, `api/library/ps_mag/service_impl.go`, `api/material_images/shared/blob.go`

---

## 1. Executive Summary

The library feature (`pmcs` documents and PS Magazine issues) uses **Service SAS (Shared Access Signature) URLs** to generate time-limited, read-only download links for blobs stored in a private Azure Blob Storage container. This is the **correct and standard approach** for sharing private blobs with unauthenticated clients. The implementation is functionally sound and follows most Azure best practices. Several improvements are recommended, primarily around upgrading to **User Delegation SAS**, hardening the public endpoint with **rate limiting**, and addressing a minor **clock skew** configuration detail.

The `material_images` package uses a **plain direct URL** with no SAS token — this only works if the container has public read access, which may have been intentional but warrants explicit documentation.

---

## 2. Architecture Overview

```
Client App
    │
    │ GET /api/v1/library/download?blob_path=pmcs/TRACK/m1-abrams.pdf
    ▼
Go API Server (Gin)
    │
    │ 1. Validate blob_path prefix (pmcs/ or bii/) and .pdf extension
    │ 2. blobClient.GetProperties() — verify blob exists
    │ 3. Sign BlobSignatureValues with SharedKeyCredential
    │ 4. Return SAS URL (expires in 1 hour)
    ▼
Azure Blob Storage (private container: "library")
    │
    │ Client uses SAS URL directly to download the PDF
    ▼
Client App (PDF download)
```

### Files Involved

| File | Role |
|---|---|
| `bootstrap/azure_blob.go` | Creates `azblob.Client` and `SharedKeyCredential` using account name + key |
| `api/library/service_impl.go` | Generates SAS URL for `pmcs/` and `bii/` documents |
| `api/library/ps_mag/service_impl.go` | Generates SAS URL for `ps-mag/` magazine issues |
| `api/material_images/shared/blob.go` | Uploads/downloads/deletes images; `GetURL()` returns a plain URL (no SAS) |
| `api/library/route.go` | Public (unauthenticated) HTTP endpoint wiring |

---

## 3. Is the Current Approach Correct?

**Yes — SAS tokens are the correct mechanism for sharing private Azure Blob Storage files.** Azure Storage containers are private by default; direct URLs without a SAS token return `403 Unauthorized`. The SAS approach grants time-limited, read-only access on a per-blob basis without exposing the storage account key or changing the container's access level.

### How the SAS URL Works

A generated URL looks like:

```
https://<account>.blob.core.windows.net/library/pmcs/TRACK/m1-abrams.pdf
  ?sv=2024-08-04
  &se=2026-02-20T13%3A00%3A00Z  ← expires in 1 hour
  &sr=b                          ← scope: blob (not container)
  &sp=r                          ← permissions: read-only
  &spr=https                     ← HTTPS only
  &sig=<HMAC-SHA256 signature>
```

Azure Storage validates the signature server-side. The client cannot forge or extend the URL without access to the storage account key.

### Azure Documentation Endorsement

From the [Azure SAS Overview](https://learn.microsoft.com/en-us/azure/storage/common/storage-sas-overview):

> "A lightweight service authenticates the client as needed and then generates a SAS. Once the client application receives the SAS, it can access storage account resources directly."

This is exactly the pattern implemented here — the Go server is the SAS provider and the client downloads directly from Azure, avoiding unnecessary data proxying through the server.

---

## 4. Implementation Analysis

### 4.1 What's Done Well ✅

**Correct SAS type and scope**
```go
// Blob-scoped (sr=b), not container-scoped — least privilege
sasQueryParams, err := sas.BlobSignatureValues{
    Protocol:      sas.ProtocolHTTPS,   // HTTPS-only — prevents interception
    StartTime:     time.Now().UTC().Add(-5 * time.Minute), // clock skew buffer
    ExpiryTime:    expiryTime,           // 1-hour TTL
    Permissions:   permissions.String(), // Read-only
    ContainerName: LibraryContainerName,
    BlobName:      blobPath,            // Scoped to specific blob
}.SignWithSharedKey(s.credential)
```

- **Blob-scoped** (`sr=b`): more restrictive than container-scoped; the token can only access the specific file requested.
- **HTTPS-only** (`spr=https`): prevents token interception over plaintext HTTP.
- **Read-only** (`sp=r`): no write, delete, or list permissions.
- **1-hour expiry**: short-lived tokens minimize exposure window if leaked.
- **Existence check before signing**: `blobClient.GetProperties()` is called first, returning `404` for missing blobs rather than issuing a valid SAS for a non-existent file.

**Proper error handling and structured logging**
```go
if err != nil {
    slog.Error("Failed to generate SAS token", "error", err, "blobPath", blobPath)
    return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
}
```
Errors are wrapped for `errors.Is()` compatibility and logged with `slog` (Go 1.21+ structured logging).

**Input validation**
```go
if !strings.HasPrefix(blobPath, "pmcs/") && !strings.HasPrefix(blobPath, "bii/") {
    return nil, ErrInvalidBlobPath
}
if !strings.HasSuffix(strings.ToLower(blobPath), ".pdf") {
    return nil, ErrInvalidFileType
}
```
Enforces a strict allowlist on path prefixes and file type, preventing access to arbitrary blobs.

**ExpiresAt returned to client**
The response includes `expires_at` in ISO 8601 format, allowing the client to cache the SAS URL and refresh proactively before expiry.

**Analytics tracking**
Download events are tracked via `trackPMCSDownload()` which calls the analytics service — a good practice for auditing download usage.

---

### 4.2 Issues and Recommendations

#### 🔴 HIGH: Use User Delegation SAS Instead of Account Key SAS

**Current approach:** A Service SAS signed with the **storage account key** (Shared Key).

**Problem:** The storage account key provides root-level access to the entire storage account. Storing it in environment variables creates risk: if the account key is rotated, the app breaks; if it leaks, an attacker has full control over all blobs in the account.

**Azure's recommendation** (from official docs):
> "Microsoft recommends using a user delegation SAS when possible. A user delegation SAS is secured with Microsoft Entra credentials, so that you do not need to store your account key with your code."

**Recommended approach:** Use a **User Delegation SAS** via Azure Managed Identity (MSI):

```go
// No account key needed — use Managed Identity or Workload Identity
credential, err := azidentity.NewDefaultAzureCredential(nil)
client, err := azblob.NewClient(accountURL, credential, nil)

// Get user delegation key (valid up to 7 days)
userDelegationKey, err := client.ServiceClient().GetUserDelegationCredential(
    ctx,
    time.Now().UTC().Add(-5*time.Minute),
    time.Now().UTC().Add(7*24*time.Hour),
    nil,
)

// Sign SAS with user delegation key instead of shared key
sasQueryParams, err := sas.BlobSignatureValues{
    Protocol:    sas.ProtocolHTTPS,
    StartTime:   time.Now().UTC().Add(-5 * time.Minute),
    ExpiryTime:  time.Now().UTC().Add(1 * time.Hour),
    Permissions: sas.BlobPermissions{Read: true}.String(),
    ContainerName: containerName,
    BlobName:    blobPath,
}.SignWithUserDelegation(userDelegationKey)
```

This eliminates account key storage entirely and allows revocation via Azure RBAC.

> **Note:** This requires assigning the Managed Identity a role like `Storage Blob Data Reader` on the container.

---

#### 🟡 MEDIUM: Clock Skew Buffer is Below Microsoft's Recommendation

**Current code:**
```go
StartTime: time.Now().UTC().Add(-5 * time.Minute),
```

**Azure documentation recommends:**
> "In general, set the start time to be at least 15 minutes in the past. Or, don't set it at all, which will make it valid immediately in all cases. The same generally applies to expiry time—remember that you may observe up to 15 minutes of clock skew."

**Recommendation:** Increase the backward clock skew from `-5` to `-15` minutes, or omit `StartTime` entirely (it defaults to "immediate"):

```go
// Option A: omit StartTime (simplest and correct)
sasQueryParams, err := sas.BlobSignatureValues{
    Protocol:      sas.ProtocolHTTPS,
    ExpiryTime:    time.Now().UTC().Add(1 * time.Hour),
    Permissions:   permissions.String(),
    ContainerName: LibraryContainerName,
    BlobName:      blobPath,
}.SignWithSharedKey(s.credential)

// Option B: use 15-minute buffer
StartTime: time.Now().UTC().Add(-15 * time.Minute),
```

---

#### 🟡 MEDIUM: No Rate Limiting on Public Download Endpoint

**Route registration:**
```go
// In api/library/route.go — intentionally public, no auth required
publicGroup.GET("/library/download", handler.generateDownloadURL)
```

The download endpoint is **intentionally public** — no authentication is required for users to generate download links. The path validation (`pmcs/` or `bii/` prefix, `.pdf` only) restricts access to library documents only.

**Remaining risk:** Without rate limiting, an attacker could flood the endpoint, each request triggering an Azure `GetProperties` call (network I/O) and SAS signing. This is a cost and availability concern.

**Recommendation:** Add per-IP rate limiting via a Gin middleware:

```go
import "golang.org/x/time/rate"
// or use a Gin-native middleware like github.com/ulule/limiter
```

A stored access policy on the container can also enable bulk SAS revocation without rotating the account key, which is a useful incident-response capability even for public endpoints.

---

#### 🟡 MEDIUM: `context.Background()` Used Throughout (No Request Cancellation)

**Current code:**
```go
ctx := context.Background()
blobClient.GetProperties(ctx, nil)
```

`context.Background()` never cancels. If a client disconnects mid-request, the server continues the GetProperties call to Azure (wasting network I/O and a goroutine).

**Recommendation:** Pass the Gin request context to all Azure calls:

```go
func (s *ServiceImpl) GenerateDownloadURL(ctx context.Context, blobPath string) (*DownloadURLResponse, error) {
    _, err := blobClient.GetProperties(ctx, nil)
    // ...
}

// In route handler:
downloadURLResp, err := handler.service.GenerateDownloadURL(c.Request.Context(), blobPath)
```

This ensures Azure calls are cancelled when the HTTP request is cancelled or times out.

---

#### 🟢 LOW: Path Traversal Hardening

The `strings.HasPrefix` validations are good, but a crafted path like `pmcs/../secret.pdf` could theoretically pass prefix validation before path normalization. The current Azure SDK client likely handles this, but explicit normalization is defensive:

```go
import "path"

blobPath = path.Clean(blobPath)
if !strings.HasPrefix(blobPath, "pmcs/") && !strings.HasPrefix(blobPath, "bii/") {
    return nil, ErrInvalidBlobPath
}
```

`path.Clean` resolves `..` components and double slashes.

---

#### ✅ `material_images` Direct URL (Intentional Design)

In `api/material_images/shared/blob.go`:
```go
func (b *BlobStorage) GetURL(blobName string, accountName string) string {
    return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", accountName, ContainerName, blobName)
}
```

The `material-images` container is configured with **public anonymous read access** in Azure — direct URLs work by design. Material images are publicly accessible to all users and do not require a SAS token. No action needed.

---

#### 🟢 LOW: Duplicated SAS Logic Between `library` and `ps_mag`

The `GenerateDownloadURL` implementation in `service_impl.go` and `ps_mag/service_impl.go` are nearly identical. The only differences are the container prefix validation and the log message prefix. This creates a maintenance burden — if SAS parameters need to change (e.g., expiry time, clock skew buffer), both files must be updated.

**Recommendation:** Extract common SAS generation into a shared utility:

```go
// api/library/shared/sas.go
package shared

import (
    "time"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

type SASConfig struct {
    ContainerName string
    BlobPath      string
    Expiry        time.Duration
}

func GenerateBlobSASURL(blobClient *azblob.Client, credential *azblob.SharedKeyCredential, cfg SASConfig) (url string, expiresAt time.Time, err error) {
    expiresAt = time.Now().UTC().Add(cfg.Expiry)
    permissions := sas.BlobPermissions{Read: true}
    bc := blobClient.ServiceClient().NewContainerClient(cfg.ContainerName).NewBlobClient(cfg.BlobPath)

    sasParams, err := sas.BlobSignatureValues{
        Protocol:      sas.ProtocolHTTPS,
        ExpiryTime:    expiresAt,
        Permissions:   permissions.String(),
        ContainerName: cfg.ContainerName,
        BlobName:      cfg.BlobPath,
    }.SignWithSharedKey(credential)
    if err != nil {
        return "", time.Time{}, err
    }

    return fmt.Sprintf("%s?%s", bc.URL(), sasParams.Encode()), expiresAt, nil
}
```

---

## 5. Summary Table

| Finding | Severity | Status |
|---|---|---|
| SAS is the correct mechanism for private blob sharing | ✅ Correct | No action needed |
| Blob-scoped, read-only, HTTPS-only SAS | ✅ Correct | No action needed |
| 1-hour TTL on SAS tokens | ✅ Good | No action needed |
| Blob existence check before SAS generation | ✅ Good | No action needed |
| Structured logging with slog | ✅ Good | No action needed |
| Public download endpoints (intentional design) | ✅ Designed | No action needed |
| `material_images` public anonymous read access (intentional design) | ✅ Designed | No action needed |
| Using account key SharedKey SAS instead of User Delegation SAS | 🔴 High | Migrate to MSI + User Delegation SAS |
| Clock skew buffer too small (-5min vs. recommended -15min) | 🟡 Medium | Increase to -15min or omit StartTime |
| No rate limiting on public download endpoint | 🟡 Medium | Add per-IP rate limiting middleware |
| `context.Background()` used instead of request context | 🟡 Medium | Pass `c.Request.Context()` to service layer |
| Path traversal not explicitly mitigated | 🟢 Low | Add `path.Clean()` before prefix check |
| SAS logic duplicated in `library` and `ps_mag` | 🟢 Low | Extract shared SAS utility function |

---

## 6. Conclusion

The SAS URL approach used in the library feature is **architecturally correct** for serving private Azure Blob Storage files. Clients get a one-hour, read-only, HTTPS-only URL scoped to a specific PDF — exactly what Azure's SAS design is intended for.

The primary improvement opportunity is migrating from an **account key-signed Service SAS** to a **User Delegation SAS** using Azure Managed Identity. This is Microsoft's official recommendation and eliminates the need to store sensitive account keys in environment variables. The secondary priority is adding **rate limiting** to the unauthenticated download endpoint to guard against abuse.

The remaining findings are minor polish items that improve resilience and maintainability without changing the fundamental correctness of the implementation.

---

## 7. References

- [Azure Storage SAS Overview](https://learn.microsoft.com/en-us/azure/storage/common/storage-sas-overview)
- [Create a User Delegation SAS](https://learn.microsoft.com/en-us/rest/api/storageservices/create-user-delegation-sas)
- [Azure SDK for Go – azblob package](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob)
- [Azure SDK for Go – sas package](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas)
- [Prevent Shared Key Authorization](https://learn.microsoft.com/en-us/azure/storage/common/shared-key-authorization-prevent)

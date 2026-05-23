package pmcs_sbs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

const (
	LibraryContainerName = "library"
	PMCSSBSPrefix        = "pmcs_sbs/"
)

// ServiceImpl holds the Azure blob client for all blob operations.
type ServiceImpl struct {
	blobClient *azblob.Client
}

// NewService creates a Service backed by the given Azure blob client.
func NewService(blobClient *azblob.Client) Service {
	return &ServiceImpl{blobClient: blobClient}
}

// GetFolders retrieves all top-level folders from pmcs_sbs/ in Azure Blob Storage.
func (s *ServiceImpl) GetFolders(ctx context.Context) (*FoldersListResponse, error) {
	slog.Info("Fetching PMCS SBS folders from Azure Blob Storage",
		"container", LibraryContainerName,
		"prefix", PMCSSBSPrefix)

	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)
	prefix := PMCSSBSPrefix
	pager := containerClient.NewListBlobsHierarchyPager(
		"/",
		&container.ListBlobsHierarchyOptions{Prefix: &prefix},
	)

	folders := []FolderResponse{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS SBS folders from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"prefix", PMCSSBSPrefix)
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}

		for _, p := range page.Segment.BlobPrefixes {
			if p.Name == nil {
				continue
			}
			fullPath := *p.Name
			folderName := strings.TrimSuffix(strings.TrimPrefix(fullPath, PMCSSBSPrefix), "/")
			if folderName == "" {
				continue
			}
			folders = append(folders, FolderResponse{
				Name:        folderName,
				FullPath:    fullPath,
				DisplayName: formatDisplayName(folderName),
			})
		}
	}

	slog.Info("Successfully fetched PMCS SBS folders",
		"count", len(folders),
		"container", LibraryContainerName)

	return &FoldersListResponse{Folders: folders, Count: len(folders)}, nil
}

// GetFiles retrieves all JSON files from a specific folder in pmcs_sbs/.
func (s *ServiceImpl) GetFiles(ctx context.Context, folderName string) (*FilesListResponse, error) {
	if strings.TrimSpace(folderName) == "" {
		return nil, ErrEmptyFolderName
	}

	folderPrefix := fmt.Sprintf("%s%s/", PMCSSBSPrefix, folderName)

	slog.Info("Fetching PMCS SBS files from Azure Blob Storage",
		"container", LibraryContainerName,
		"folderPrefix", folderPrefix)

	containerClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &folderPrefix,
	})

	files := []FileResponse{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("Failed to list PMCS SBS files from Azure Blob Storage",
				"error", err,
				"container", LibraryContainerName,
				"folderPrefix", folderPrefix)
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			if !strings.HasSuffix(strings.ToLower(blobPath), ".json") {
				slog.Debug("Skipping non-JSON file", "blobPath", blobPath)
				continue
			}

			var sizeBytes int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				sizeBytes = *blob.Properties.ContentLength
			}
			var lastModified string
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = blob.Properties.LastModified.Format(time.RFC3339)
			}

			files = append(files, FileResponse{
				Name:         extractFileName(blobPath),
				BlobPath:     blobPath,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}

	slog.Info("Successfully fetched PMCS SBS files",
		"count", len(files),
		"folderName", folderName,
		"container", LibraryContainerName)

	return &FilesListResponse{FolderName: folderName, Files: files, Count: len(files)}, nil
}

// GetFileContent downloads a JSON blob from Azure and returns its raw content.
// ctx should be the request context so the Azure DownloadStream call is cancelled on client disconnect.
func (s *ServiceImpl) GetFileContent(ctx context.Context, blobPath string) (json.RawMessage, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}

	// Sanitise path to prevent directory traversal (e.g. "pmcs_sbs/../secret.json").
	blobPath = path.Clean(blobPath)

	if !strings.HasPrefix(blobPath, PMCSSBSPrefix) {
		return nil, ErrInvalidBlobPath
	}
	if !strings.HasSuffix(strings.ToLower(blobPath), ".json") {
		return nil, ErrInvalidFileType
	}

	slog.Info("Downloading PMCS SBS file content from Azure Blob Storage",
		"container", LibraryContainerName,
		"blobPath", blobPath)

	blobClient := s.blobClient.ServiceClient().NewContainerClient(LibraryContainerName).NewBlobClient(blobPath)
	downloadResponse, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		slog.Error("Failed to download PMCS SBS file", "error", err, "blobPath", blobPath)
		return nil, fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	defer downloadResponse.Body.Close()

	data, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		slog.Error("Failed to read PMCS SBS file content", "error", err, "blobPath", blobPath)
		return nil, fmt.Errorf("%w: %v", ErrBlobReadFailed, err)
	}

	if !json.Valid(data) {
		slog.Error("PMCS SBS blob contains invalid JSON", "blobPath", blobPath, "size", len(data))
		return nil, ErrInvalidJSON
	}

	slog.Info("Successfully downloaded PMCS SBS file content",
		"blobPath", blobPath,
		"size", len(data))

	return json.RawMessage(data), nil
}

// formatDisplayName converts folder names to human-readable display names.
// Examples: "hmmwv" -> "HMMWV", "m2-bradley" -> "M2 BRADLEY", "m2_bradley" -> "M2 BRADLEY"
func formatDisplayName(name string) string {
	display := strings.ToUpper(name)
	display = strings.ReplaceAll(display, "-", " ")
	display = strings.ReplaceAll(display, "_", " ")
	return display
}

// extractFileName returns the filename from a blob path.
// Example: "pmcs_sbs/hmmwv/file.json" -> "file.json"
func extractFileName(blobPath string) string {
	parts := strings.Split(blobPath, "/")
	if len(parts) == 0 {
		return blobPath
	}
	return parts[len(parts)-1]
}

package docs_equipment

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"miltechserver/api/library/shared"
)

const (
	containerName = "library"
	imagePrefix   = "docs_equipment/images/"
)

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

type serviceImpl struct {
	repo       Repository
	blobClient *azblob.Client
}

func NewService(repo Repository, blobClient *azblob.Client) Service {
	return &serviceImpl{repo: repo, blobClient: blobClient}
}

func (s *serviceImpl) GetAllPaginated(page int) (EquipmentDetailsPageResponse, error) {
	return s.repo.GetAllPaginated(page)
}

func (s *serviceImpl) GetFamilies() (FamiliesResponse, error) {
	return s.repo.GetFamilies()
}

func (s *serviceImpl) GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error) {
	return s.repo.GetByFamilyPaginated(strings.TrimSpace(family), page)
}

func (s *serviceImpl) SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error) {
	return s.repo.SearchPaginated(strings.TrimSpace(query), page)
}

func isImageFile(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	return allowedImageExts[ext]
}

func (s *serviceImpl) ListImageFamilies() (*ImageFamiliesResponse, error) {
	ctx := context.Background()
	containerClient := s.blobClient.ServiceClient().NewContainerClient(containerName)
	prefix := imagePrefix
	pager := containerClient.NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{
		Prefix: &prefix,
	})

	var families []ImageFamilyFolder
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, p := range page.Segment.BlobPrefixes {
			if p.Name == nil {
				continue
			}
			fullPath := *p.Name
			name := strings.TrimPrefix(fullPath, imagePrefix)
			name = strings.TrimSuffix(name, "/")
			if name == "" {
				continue
			}
			displayName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(name, "-", " "), "_", " "))
			families = append(families, ImageFamilyFolder{
				Name:        name,
				FullPath:    fullPath,
				DisplayName: displayName,
			})
		}
	}

	return &ImageFamiliesResponse{Families: families, Count: len(families)}, nil
}

func (s *serviceImpl) ListFamilyImages(family string) (*FamilyImagesResponse, error) {
	if strings.TrimSpace(family) == "" {
		return nil, ErrEmptyParam
	}
	ctx := context.Background()
	containerClient := s.blobClient.ServiceClient().NewContainerClient(containerName)
	prefix := imagePrefix + strings.TrimSpace(family) + "/"
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var images []ImageItem
	for pager.More() {
		pg, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, blob := range pg.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			parts := strings.Split(blobPath, "/")
			fileName := parts[len(parts)-1]
			if !isImageFile(fileName) {
				slog.Debug("Skipping non-image blob", "blobPath", blobPath)
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
			images = append(images, ImageItem{
				Name:         fileName,
				BlobPath:     blobPath,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}

	return &FamilyImagesResponse{
		Family: strings.TrimSpace(family),
		Images: images,
		Count:  len(images),
	}, nil
}

// GetFamilyImageURLs lists all images in a family folder and generates SAS download
// URLs for every image in a single call. It fetches one User Delegation Key from
// Azure AD and signs all blob paths locally, avoiding the N+1 credential call that
// would result from calling GenerateImageDownloadURL per image.
func (s *serviceImpl) GetFamilyImageURLs(ctx context.Context, family string) (*FamilyImageURLsResponse, error) {
	family = strings.TrimSpace(family)
	if family == "" {
		return nil, ErrEmptyParam
	}

	// Step 1: List all image blobs in the family folder (one Azure Storage call).
	imageList, err := s.ListFamilyImages(family)
	if err != nil {
		return nil, err
	}

	if len(imageList.Images) == 0 {
		return &FamilyImageURLsResponse{
			Family:    family,
			Images:    []ImageURLItem{},
			Count:     0,
			ExpiresAt: "",
		}, nil
	}

	// Step 2: Obtain a single User Delegation Key (one Azure AD call).
	expiresAt := time.Now().UTC().Add(1 * time.Hour)
	svcClient := s.blobClient.ServiceClient()

	strPtr := func(s string) *string { return &s }
	udk, err := svcClient.GetUserDelegationCredential(
		ctx,
		service.KeyInfo{
			Start:  strPtr(time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339)),
			Expiry: strPtr(expiresAt.Format(time.RFC3339)),
		},
		nil,
	)
	if err != nil {
		slog.Error("Failed to get user delegation credential for batch signing",
			"family", family, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
	}

	// Step 3: Sign each blob path locally (pure CPU, no network calls).
	permissions := sas.BlobPermissions{Read: true}
	items := make([]ImageURLItem, 0, len(imageList.Images))

	for _, img := range imageList.Images {
		bc := svcClient.NewContainerClient(containerName).NewBlobClient(img.BlobPath)

		sasQueryParams, signErr := sas.BlobSignatureValues{
			Protocol:      sas.ProtocolHTTPS,
			ExpiryTime:    expiresAt,
			Permissions:   permissions.String(),
			ContainerName: containerName,
			BlobName:      img.BlobPath,
		}.SignWithUserDelegation(udk)
		if signErr != nil {
			slog.Warn("Failed to sign SAS for image, skipping",
				"blobPath", img.BlobPath, "error", signErr)
			continue
		}

		downloadURL := fmt.Sprintf("%s?%s", bc.URL(), sasQueryParams.Encode())

		items = append(items, ImageURLItem{
			Name:         img.Name,
			BlobPath:     img.BlobPath,
			DownloadURL:  downloadURL,
			SizeBytes:    img.SizeBytes,
			LastModified: img.LastModified,
		})
	}

	return &FamilyImageURLsResponse{
		Family:    family,
		Images:    items,
		Count:     len(items),
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *serviceImpl) GenerateImageDownloadURL(ctx context.Context, blobPath string) (*ImageDownloadResponse, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}
	blobPath = path.Clean(blobPath)
	if !strings.HasPrefix(blobPath, imagePrefix) {
		return nil, ErrInvalidBlobPath
	}
	if !isImageFile(blobPath) {
		return nil, ErrInvalidFileType
	}

	blobClient := s.blobClient.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobPath)
	if _, err := blobClient.GetProperties(ctx, nil); err != nil {
		slog.Error("Equipment image blob not found", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrImageNotFound, err)
	}

	sasResult, err := shared.GenerateBlobSASURL(ctx, s.blobClient, containerName, blobPath)
	if err != nil {
		slog.Error("Failed to generate SAS for equipment image", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
	}

	return &ImageDownloadResponse{
		BlobPath:    blobPath,
		DownloadURL: sasResult.URL,
		ExpiresAt:   sasResult.ExpiresAt.Format(time.RFC3339),
	}, nil
}

package docs_equipment

import "context"

// Service provides methods for equipment details data and image operations.
type Service interface {
	// DB operations
	GetAllPaginated(ctx context.Context, page int) (EquipmentDetailsPageResponse, error)
	GetFamilies(ctx context.Context) (FamiliesResponse, error)
	GetByFamilyPaginated(ctx context.Context, family string, page int) (EquipmentDetailsPageResponse, error)
	SearchPaginated(ctx context.Context, query string, page int) (EquipmentDetailsPageResponse, error)

	// Blob operations
	ListImageFamilies(ctx context.Context) (*ImageFamiliesResponse, error)
	ListFamilyImages(ctx context.Context, family string) (*FamilyImagesResponse, error)
	GetFamilyImageURLs(ctx context.Context, family string) (*FamilyImageURLsResponse, error)
	GenerateImageDownloadURL(ctx context.Context, blobPath string) (*ImageDownloadResponse, error)
}

package docs_equipment

import "context"

// Service provides methods for equipment details data and image operations.
type Service interface {
	// DB operations
	GetAllPaginated(page int) (EquipmentDetailsPageResponse, error)
	GetFamilies() (FamiliesResponse, error)
	GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error)
	SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error)

	// Blob operations
	ListImageFamilies() (*ImageFamiliesResponse, error)
	ListFamilyImages(family string) (*FamilyImagesResponse, error)
	GetFamilyImageURLs(ctx context.Context, family string) (*FamilyImageURLsResponse, error)
	GenerateImageDownloadURL(ctx context.Context, blobPath string) (*ImageDownloadResponse, error)
}

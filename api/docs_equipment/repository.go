package docs_equipment

// Repository defines database operations for equipment details.
type Repository interface {
	GetAllPaginated(page int) (EquipmentDetailsPageResponse, error)
	GetFamilies() (FamiliesResponse, error)
	GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error)
	SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error)
}

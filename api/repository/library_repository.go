package repository

// LibraryRepository handles database operations for library feature
// Note: Currently not needed for Phase 1 as we only query Azure Blob Storage
// This is scaffolding for future features like tracking downloads, favorites, etc.
type LibraryRepository interface {
	// Future: Track user downloads, favorites, etc.
	// RecordDownload(userID string, documentPath string) error
	// GetUserDownloadHistory(userID string) ([]DownloadRecord, error)
	// AddFavorite(userID string, documentPath string) error
	// RemoveFavorite(userID string, documentPath string) error
	// GetUserFavorites(userID string) ([]string, error)
}

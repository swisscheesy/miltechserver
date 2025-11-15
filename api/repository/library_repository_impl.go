package repository

import (
	"database/sql"
)

type LibraryRepositoryImpl struct {
	db *sql.DB
}

func NewLibraryRepositoryImpl(db *sql.DB) LibraryRepository {
	return &LibraryRepositoryImpl{
		db: db,
	}
}

// Future implementations will go here when we add features like:
// - Download tracking
// - User favorites
// - Analytics on most downloaded documents

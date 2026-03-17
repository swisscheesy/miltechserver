package analytics

import "database/sql"

type Service interface {
	IncrementItemSearchSuccess(niin string, nomenclature string) error
	IncrementPMCSManualDownload(entityKey string, entityLabel string) error
	IncrementPSMagDownload(filename string) error
	IncrementCounter(eventType string, entityKey string, entityLabel string) error
}

func New(db *sql.DB) Service {
	repo := NewRepository(db)
	return NewService(repo)
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

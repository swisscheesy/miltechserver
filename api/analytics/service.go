package analytics

import (
	"context"
	"database/sql"
)

type Service interface {
	IncrementItemSearchSuccess(ctx context.Context, niin string, nomenclature string) error
	IncrementPMCSManualDownload(ctx context.Context, entityKey string, entityLabel string) error
	IncrementPSMagDownload(ctx context.Context, filename string) error
	IncrementCounter(ctx context.Context, eventType string, entityKey string, entityLabel string) error
}

func New(db *sql.DB) Service {
	repo := NewRepository(db)
	return NewService(repo)
}

func NewService(repo Repository) Service {
	return &ServiceImpl{repo: repo}
}

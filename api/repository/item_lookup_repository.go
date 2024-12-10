package repository

import (
	"context"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemLookupRepository interface {
	SearchLINByPage(ctx context.Context, page int) (model.LINPageResponse, error)
	SearchLINByNIIN(ctx context.Context, lin string) ([]db.LookupLinNiinModel, error)

	//SearchUOCByPage(page int) ([]string, error)
	//SearchSpecificUOC(uoc string) ([]string, error)
}

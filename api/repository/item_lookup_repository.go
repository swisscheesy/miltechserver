package repository

import (
	"context"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemLookupRepository interface {
	SearchLINByPage(ctx context.Context, page int) (model.LINPageResponse, error)
	SearchLINByNIIN(ctx context.Context, niin string) ([]db.LookupLinNiinModel, error)
	SearchNIINByLIN(ctx context.Context, lin string) ([]db.LookupLinNiinModel, error)

	SearchUOCByPage(ctx context.Context, page int) (model.UOCPageResponse, error)
	SearchSpecificUOC(ctx context.Context, uoc string) ([]db.LookupUocModel, error)
	SearchUOCByModel(ctx context.Context, model string) ([]db.LookupUocModel, error)
}

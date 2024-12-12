package service

import (
	"context"
	"miltechserver/model"
	"miltechserver/prisma/db"
)

type ItemLookupService interface {
	LookupLINByPage(ctx context.Context, page int) (model.LINPageResponse, error)
	LookupLINByNIIN(ctx context.Context, niin string) ([]db.LookupLinNiinModel, error)
	LookupNIINByLIN(ctx context.Context, niin string) ([]db.LookupLinNiinModel, error)

	LookupUOCByPage(ctx context.Context, page int) (model.UOCPageResponse, error)
	LookupSpecificUOC(ctx context.Context, uoc string) ([]db.LookupUocModel, error)
	LookupUOCByModel(ctx context.Context, model string) ([]db.LookupUocModel, error)
}

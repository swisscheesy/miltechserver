package service

import (
	"context"
	"miltechserver/model"
)

type ItemQueryService interface {
	FindShortByNiin(ctx context.Context, niin string) (model.ShortItem, error)
	//FindShortByPart(ctx context.Context, part string) response.StandardResponse
}

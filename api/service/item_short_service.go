package service

import (
	"context"
	"miltechserver/model"
)

type ItemShortService interface {
	FindShortByNiin(ctx context.Context, niin string) (model.ShortItem, error)
	FindShortByPart(ctx context.Context, part string) ([]model.ShortItem, error)
}

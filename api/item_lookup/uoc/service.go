package uoc

import (
	"context"
	"miltechserver/api/response"
)

type Service interface {
	LookupByPage(ctx context.Context, page int) (response.UOCPageResponse, error)
	LookupSpecific(ctx context.Context, uoc string) (response.UOCPageResponse, error)
	LookupByModel(ctx context.Context, model string) (response.UOCPageResponse, error)
}

package eic

import (
	"context"
	"miltechserver/api/response"
)

type Service interface {
	LookupByNIIN(ctx context.Context, niin string) ([]response.EICConsolidatedItem, error)
	LookupByLIN(ctx context.Context, lin string) ([]response.EICConsolidatedItem, error)
	LookupByFSCPaginated(ctx context.Context, fsc string, page int) (response.EICPageResponse, error)
	LookupAllPaginated(ctx context.Context, page int, search string) (response.EICPageResponse, error)
}

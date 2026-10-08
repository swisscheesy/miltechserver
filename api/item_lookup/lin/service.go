package lin

import (
	"context"
	"miltechserver/api/response"
)

type Service interface {
	LookupByPage(ctx context.Context, page int) (response.LINPageResponse, error)
	LookupByNIIN(ctx context.Context, niin string) (response.LINPageResponse, error)
	LookupNIINByLIN(ctx context.Context, lin string) (response.LINPageResponse, error)
}

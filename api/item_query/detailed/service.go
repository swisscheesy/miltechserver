package detailed

import (
	"context"

	"miltechserver/api/response"
)

type Service interface {
	FindDetailedItem(ctx context.Context, niin string) (response.DetailedResponse, error)
}

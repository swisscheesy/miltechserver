package detailed

import (
	"context"

	"miltechserver/api/response"
)

type Repository interface {
	GetDetailedItemData(ctx context.Context, niin string) (response.DetailedResponse, error)
}

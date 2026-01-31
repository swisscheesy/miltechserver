package detailed

import "miltechserver/api/response"

type Repository interface {
	GetDetailedItemData(niin string) (response.DetailedResponse, error)
}

package repository

import "miltechserver/api/response"

type ItemDetailedRepository interface {
	GetDetailedItemData(niin string) (response.DetailedResponse, error)
}

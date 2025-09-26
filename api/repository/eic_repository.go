package repository

import (
	"miltechserver/api/response"
)

type EICRepository interface {
	GetByNIIN(niin string) ([]response.EICConsolidatedItem, error)
	GetByLIN(lin string) ([]response.EICConsolidatedItem, error)
	GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
	GetAllPaginated(page int, search string) (response.EICPageResponse, error)
}

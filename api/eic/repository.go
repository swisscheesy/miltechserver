package eic

import "miltechserver/api/response"

type Repository interface {
	GetByNIIN(niin string) ([]response.EICConsolidatedItem, error)
	GetByLIN(lin string) ([]response.EICConsolidatedItem, error)
	GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
	GetAllPaginated(page int, search string) (response.EICPageResponse, error)
}

package service

import (
	"miltechserver/api/response"
)

type EICService interface {
	LookupByNIIN(niin string) ([]response.EICConsolidatedItem, error)
	LookupByLIN(lin string) ([]response.EICConsolidatedItem, error)
	LookupByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
	LookupAllPaginated(page int, search string) (response.EICPageResponse, error)
}

package service

import "miltechserver/api/response"

type ItemDetailedService interface {
	FindDetailedItem(niin string) (response.DetailedResponse, error)
}

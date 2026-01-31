package detailed

import "miltechserver/api/response"

type Service interface {
	FindDetailedItem(niin string) (response.DetailedResponse, error)
}

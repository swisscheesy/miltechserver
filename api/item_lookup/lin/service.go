package lin

import "miltechserver/api/response"

type Service interface {
	LookupByPage(page int) (response.LINPageResponse, error)
	LookupByNIIN(niin string) (response.LINPageResponse, error)
	LookupNIINByLIN(lin string) (response.LINPageResponse, error)
}

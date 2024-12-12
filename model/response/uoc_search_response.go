package response

import "miltechserver/prisma/db"

type UOCPLookupResponse struct {
	UOCs  []db.LookupUocModel `json:"uocs"`
	Count int                 `json:"count"`
}

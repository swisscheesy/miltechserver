package model

import "miltechserver/prisma/db"

type UOCPageResponse struct {
	UOCs       []db.LookupUocModel `json:"uocs"`
	Count      int                 `json:"count"`
	Page       int                 `json:"page"`
	TotalPages int                 `json:"total_pages"`
	IsLastPage bool                `json:"is_last_page"`
}

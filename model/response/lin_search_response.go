package response

import "miltechserver/prisma/db"

// LinSearchResponse represents the response structure for LIN search.
// \param Lins - the LIN data retrieved from the database.
// \param Count - the total count of LINs.
type LinSearchResponse struct {
	Lins  []db.LookupLinNiinModel `json:"lins"`
	Count int                     `json:"count"`
}

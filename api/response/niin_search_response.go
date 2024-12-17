package response

import "miltechserver/prisma/db"

// NiinSearchResponse represents the response structure for Niin search by LIN search.
// \param Niins - the Niin data retrieved from the database.
// \param Count - the total count of NIINs.
type NiinSearchResponse struct {
	Niins []db.LookupLinNiinModel `json:"niins"`
	Count int                     `json:"count"`
}

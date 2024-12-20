package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

// NiinSearchResponse represents the response structure for Niin search by LIN search.
// \param Niins - the Niin data retrieved from the database.
// \param Count - the total count of NIINs.
type NiinSearchResponse struct {
	Niins []model.LookupLinNiin `json:"niins"`
	Count int                   `json:"count"`
}

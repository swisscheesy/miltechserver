package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type QuickListsClothingResponse struct {
	Clothing []model.QuickListClothing `json:"clothing"`
	Count    int                       `json:"count"`
}

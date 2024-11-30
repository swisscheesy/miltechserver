package model

import (
	"miltechserver/model/details"
)

// DetailedItem is a struct that represents a detailed item, containing all relevant information about an item.
type DetailedItem struct {
	Amdf                    details.Amdf                    `json:"army_master_data_file"`
	ArmyPackagingAndFreight details.ArmyPackagingAndFreight `json:"army_packaging_and_freight"`
	Sarsscat                details.Sarsscat                `json:"sarsscat"`
	Identification          details.Identification          `json:"identification"`
	Management              details.Management              `json:"management"`
	Reference               details.Reference               `json:"reference"`
	Freight                 details.Freight                 `json:"freight"`
	Packaging               details.Packaging               `json:"packaging"`
	Characteristics         details.Characteristics         `json:"characteristics"`
	Disposition             details.Disposition             `json:"disposition"`
}

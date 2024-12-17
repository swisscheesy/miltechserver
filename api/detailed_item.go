package api

import (
	details2 "miltechserver/api/details"
)

// DetailedItem is a struct that represents a detailed item, containing all relevant information about an item.
type DetailedItem struct {
	Amdf                    details2.Amdf                    `json:"army_master_data_file"`
	ArmyPackagingAndFreight details2.ArmyPackagingAndFreight `json:"army_packaging_and_freight"`
	Sarsscat                details2.Sarsscat                `json:"sarsscat"`
	Identification          details2.Identification          `json:"identification"`
	Management              details2.Management              `json:"management"`
	Reference               details2.Reference               `json:"reference"`
	Freight                 details2.Freight                 `json:"freight"`
	Packaging               details2.Packaging               `json:"packaging"`
	Characteristics         details2.Characteristics         `json:"characteristics"`
	Disposition             details2.Disposition             `json:"disposition"`
}

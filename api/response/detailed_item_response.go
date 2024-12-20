package response

import (
	"miltechserver/api/details"
)

type DetailedResponse struct {
	Amdf                    details.Amdf
	ArmyPackagingAndFreight details.ArmyPackagingAndFreight
	Characteristics         details.Characteristics
	Disposition             details.Disposition
	Freight                 details.Freight
	Identification          details.Identification
	Management              details.Management
	Packaging               details.Packaging
	Reference               details.Reference
	Sarsscat                details.Sarsscat
}

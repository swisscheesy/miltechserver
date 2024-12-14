package repository

import (
	"github.com/gin-gonic/gin"
	"miltechserver/model/details"
)

type ItemDetailedRepository interface {
	GetAmdfData(ctx *gin.Context, niin string) (details.Amdf, error)
	GetArmyPackagingAndFreight(ctx *gin.Context, niin string) (details.ArmyPackagingAndFreight, error)
	GetSarsscat(ctx *gin.Context, niin string) (details.Sarsscat, error)
	GetIdentification(ctx *gin.Context, niin string) (details.Identification, error)
	GetManagement(ctx *gin.Context, niin string) (details.Management, error)
	GetReference(ctx *gin.Context, niin string) (details.Reference, error)
	GetFreight(ctx *gin.Context, niin string) (details.Freight, error)
	GetPackaging(ctx *gin.Context, niin string) (details.Packaging, error)
	GetCharacteristics(ctx *gin.Context, niin string) (details.Characteristics, error)
	GetDisposition(ctx *gin.Context, niin string) (details.Disposition, error)

	// Helper methods to pull individual table data

}

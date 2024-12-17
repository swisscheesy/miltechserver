package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/miltech_ng/public/model"
)

type ItemShortService interface {
	FindShortByNiin(ctx *gin.Context, niin string) (model.NiinLookup, error)
	FindShortByPart(ctx *gin.Context, part string) ([]model.NiinLookup, error)
}

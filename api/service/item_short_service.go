package service

import (
	"github.com/gin-gonic/gin"
	"miltechserver/model"
)

type ItemShortService interface {
	FindShortByNiin(ctx *gin.Context, niin string) (model.ShortItem, error)
	FindShortByPart(ctx *gin.Context, part string) ([]model.ShortItem, error)
}

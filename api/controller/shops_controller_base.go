package controller

import (
	"miltechserver/api/shops/facade"
	"miltechserver/api/shops/shared"

	"github.com/gin-gonic/gin"
)

type ShopsController struct {
	ShopsService facade.Service
	auth         shared.ShopAuthorization
}

func NewShopsController(shopsService facade.Service, auth shared.ShopAuthorization) *ShopsController {
	return &ShopsController{
		ShopsService: shopsService,
		auth:         auth,
	}
}

func (controller *ShopsController) serviceForRequest(c *gin.Context) facade.Service {
	cachedAuth := shared.CachedAuthorizationFromContext(c, func() shared.ShopAuthorization {
		return controller.auth
	})
	return controller.ShopsService.WithAuthorization(cachedAuth)
}

package controller

import "miltechserver/api/shops/facade"

type ShopsController struct {
	ShopsService facade.Service
}

func NewShopsController(shopsService facade.Service) *ShopsController {
	return &ShopsController{ShopsService: shopsService}
}

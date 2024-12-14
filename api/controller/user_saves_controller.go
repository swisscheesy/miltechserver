package controller

import "miltechserver/api/service"

type UserSavesController struct {
	UserSavesService service.UserSavesService
}

func NewUserSavesController(userSavesService service.UserSavesService) *UserSavesController {
	return &UserSavesController{UserSavesService: userSavesService}
}

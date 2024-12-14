package service

import (
	"firebase.google.com/go/v4/auth"
	"miltechserver/prisma/db"
)

type UserSavesService struct {
	Db       *db.PrismaClient
	FireAuth *auth.Client
}

func NewUserSavesService(db *db.PrismaClient, fireAuth *auth.Client) *UserSavesService {
	return &UserSavesService{Db: db, FireAuth: fireAuth}
}

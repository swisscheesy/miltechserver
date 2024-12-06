package service

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"miltechserver/prisma/db"
)

type AuthService struct {
	Db       *db.PrismaClient
	FireAuth *auth.Client
}

func NewAuthService(db *db.PrismaClient, fireAuth *auth.Client) *AuthService {
	return &AuthService{Db: db, FireAuth: fireAuth}
}

func (service *AuthService) Login(ctx context.Context) {

}

func (service *AuthService) Register(ctx context.Context) {

}

func (service *AuthService) Logout(ctx context.Context) {

}

func (service *AuthService) RefreshToken(ctx context.Context) {}

func (service *AuthService) VerifyToken(ctx context.Context) {}

func (service *AuthService) GetUser(ctx context.Context) {}

func (service *AuthService) UpdateUser(ctx context.Context) {}

func (service *AuthService) DeleteUser(ctx context.Context) {}

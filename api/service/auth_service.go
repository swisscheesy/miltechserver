package service

import (
	"context"
	"database/sql"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type AuthService struct {
	Db       *sql.DB
	FireAuth *auth.Client
}

func NewAuthService(db *sql.DB, fireAuth *auth.Client) *AuthService {
	return &AuthService{Db: db, FireAuth: fireAuth}
}

//func (service *AuthService) Login(c *gin.Context) {
//	var credentials struct {
//		Email    string `json:"email"`
//		Password string `json:"password"`
//	}
//
//	if err := c.ShouldBindJSON(&credentials); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
//		return
//	}
//
//}

func (service *AuthService) Register(c *gin.Context) {

}

func (service *AuthService) Logout(c *gin.Context) {

}

func (service *AuthService) RefreshToken(c *gin.Context) {}

func (service *AuthService) VerifyToken(c *gin.Context) {}

func (service *AuthService) GetUser(c *gin.Context) {}

func (service *AuthService) UpdateUser(c *gin.Context) {}

func (service *AuthService) DeleteUser(c context.Context) {}

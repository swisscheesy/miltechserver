package shared

import (
	"errors"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

func GetUserFromContext(c *gin.Context) (*bootstrap.User, error) {
	ctxUser, ok := c.Get("user")
	if !ok {
		return nil, errors.New("unauthorized")
	}

	user, ok := ctxUser.(*bootstrap.User)
	if !ok || user == nil {
		return nil, errors.New("unauthorized")
	}

	return user, nil
}

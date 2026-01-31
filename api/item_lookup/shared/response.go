package shared

import (
	"errors"
	"net/http"

	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		return
	}
	if errors.Is(err, ErrEmptyParam) || errors.Is(err, ErrInvalidPage) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
}

func WriteSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    data,
	})
}

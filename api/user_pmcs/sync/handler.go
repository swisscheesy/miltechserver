package sync

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type Handler struct {
	service Service
	config  shared.Config
}

func (handler Handler) getDelta(context *gin.Context) {
	context.Header("Vary", "Accept-Encoding")
	user, apiError := authenticatedUser(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}

	delta, err := handler.service.GetDelta(
		context.Request.Context(),
		user,
		context.Query("after"),
		context.Query("limit"),
	)
	if err != nil {
		writeError(context, err)
		return
	}

	context.JSON(http.StatusOK, response.StandardResponse{
		Status:  http.StatusOK,
		Message: "",
		Data:    delta,
	})
}

func authenticatedUser(
	context *gin.Context,
) (*bootstrap.User, *shared.APIError) {
	value, exists := context.Get("user")
	if !exists {
		return nil, shared.NewAuthenticationRequired(
			"authentication is required",
			nil,
		)
	}
	user, ok := value.(*bootstrap.User)
	if !ok || user == nil || user.UserID == "" {
		return nil, shared.NewAuthenticationRequired(
			"authentication is required",
			nil,
		)
	}
	return user, nil
}

func writeError(context *gin.Context, err error) {
	var apiError *shared.APIError
	if errors.As(err, &apiError) {
		shared.WriteAPIError(context, apiError)
		return
	}
	internalError := shared.NewInternalError(
		"unexpected server failure",
		nil,
	)
	internalError.Cause = err
	shared.WriteAPIError(context, internalError)
}

package community

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

const ownerCommunityCacheControl = "private, no-cache"

type Handler struct {
	service Service
}

func (handler Handler) release(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.Release(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		context.Param("revision_id"),
		context.GetHeader("If-Match"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setOwnerHeaders(context, etag)
	writeSuccess(context, http.StatusOK, result.Aggregate)
}

func (handler Handler) retire(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.Retire(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		context.GetHeader("If-Match"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setOwnerHeaders(context, etag)
	writeSuccess(context, http.StatusOK, result.Aggregate)
}

func userFromContext(context *gin.Context) (*bootstrap.User, *shared.APIError) {
	value, exists := context.Get("user")
	if !exists {
		return nil, shared.NewAuthenticationRequired(
			"authentication is required",
			nil,
		)
	}
	user, ok := value.(*bootstrap.User)
	if !ok || user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, shared.NewAuthenticationRequired(
			"authentication is required",
			nil,
		)
	}
	return user, nil
}

func writeServiceError(context *gin.Context, err error) {
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

func setOwnerHeaders(context *gin.Context, etag string) {
	context.Header("ETag", etag)
	context.Header("Cache-Control", ownerCommunityCacheControl)
}

func writeSuccess(context *gin.Context, status int, data any) {
	context.JSON(status, response.StandardResponse{
		Status:  status,
		Message: "",
		Data:    data,
	})
}

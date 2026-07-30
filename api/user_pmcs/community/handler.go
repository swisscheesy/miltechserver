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
const publicCommunityCacheControl = "public, no-cache"

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

func (handler Handler) browse(context *gin.Context) {
	page, err := handler.service.Browse(
		context.Request.Context(),
		context.Query("after"),
		context.Query("limit"),
		context.Query("model"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setPublicHeaders(context)
	writeSuccess(context, http.StatusOK, page)
}

func (handler Handler) getCurrentRelease(context *gin.Context) {
	release, etag, err := handler.service.GetCurrentRelease(
		context.Request.Context(),
		context.Param("checklist_id"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setPublicHeaders(context)
	context.Header("ETag", etag)
	matches, apiError := shared.IfNoneMatchMatches(
		context.Request.Header.Values("If-None-Match"),
		etag,
	)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	if matches {
		context.Status(http.StatusNotModified)
		return
	}
	writeSuccess(context, http.StatusOK, release)
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

func setPublicHeaders(context *gin.Context) {
	context.Header("Cache-Control", publicCommunityCacheControl)
	context.Header("Vary", "Accept-Encoding")
}

func writeSuccess(context *gin.Context, status int, data any) {
	context.JSON(status, response.StandardResponse{
		Status:  status,
		Message: "",
		Data:    data,
	})
}

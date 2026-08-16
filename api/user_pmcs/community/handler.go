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

type voteRequest struct {
	Direction int16 `json:"direction"`
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
	page, err := handler.service.BrowsePublic(
		context.Request.Context(),
		context.Query("after"),
		context.Query("limit"),
		context.Query("model"),
		context.Query("sort"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setPublicHeaders(context)
	writeSuccess(context, http.StatusOK, page)
}

func (handler Handler) browseAuthenticated(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	page, err := handler.service.BrowseAuthenticated(
		context.Request.Context(),
		user,
		context.Query("after"),
		context.Query("limit"),
		context.Query("model"),
		context.Query("sort"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setPrivateHeaders(context)
	writeSuccess(context, http.StatusOK, page)
}

func (handler Handler) putVote(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	var request voteRequest
	if apiError := shared.DecodeStrictJSON(context, &request, 1024); apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	if !isVoteDirection(request.Direction) {
		shared.WriteAPIError(context, shared.NewInvalidRequest(
			"direction must be 1 or -1",
			map[string]any{"direction": "1 or -1"},
		))
		return
	}
	mutation, err := handler.service.PutVote(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		request.Direction,
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setPrivateHeaders(context)
	writeSuccess(context, http.StatusOK, mutation)
}

func (handler Handler) deleteVote(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	mutation, err := handler.service.DeleteVote(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}
	setPrivateHeaders(context)
	writeSuccess(context, http.StatusOK, mutation)
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

func setPrivateHeaders(context *gin.Context) {
	context.Header("Cache-Control", ownerCommunityCacheControl)
}

func setPublicHeaders(context *gin.Context) {
	context.Header("Cache-Control", publicCommunityCacheControl)
	context.Header("Vary", "Accept-Encoding")
}

func writeSuccess(context *gin.Context, status int, data any) {
	shared.RecordNodeCount(
		context.Request.Context(),
		shared.TreeNodeCount(data),
	)
	shared.WriteJSON(context, status, response.StandardResponse{
		Status:  status,
		Message: "",
		Data:    data,
	})
}

package subscriptions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

const immutableSubscriptionCacheControl = "private, max-age=31536000, immutable"
const subscriptionCacheControl = "private, no-cache"

type Handler struct{ service Service }

func (handler Handler) install(context *gin.Context) {
	user, apiError := subscriptionUser(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.Install(context.Request.Context(), user, context.Param("checklist_id"), context.GetHeader("If-None-Match"), context.GetHeader("If-Match"))
	if err != nil {
		writeSubscriptionError(context, err)
		return
	}
	setSubscriptionHeaders(context, etag)
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeSubscriptionSuccess(context, status, result)
}
func (handler Handler) unsubscribe(context *gin.Context) {
	user, apiError := subscriptionUser(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.Unsubscribe(context.Request.Context(), user, context.Param("checklist_id"), context.GetHeader("If-Match"))
	if err != nil {
		writeSubscriptionError(context, err)
		return
	}
	setSubscriptionHeaders(context, etag)
	writeSubscriptionSuccess(context, http.StatusOK, result.Subscription)
}
func (handler Handler) getInstalledRelease(context *gin.Context) {
	user, apiError := subscriptionUser(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	release, etag, err := handler.service.GetInstalledRelease(context.Request.Context(), user, context.Param("checklist_id"), context.Param("revision_id"))
	if err != nil {
		writeSubscriptionError(context, err)
		return
	}
	matches, apiError := shared.IfNoneMatchMatches(context.Request.Header.Values("If-None-Match"), etag)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	context.Header("ETag", etag)
	context.Header("Cache-Control", immutableSubscriptionCacheControl)
	if matches {
		context.Status(http.StatusNotModified)
		return
	}
	writeSubscriptionSuccess(context, http.StatusOK, release)
}
func subscriptionUser(context *gin.Context) (*bootstrap.User, *shared.APIError) {
	value, exists := context.Get("user")
	if !exists {
		return nil, shared.NewAuthenticationRequired("authentication is required", nil)
	}
	user, ok := value.(*bootstrap.User)
	if !ok || user == nil || strings.TrimSpace(user.UserID) == "" {
		return nil, shared.NewAuthenticationRequired("authentication is required", nil)
	}
	return user, nil
}
func writeSubscriptionError(context *gin.Context, err error) {
	var apiError *shared.APIError
	if errors.As(err, &apiError) {
		shared.WriteAPIError(context, apiError)
		return
	}
	shared.WriteAPIError(context, shared.NewInternalError("unexpected server failure", nil))
}
func setSubscriptionHeaders(context *gin.Context, etag string) {
	context.Header("ETag", etag)
	context.Header("Cache-Control", subscriptionCacheControl)
}
func writeSubscriptionSuccess(context *gin.Context, status int, data any) {
	context.JSON(status, response.StandardResponse{Status: status, Data: data})
}

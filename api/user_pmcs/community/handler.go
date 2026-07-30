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
	if ifNoneMatchMatches(context.GetHeader("If-None-Match"), etag) {
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

func ifNoneMatchMatches(header string, currentETag string) bool {
	value := strings.TrimSpace(header)
	if value == "*" {
		return true
	}
	tags, valid := splitEntityTagList(value)
	if !valid {
		return false
	}
	currentOpaque, valid := weakEntityTagValue(currentETag)
	if !valid {
		return false
	}
	for _, tag := range tags {
		opaque, tagValid := weakEntityTagValue(tag)
		if tagValid && opaque == currentOpaque {
			return true
		}
	}
	return false
}

func splitEntityTagList(value string) ([]string, bool) {
	if value == "" {
		return nil, true
	}
	var (
		tags     []string
		start    int
		inQuotes bool
	)
	for index, character := range value {
		switch character {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				tag := strings.TrimSpace(value[start:index])
				if tag == "" {
					return nil, false
				}
				tags = append(tags, tag)
				start = index + 1
			}
		}
	}
	if inQuotes {
		return nil, false
	}
	last := strings.TrimSpace(value[start:])
	if last == "" {
		return nil, false
	}
	return append(tags, last), true
}

func weakEntityTagValue(value string) (string, bool) {
	tag := strings.TrimSpace(value)
	if strings.HasPrefix(tag, "W/") {
		tag = tag[2:]
	}
	if len(tag) < 2 || tag[0] != '"' || tag[len(tag)-1] != '"' {
		return "", false
	}
	opaque := tag[1 : len(tag)-1]
	for _, character := range opaque {
		if character == '"' || character <= 0x20 || character == 0x7f {
			return "", false
		}
	}
	return opaque, true
}

func writeSuccess(context *gin.Context, status int, data any) {
	context.JSON(status, response.StandardResponse{
		Status:  status,
		Message: "",
		Data:    data,
	})
}

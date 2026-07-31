package owned

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/response"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

const ownedCacheControl = "private, no-cache"
const immutableOwnedCacheControl = "private, max-age=31536000, immutable"

type Handler struct {
	service Service
	config  shared.Config
}

func (handler Handler) get(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	var conditionalETag string
	if header := context.GetHeader("If-None-Match"); header != "" {
		precondition, err := shared.ParseExistingPrecondition(header)
		if err != nil {
			writeServiceError(context, err)
			return
		}
		conditionalETag = precondition.ETag
	}
	aggregate, etag, err := handler.service.Get(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}

	setOwnedHeaders(context, etag)
	if conditionalETag == etag {
		context.Status(http.StatusNotModified)
		return
	}
	writeSuccess(context, http.StatusOK, aggregate)
}

func (handler Handler) create(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	var draft shared.RevisionInput
	if apiError := shared.DecodeStrictJSON(
		context,
		&draft,
		handler.config.MaxMutationBodyBytes,
	); apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.Create(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		draft,
		context.GetHeader("If-None-Match"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}

	setOwnedHeaders(context, etag)
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeSuccess(context, status, result.Aggregate)
}

func (handler Handler) putDraft(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	var draft shared.RevisionInput
	if apiError := shared.DecodeStrictJSON(
		context,
		&draft,
		handler.config.MaxMutationBodyBytes,
	); apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.PutDraft(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		context.Param("revision_id"),
		draft,
		context.GetHeader("If-Match"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}

	setOwnedHeaders(context, etag)
	writeSuccess(context, http.StatusOK, result.Aggregate)
}

func (handler Handler) deleteDraft(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.DeleteDraft(
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

	setOwnedHeaders(context, etag)
	writeSuccess(context, http.StatusOK, result.Aggregate)
}

func (handler Handler) deleteChecklist(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.DeleteChecklist(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		context.GetHeader("If-Match"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}

	setImmutableOwnedHeaders(context, etag)
	writeSuccess(context, http.StatusOK, result.Aggregate)
}

func (handler Handler) publish(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	var revision shared.RevisionInput
	if apiError := shared.DecodeStrictJSON(
		context,
		&revision,
		handler.config.MaxMutationBodyBytes,
	); apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	result, etag, err := handler.service.Publish(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		context.Param("revision_id"),
		revision,
		context.GetHeader("If-Match"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}

	setOwnedHeaders(context, etag)
	writeSuccess(context, http.StatusOK, result.Aggregate)
}

func (handler Handler) getRevision(context *gin.Context) {
	user, apiError := userFromContext(context)
	if apiError != nil {
		shared.WriteAPIError(context, apiError)
		return
	}
	var conditionalETag string
	if header := context.GetHeader("If-None-Match"); header != "" {
		precondition, err := shared.ParseExistingPrecondition(header)
		if err != nil {
			writeServiceError(context, err)
			return
		}
		conditionalETag = precondition.ETag
	}
	revision, etag, err := handler.service.GetRevision(
		context.Request.Context(),
		user,
		context.Param("checklist_id"),
		context.Param("revision_id"),
	)
	if err != nil {
		writeServiceError(context, err)
		return
	}

	setImmutableOwnedHeaders(context, etag)
	if conditionalETag == etag {
		context.Status(http.StatusNotModified)
		return
	}
	writeSuccess(context, http.StatusOK, revision)
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
	if !ok || user == nil || user.UserID == "" {
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

func setOwnedHeaders(context *gin.Context, etag string) {
	context.Header("ETag", etag)
	context.Header("Cache-Control", ownedCacheControl)
}

func setImmutableOwnedHeaders(context *gin.Context, etag string) {
	context.Header("ETag", etag)
	context.Header("Cache-Control", immutableOwnedCacheControl)
}

func writeSuccess(context *gin.Context, status int, data any) {
	shared.RecordNodeCount(
		context.Request.Context(),
		ownedTreeNodeCount(data),
	)
	shared.WriteJSON(context, status, response.StandardResponse{
		Status:  status,
		Message: "",
		Data:    data,
	})
}

func ownedTreeNodeCount(data any) int {
	if revision, ok := data.(*HistoricalRevision); ok && revision != nil {
		return historicalRevisionNodeCount(*revision)
	}
	return shared.TreeNodeCount(data)
}

func historicalRevisionNodeCount(revision HistoricalRevision) int {
	count := 1 + len(revision.Models) + len(revision.Sections)
	for _, section := range revision.Sections {
		count += len(section.Models) + len(section.Items)
		for _, item := range section.Items {
			count += len(item.Notices) + len(item.ProcedureSteps)
		}
	}
	return count
}

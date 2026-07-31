package sync

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

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

	shared.RecordNodeCount(
		context.Request.Context(),
		accountDeltaNodeCount(delta),
	)
	shared.WriteJSON(context, http.StatusOK, accountDeltaEnvelope(delta))
}

func accountDeltaNodeCount(delta *AccountDelta) int {
	if delta == nil {
		return 0
	}
	count := 0
	for _, change := range delta.Changes {
		count += shared.TreeNodeCount(change.Checklist)
		count += shared.TreeNodeCount(change.Installed)
	}
	return count
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

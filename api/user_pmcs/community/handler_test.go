package community

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type serviceStub struct {
	releaseResult *ReleaseMutationResult
	releaseETag   string
	releaseError  error
	retireResult  *ReleaseMutationResult
	retireETag    string
	retireError   error
	releaseCalls  int
	retireCalls   int
}

func (stub *serviceStub) Release(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ string,
	_ string,
) (*ReleaseMutationResult, string, error) {
	stub.releaseCalls++
	return stub.releaseResult, stub.releaseETag, stub.releaseError
}

func (stub *serviceStub) Retire(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ string,
) (*ReleaseMutationResult, string, error) {
	stub.retireCalls++
	return stub.retireResult, stub.retireETag, stub.retireError
}

func TestReleaseHandlerRequiresAuthentication(t *testing.T) {
	stub := &serviceStub{}
	router := communityTestRouter(stub, false)
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+uuid.NewString()+
			"/community-releases/"+uuid.NewString(),
		nil,
	)
	request.Header.Set("If-Match", `"etag"`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Equal(t, "authentication_required", responseErrorCode(t, response))
	require.Zero(t, stub.releaseCalls)
}

func TestReleaseHandlerRejectsWhitespaceAuthentication(t *testing.T) {
	stub := &serviceStub{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{UserID: "\u00a0"})
		context.Next()
	})
	RegisterRoutes(group, stub)
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+uuid.NewString()+
			"/community-releases/"+uuid.NewString(),
		nil,
	)
	request.Header.Set("If-Match", `"etag"`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Equal(t, "authentication_required", responseErrorCode(t, response))
	require.Zero(t, stub.releaseCalls)
}

func TestReleaseHandlerReturnsCanonicalAggregateAndHeaders(t *testing.T) {
	checklistID := uuid.New()
	etag := shared.MakeChecklistETag(checklistID, 7)
	stub := &serviceStub{
		releaseResult: &ReleaseMutationResult{
			Aggregate: shared.ChecklistAggregate{
				ID:          checklistID,
				SyncVersion: 7,
			},
		},
		releaseETag: etag,
	}
	router := communityTestRouter(stub, true)
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String()+
			"/community-releases/"+uuid.NewString(),
		nil,
	)
	request.Header.Set("If-Match", `"parent"`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, etag, response.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", response.Header().Get("Cache-Control"))
	require.Equal(t, checklistID.String(), responseDataID(t, response))
	require.Equal(t, 1, stub.releaseCalls)
}

func TestRetireHandlerReturnsTypedError(t *testing.T) {
	stub := &serviceStub{
		retireError: shared.NewInvalidTransition(
			"community source has not been released",
			nil,
		),
	}
	router := communityTestRouter(stub, true)
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/auth/user-pmcs/checklists/"+uuid.NewString()+
			"/community-source",
		nil,
	)
	request.Header.Set("If-Match", `"parent"`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, "invalid_transition", responseErrorCode(t, response))
	require.Equal(t, 1, stub.retireCalls)
}

func communityTestRouter(service Service, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	if authenticated {
		group.Use(func(context *gin.Context) {
			context.Set("user", &bootstrap.User{UserID: "owner-1"})
			context.Next()
		})
	}
	RegisterRoutes(group, service)
	return router
}

func responseErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	return body.Error.Code
}

func responseDataID(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	return body.Data.ID
}

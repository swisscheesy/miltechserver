package sync

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	ginGzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type serviceStub struct {
	result       *AccountDelta
	err          error
	calls        int
	capturedUser *bootstrap.User
	after        string
	limit        string
}

func (stub *serviceStub) GetDelta(
	_ context.Context,
	user *bootstrap.User,
	after string,
	limit string,
) (*AccountDelta, error) {
	stub.calls++
	stub.capturedUser = user
	stub.after = after
	stub.limit = limit
	return stub.result, stub.err
}

func TestAccountDeltaRejectsMissingAuthentication(t *testing.T) {
	stub := &serviceStub{}
	router := accountDeltaTestRouter(stub, false, shared.DefaultConfig())

	response := httptest.NewRecorder()
	router.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/v1/auth/user-pmcs/sync", nil),
	)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Equal(t, "authentication_required", responseErrorCode(t, response))
	require.Zero(t, stub.calls)
}

func TestAccountDeltaPassesRawPaginationToService(t *testing.T) {
	expected := &AccountDelta{
		FromCursor:     41,
		ThroughCursor:  46,
		AccountVersion: 49,
		HasMore:        true,
		Changes: []AccountChange{
			{
				AccountChangeVersion: 46,
				Kind:                 ChangeKindChecklist,
				ETag:                 `"opaque-current-root"`,
				Checklist: &shared.ChecklistAggregate{
					ID:          uuid.MustParse("60000000-0000-4000-8000-000000000001"),
					SyncVersion: 7,
				},
			},
		},
	}
	stub := &serviceStub{result: expected}
	router := accountDeltaTestRouter(stub, true, shared.DefaultConfig())

	response := httptest.NewRecorder()
	router.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/auth/user-pmcs/sync?after=41&limit=7",
			nil,
		),
	)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "41", stub.after)
	require.Equal(t, "7", stub.limit)
	require.Equal(t, "user-1", stub.capturedUser.UserID)
	require.Equal(t, "Accept-Encoding", response.Header().Get("Vary"))

	var envelope struct {
		Status int           `json:"status"`
		Data   *AccountDelta `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Equal(t, http.StatusOK, envelope.Status)
	require.Equal(t, expected, envelope.Data)
}

func TestAccountDeltaReturnsTypedServiceError(t *testing.T) {
	stub := &serviceStub{
		err: shared.NewInvalidRequest("invalid account delta cursor", nil),
	}
	router := accountDeltaTestRouter(stub, true, shared.DefaultConfig())

	response := httptest.NewRecorder()
	router.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/auth/user-pmcs/sync?after=-1",
			nil,
		),
	)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, "invalid_request", responseErrorCode(t, response))
	require.Equal(t, "Accept-Encoding", response.Header().Get("Vary"))
}

func TestAccountDeltaHidesInternalErrors(t *testing.T) {
	stub := &serviceStub{err: errors.New("database failed with private details")}
	router := accountDeltaTestRouter(stub, true, shared.DefaultConfig())

	response := httptest.NewRecorder()
	router.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/v1/auth/user-pmcs/sync", nil),
	)

	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, "internal_error", responseErrorCode(t, response))
	require.NotContains(t, response.Body.String(), "database")
}

func TestAccountDeltaSupportsGzipAndVary(t *testing.T) {
	stub := &serviceStub{result: &AccountDelta{Changes: []AccountChange{}}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{UserID: "user-1"})
		context.Next()
	})
	handler := Handler{service: stub, config: shared.DefaultConfig()}
	router.GET(
		"/api/v1/auth/user-pmcs/sync",
		ginGzip.Gzip(ginGzip.DefaultCompression),
		handler.getDelta,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/user-pmcs/sync",
		nil,
	)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
	require.Contains(t, response.Header().Values("Vary"), "Accept-Encoding")
	reader, err := gzip.NewReader(response.Body)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Contains(t, string(body), `"changes":[]`)
}

func TestAccountDeltaServiceValidatesCursorAndLimit(t *testing.T) {
	config := shared.DefaultConfig()
	repository := &repositoryStub{
		result: &AccountDelta{Changes: []AccountChange{}},
	}
	service := NewService(repository, config)
	user := &bootstrap.User{UserID: "user-1"}

	tests := []struct {
		name      string
		after     string
		limit     string
		wantAfter int64
		wantLimit int
		wantError bool
	}{
		{name: "defaults", wantLimit: config.DeltaDefaultLimit},
		{name: "valid", after: "17", limit: "25", wantAfter: 17, wantLimit: 25},
		{name: "negative cursor", after: "-1", wantError: true},
		{name: "malformed cursor", after: "abc", wantError: true},
		{
			name:      "overflow cursor",
			after:     "9223372036854775808",
			wantError: true,
		},
		{name: "zero limit", limit: "0", wantError: true},
		{name: "negative limit", limit: "-1", wantError: true},
		{name: "malformed limit", limit: "abc", wantError: true},
		{
			name:      "over maximum limit",
			limit:     "26",
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository.calls = 0
			_, err := service.GetDelta(
				context.Background(),
				user,
				test.after,
				test.limit,
			)
			if test.wantError {
				var apiError *shared.APIError
				require.ErrorAs(t, err, &apiError)
				require.Equal(t, http.StatusBadRequest, apiError.Status)
				require.Equal(t, "invalid_request", apiError.Code)
				require.Zero(t, repository.calls)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantAfter, repository.after)
			require.Equal(t, test.wantLimit, repository.limit)
			require.Equal(t, config.MaxDeltaResponseBytes, repository.byteLimit)
		})
	}
}

func TestAccountDeltaRejectsMissingSelectedChecklistTree(t *testing.T) {
	tests := []struct {
		name      string
		entry     loadedChecklist
		wantError string
	}{
		{
			name: "draft",
			entry: loadedChecklist{
				aggregate: shared.ChecklistAggregate{ID: uuid.New()},
				draftID: uuid.NullUUID{
					UUID:  uuid.New(),
					Valid: true,
				},
			},
			wantError: "draft revision tree disappeared",
		},
		{
			name: "publication",
			entry: loadedChecklist{
				aggregate: shared.ChecklistAggregate{ID: uuid.New()},
				publicationID: uuid.NullUUID{
					UUID:  uuid.New(),
					Valid: true,
				},
			},
			wantError: "publication revision tree disappeared",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := test.entry
			err := attachChecklistTrees(
				&entry,
				map[uuid.UUID]shared.Revision{},
			)

			require.ErrorContains(t, err, test.wantError)
			require.Nil(t, entry.aggregate.Draft)
			require.Nil(t, entry.aggregate.Publication)
		})
	}
}

type repositoryStub struct {
	result    *AccountDelta
	err       error
	calls     int
	userUID   string
	after     int64
	limit     int
	byteLimit int
}

func (stub *repositoryStub) GetDelta(
	_ context.Context,
	userUID string,
	after int64,
	limit int,
	byteLimit int,
) (*AccountDelta, error) {
	stub.calls++
	stub.userUID = userUID
	stub.after = after
	stub.limit = limit
	stub.byteLimit = byteLimit
	return stub.result, stub.err
}

func accountDeltaTestRouter(
	service Service,
	authenticated bool,
	config shared.Config,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if authenticated {
		router.Use(func(context *gin.Context) {
			context.Set("user", &bootstrap.User{UserID: "user-1"})
			context.Next()
		})
	}
	handler := Handler{service: service, config: config}
	router.GET("/api/v1/auth/user-pmcs/sync", handler.getDelta)
	return router
}

func responseErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	return envelope.Error.Code
}

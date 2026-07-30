package owned

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type serviceStub struct {
	getResult      *shared.ChecklistAggregate
	getETag        string
	getError       error
	revisionResult *HistoricalRevision
	revisionETag   string
	revisionError  error
	mutationResult *MutationResult
	mutationETag   string
	mutationError  error
	createCalls    int
	putDraftCalls  int
	deleteCalls    int
	publishCalls   int
}

func (stub *serviceStub) GetRevision(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ string,
) (*HistoricalRevision, string, error) {
	return stub.revisionResult, stub.revisionETag, stub.revisionError
}

func (stub *serviceStub) Get(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
) (*shared.ChecklistAggregate, string, error) {
	return stub.getResult, stub.getETag, stub.getError
}

func (stub *serviceStub) Create(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ shared.RevisionInput,
	_ string,
) (*MutationResult, string, error) {
	stub.createCalls++
	return stub.mutationResult, stub.mutationETag, stub.mutationError
}

func (stub *serviceStub) PutDraft(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ string,
	_ shared.RevisionInput,
	_ string,
) (*MutationResult, string, error) {
	stub.putDraftCalls++
	return stub.mutationResult, stub.mutationETag, stub.mutationError
}

func (stub *serviceStub) DeleteDraft(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ string,
	_ string,
) (*MutationResult, string, error) {
	stub.deleteCalls++
	return stub.mutationResult, stub.mutationETag, stub.mutationError
}

func (stub *serviceStub) Publish(
	_ context.Context,
	_ *bootstrap.User,
	_ string,
	_ string,
	_ shared.RevisionInput,
	_ string,
) (*MutationResult, string, error) {
	stub.publishCalls++
	return stub.mutationResult, stub.mutationETag, stub.mutationError
}

func TestCreateRejectsMissingAuthenticationWithTypedError(t *testing.T) {
	stub := &serviceStub{}
	router := newHandlerTestRouter(stub, false, shared.DefaultConfig())
	request := jsonRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+uuid.NewString(),
		validDraftInput(uuid.New()),
	)
	request.Header.Set("If-None-Match", "*")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	requireErrorCode(t, response.Body.Bytes(), "authentication_required")
	require.Zero(t, stub.createCalls)
}

func TestCreateRejectsStrictJSONViolations(t *testing.T) {
	config := shared.DefaultConfig()
	checklistID := uuid.NewString()
	tests := []struct {
		name          string
		body          string
		contentType   string
		contentEncode string
		wantStatus    int
		wantCode      string
	}{
		{
			name:        "wrong media type",
			body:        `{}`,
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:          "compressed body",
			body:          `{}`,
			contentType:   "application/json",
			contentEncode: "gzip",
			wantStatus:    http.StatusBadRequest,
			wantCode:      "invalid_request",
		},
		{
			name:        "unknown field",
			body:        `{"id":"` + uuid.NewString() + `","name":"","description":"","models":[],"sections":[],"extra":true}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "trailing JSON",
			body:        `{} {}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
		},
		{
			name:        "raw body over ceiling",
			body:        strings.Repeat(" ", int(config.MaxMutationBodyBytes)+1),
			contentType: "application/json",
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    "content_too_large",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := &serviceStub{}
			router := newHandlerTestRouter(stub, true, config)
			request := httptest.NewRequest(
				http.MethodPut,
				"/api/v1/auth/user-pmcs/checklists/"+checklistID,
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", test.contentType)
			request.Header.Set("If-None-Match", "*")
			if test.contentEncode != "" {
				request.Header.Set("Content-Encoding", test.contentEncode)
			}

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, test.wantStatus, response.Code)
			requireErrorCode(t, response.Body.Bytes(), test.wantCode)
			require.Zero(t, stub.createCalls)
		})
	}
}

func TestCreateReturns201WithCanonicalHeadersAndEnvelope(t *testing.T) {
	checklistID := uuid.New()
	aggregate := shared.ChecklistAggregate{ID: checklistID, SyncVersion: 1}
	etag := shared.MakeChecklistETag(checklistID, 1)
	stub := &serviceStub{
		mutationResult: &MutationResult{Aggregate: aggregate, Created: true},
		mutationETag:   etag,
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := jsonRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String(),
		validDraftInput(uuid.New()),
	)
	request.Header.Set("If-None-Match", "*")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.Equal(t, etag, response.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", response.Header().Get("Cache-Control"))
	requireStandardEnvelopeDataID(t, response.Body.Bytes(), checklistID)
}

func TestCreateIdempotentRetryReturns200(t *testing.T) {
	checklistID := uuid.New()
	aggregate := shared.ChecklistAggregate{ID: checklistID, SyncVersion: 1}
	stub := &serviceStub{
		mutationResult: &MutationResult{
			Aggregate:  aggregate,
			Idempotent: true,
		},
		mutationETag: shared.MakeChecklistETag(checklistID, 1),
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := jsonRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String(),
		validDraftInput(uuid.New()),
	)
	request.Header.Set("If-None-Match", "*")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
}

func TestGetOwnedMatchingIfNoneMatchReturnsBodyless304(t *testing.T) {
	checklistID := uuid.New()
	etag := shared.MakeChecklistETag(checklistID, 3)
	stub := &serviceStub{
		getResult: &shared.ChecklistAggregate{ID: checklistID, SyncVersion: 3},
		getETag:   etag,
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String(),
		nil,
	)
	request.Header.Set("If-None-Match", etag)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNotModified, response.Code)
	require.Empty(t, response.Body.Bytes())
	require.Equal(t, etag, response.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", response.Header().Get("Cache-Control"))
}

func TestGetOwnedRejectsMalformedIfNoneMatch(t *testing.T) {
	checklistID := uuid.New()
	stub := &serviceStub{
		getResult: &shared.ChecklistAggregate{
			ID:          checklistID,
			SyncVersion: 3,
		},
		getETag: shared.MakeChecklistETag(checklistID, 3),
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String(),
		nil,
	)
	request.Header.Set("If-None-Match", "*")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	requireErrorCode(t, response.Body.Bytes(), "invalid_precondition")
}

func TestPutDraftRequiresParentChecklistETag(t *testing.T) {
	stub := &serviceStub{
		mutationError: shared.NewPreconditionRequired(
			"If-Match header is required",
			nil,
		),
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	checklistID := uuid.NewString()
	revisionID := uuid.New()
	request := jsonRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID+
			"/drafts/"+revisionID.String(),
		validDraftInput(revisionID),
	)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusPreconditionRequired, response.Code)
	requireErrorCode(t, response.Body.Bytes(), "precondition_required")
}

func TestDeleteDraftWritesTypedTransitionError(t *testing.T) {
	stub := &serviceStub{
		mutationError: shared.NewInvalidTransition(
			"a current publication is required",
			nil,
		),
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/auth/user-pmcs/checklists/"+uuid.NewString()+
			"/drafts/"+uuid.NewString(),
		nil,
	)
	request.Header.Set("If-Match", `"current"`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusConflict, response.Code)
	requireErrorCode(t, response.Body.Bytes(), "invalid_transition")
}

func TestPublishReturnsCurrentAggregateAndOwnedHeaders(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	aggregate := shared.ChecklistAggregate{ID: checklistID, SyncVersion: 2}
	etag := shared.MakeChecklistETag(checklistID, 2)
	stub := &serviceStub{
		mutationResult: &MutationResult{Aggregate: aggregate},
		mutationETag:   etag,
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := jsonRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String()+
			"/publications/"+revisionID.String(),
		completePublicationInput(revisionID, 1),
	)
	request.Header.Set("If-Match", `"current"`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, etag, response.Header().Get("ETag"))
	require.Equal(t, ownedCacheControl, response.Header().Get("Cache-Control"))
	require.Equal(t, 1, stub.publishCalls)
}

func TestHistoricalMatchingIfNoneMatchReturnsBodylessImmutable304(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	etag := `"immutable-history"`
	stub := &serviceStub{
		revisionResult: &HistoricalRevision{ID: revisionID},
		revisionETag:   etag,
	}
	router := newHandlerTestRouter(stub, true, shared.DefaultConfig())
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String()+
			"/revisions/"+revisionID.String(),
		nil,
	)
	request.Header.Set("If-None-Match", etag)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNotModified, response.Code)
	require.Empty(t, response.Body.Bytes())
	require.Equal(t, etag, response.Header().Get("ETag"))
	require.Equal(
		t,
		"private, max-age=31536000, immutable",
		response.Header().Get("Cache-Control"),
	)
}

func newHandlerTestRouter(
	service Service,
	authenticated bool,
	config shared.Config,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if authenticated {
		router.Use(func(context *gin.Context) {
			context.Set("user", &bootstrap.User{UserID: "owner-1"})
			context.Next()
		})
	}
	RegisterRoutes(router.Group("/api/v1/auth"), service, config)
	return router
}

func jsonRequest(
	method string,
	path string,
	body any,
) *http.Request {
	payload, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func requireErrorCode(t *testing.T, body []byte, want string) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.Equal(t, want, envelope.Error.Code)
}

func requireStandardEnvelopeDataID(
	t *testing.T,
	body []byte,
	want uuid.UUID,
) {
	t.Helper()
	var envelope struct {
		Status int `json:"status"`
		Data   struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.Equal(t, http.StatusCreated, envelope.Status)
	require.Equal(t, want, envelope.Data.ID)
}

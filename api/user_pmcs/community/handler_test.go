package community

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	browseResult  *shared.CommunityPage
	browseError   error
	detailResult  *shared.PublicChecklistRelease
	detailETag    string
	detailError   error
	releaseCalls  int
	retireCalls   int
	browseCalls   int
	detailCalls   int
	browseAfter   string
	browseLimit   string
	browseModel   string
	detailID      string
}

type publicRepositoryStub struct {
	browseResult *shared.CommunityPage
	browseError  error
	detailResult *shared.PublicChecklistRelease
	detailError  error
	browseCalls  int
	detailCalls  int
	browseFilter shared.CommunityBrowseFilter
	detailID     uuid.UUID
}

func (stub *publicRepositoryStub) Release(
	context.Context,
	string,
	uuid.UUID,
	uuid.UUID,
	shared.Precondition,
) (*ReleaseMutationResult, error) {
	return nil, nil
}

func (stub *publicRepositoryStub) Retire(
	context.Context,
	string,
	uuid.UUID,
	shared.Precondition,
) (*ReleaseMutationResult, error) {
	return nil, nil
}

func (stub *publicRepositoryStub) Browse(
	_ context.Context,
	filter shared.CommunityBrowseFilter,
) (*shared.CommunityPage, error) {
	stub.browseCalls++
	stub.browseFilter = filter
	return stub.browseResult, stub.browseError
}

func (stub *publicRepositoryStub) GetCurrentRelease(
	_ context.Context,
	checklistID uuid.UUID,
) (*shared.PublicChecklistRelease, error) {
	stub.detailCalls++
	stub.detailID = checklistID
	return stub.detailResult, stub.detailError
}

func (stub *repositoryStub) Browse(
	context.Context,
	shared.CommunityBrowseFilter,
) (*shared.CommunityPage, error) {
	return nil, nil
}

func (stub *repositoryStub) GetCurrentRelease(
	context.Context,
	uuid.UUID,
) (*shared.PublicChecklistRelease, error) {
	return nil, nil
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

func (stub *serviceStub) Browse(
	_ context.Context,
	after string,
	limit string,
	model string,
) (*shared.CommunityPage, error) {
	stub.browseCalls++
	stub.browseAfter = after
	stub.browseLimit = limit
	stub.browseModel = model
	return stub.browseResult, stub.browseError
}

func (stub *serviceStub) GetCurrentRelease(
	_ context.Context,
	checklistID string,
) (*shared.PublicChecklistRelease, string, error) {
	stub.detailCalls++
	stub.detailID = checklistID
	return stub.detailResult, stub.detailETag, stub.detailError
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

func TestCommunityBrowseHandlerIsAnonymousAndGzipped(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	stub := &serviceStub{
		browseResult: &shared.CommunityPage{
			Items: []shared.PublicCommunitySummary{{
				ChecklistID:        checklistID,
				RevisionID:         revisionID,
				RevisionNumber:     2,
				Name:               "Public checklist",
				Description:        "Current release",
				Models:             []shared.ModelValue{},
				CreatorDisplayName: "Public creator",
				ReleasedAt:         now,
				UpdatedAt:          now,
			}},
		},
	}
	router := communityPublicTestRouter(stub)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community?after=cursor&limit=7&model=M1",
		nil,
	)
	request.Header.Set("Accept-Encoding", "gzip")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
	require.Equal(t, "Accept-Encoding", response.Header().Get("Vary"))
	require.Equal(t, 1, stub.browseCalls)
	require.Equal(t, "cursor", stub.browseAfter)
	require.Equal(t, "7", stub.browseLimit)
	require.Equal(t, "M1", stub.browseModel)
	body := gunzipResponse(t, response)
	require.NotContains(t, string(body), "owner_uid")
	require.NotContains(t, string(body), "email")
	require.Contains(t, string(body), `"creator_display_name":"Public creator"`)
}

func TestCommunityDetailHandlerUsesPublicCacheAndConditionalGET(t *testing.T) {
	checklistID := uuid.New()
	etag := `"public-release-validator"`
	stub := &serviceStub{
		detailResult: &shared.PublicChecklistRelease{
			ChecklistID:        checklistID,
			CreatorDisplayName: "Deleted user",
			Revision: shared.Revision{
				ID:       uuid.New(),
				Name:     "Current public revision",
				Models:   []shared.ModelValue{},
				Sections: []shared.Section{},
			},
		},
		detailETag: etag,
	}
	router := communityPublicTestRouter(stub)

	firstRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community/"+checklistID.String(),
		nil,
	)
	firstRequest.Header.Set("Accept-Encoding", "gzip")
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)

	require.Equal(t, http.StatusOK, firstResponse.Code)
	require.Equal(t, etag, firstResponse.Header().Get("ETag"))
	require.Equal(t, "public, no-cache", firstResponse.Header().Get("Cache-Control"))
	require.Equal(t, "Accept-Encoding", firstResponse.Header().Get("Vary"))
	require.Equal(t, "gzip", firstResponse.Header().Get("Content-Encoding"))
	require.Contains(t, string(gunzipResponse(t, firstResponse)), `"creator_display_name":"Deleted user"`)

	conditionalRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community/"+checklistID.String(),
		nil,
	)
	conditionalRequest.Header.Set(
		"If-None-Match",
		`W/"other", W/`+etag+`, "third"`,
	)
	conditionalResponse := httptest.NewRecorder()
	router.ServeHTTP(conditionalResponse, conditionalRequest)

	require.Equal(t, http.StatusNotModified, conditionalResponse.Code)
	require.Empty(t, conditionalResponse.Body.Bytes())
	require.Equal(t, etag, conditionalResponse.Header().Get("ETag"))
	require.Equal(t, "public, no-cache", conditionalResponse.Header().Get("Cache-Control"))

	wildcardRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community/"+checklistID.String(),
		nil,
	)
	wildcardRequest.Header.Set("If-None-Match", "*")
	wildcardResponse := httptest.NewRecorder()
	router.ServeHTTP(wildcardResponse, wildcardRequest)
	require.Equal(t, http.StatusNotModified, wildcardResponse.Code)
	require.Empty(t, wildcardResponse.Body.Bytes())
	require.Equal(t, 3, stub.detailCalls)
}

func TestCommunityPublicHandlersReturnTypedErrors(t *testing.T) {
	stub := &serviceStub{
		browseError: shared.NewInvalidRequest("invalid community cursor", nil),
		detailError: shared.NewResourceNotFound("community checklist not found", nil),
	}
	router := communityPublicTestRouter(stub)

	browseResponse := httptest.NewRecorder()
	router.ServeHTTP(
		browseResponse,
		httptest.NewRequest(http.MethodGet, "/api/v1/user-pmcs/community", nil),
	)
	require.Equal(t, http.StatusBadRequest, browseResponse.Code)
	require.Equal(t, "invalid_request", responseErrorCode(t, browseResponse))

	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(
		detailResponse,
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/user-pmcs/community/"+uuid.NewString(),
			nil,
		),
	)
	require.Equal(t, http.StatusNotFound, detailResponse.Code)
	require.Equal(t, "resource_not_found", responseErrorCode(t, detailResponse))
}

func TestCommunityBrowseServiceDefaultsValidatesAndNormalizes(t *testing.T) {
	config := shared.DefaultConfig()
	cursor := shared.CommunityCursor{
		Version:   1,
		UpdatedAt: time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC),
		Checklist: uuid.New(),
	}
	encodedCursor, err := shared.EncodeCommunityCursor(cursor)
	require.NoError(t, err)

	repository := &publicRepositoryStub{
		browseResult: &shared.CommunityPage{Items: []shared.PublicCommunitySummary{}},
	}
	service := NewService(repository, config)
	_, err = service.Browse(
		context.Background(),
		encodedCursor,
		"",
		"  M998\u00a0 HMMWV  ",
	)
	require.NoError(t, err)
	require.Equal(t, 1, repository.browseCalls)
	require.Equal(t, config.CommunityDefaultLimit, repository.browseFilter.Limit)
	require.Equal(t, "m998 hmmwv", repository.browseFilter.NormalizedModel)
	require.NotNil(t, repository.browseFilter.After)
	require.Equal(t, cursor, *repository.browseFilter.After)

	for _, test := range []struct {
		name  string
		after string
		limit string
	}{
		{name: "malformed cursor", after: "not-a-cursor"},
		{name: "non-numeric limit", limit: "many"},
		{name: "zero limit", limit: "0"},
		{name: "over maximum", limit: "51"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalidRepository := &publicRepositoryStub{}
			invalidService := NewService(invalidRepository, config)
			_, browseErr := invalidService.Browse(
				context.Background(),
				test.after,
				test.limit,
				"",
			)
			requireCommunityAPIError(
				t,
				browseErr,
				http.StatusBadRequest,
				"invalid_request",
			)
			require.Zero(t, invalidRepository.browseCalls)
		})
	}

	maxRepository := &publicRepositoryStub{
		browseResult: &shared.CommunityPage{Items: []shared.PublicCommunitySummary{}},
	}
	maxService := NewService(maxRepository, config)
	_, err = maxService.Browse(
		context.Background(),
		"",
		"50",
		"",
	)
	require.NoError(t, err)
	require.Equal(t, config.CommunityMaxLimit, maxRepository.browseFilter.Limit)
}

func TestPublicReleaseETagIncludesChecklistRevisionAndContentHash(t *testing.T) {
	checklistID := uuid.New()
	revisionID := uuid.New()
	var contentHash [32]byte
	contentHash[0] = 1
	baseline := makePublicReleaseETag(checklistID, revisionID, contentHash)

	require.NotEqual(
		t,
		baseline,
		makePublicReleaseETag(uuid.New(), revisionID, contentHash),
	)
	require.NotEqual(
		t,
		baseline,
		makePublicReleaseETag(checklistID, uuid.New(), contentHash),
	)
	contentHash[31] = 2
	require.NotEqual(
		t,
		baseline,
		makePublicReleaseETag(checklistID, revisionID, contentHash),
	)
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

func communityPublicTestRouter(service Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterPublicRoutes(router.Group("/api/v1"), service)
	return router
}

func gunzipResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	return body
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

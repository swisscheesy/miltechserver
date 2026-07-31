package user_pmcs_test

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

	userpmcs "miltechserver/api/user_pmcs"
	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
	"miltechserver/bootstrap"
)

type userPmcsRouteContract struct {
	name   string
	method string
	path   string
}

func TestHTTPContractEveryAuthenticatedRouteRejectsAbsentAndMalformedAuth(
	t *testing.T,
) {
	router := newUserPmcsContractRouter(shared.DefaultConfig())
	checklistID := uuid.NewString()
	revisionID := uuid.NewString()
	routes := []userPmcsRouteContract{
		{"get checklist", http.MethodGet, "/auth/user-pmcs/checklists/" + checklistID},
		{"create checklist", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID},
		{"delete checklist", http.MethodDelete, "/auth/user-pmcs/checklists/" + checklistID},
		{"put draft", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID + "/drafts/" + revisionID},
		{"delete draft", http.MethodDelete, "/auth/user-pmcs/checklists/" + checklistID + "/drafts/" + revisionID},
		{"publish", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID + "/publications/" + revisionID},
		{"get revision", http.MethodGet, "/auth/user-pmcs/checklists/" + checklistID + "/revisions/" + revisionID},
		{"sync", http.MethodGet, "/auth/user-pmcs/sync"},
		{"release", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID + "/community-releases/" + revisionID},
		{"retire", http.MethodDelete, "/auth/user-pmcs/checklists/" + checklistID + "/community-source"},
		{"updates", http.MethodGet, "/auth/user-pmcs/subscriptions/updates"},
		{"install", http.MethodPut, "/auth/user-pmcs/subscriptions/" + checklistID},
		{"unsubscribe", http.MethodDelete, "/auth/user-pmcs/subscriptions/" + checklistID},
		{"accept update", http.MethodPut, "/auth/user-pmcs/subscriptions/" + checklistID + "/installed-releases/" + revisionID},
		{"installed release", http.MethodGet, "/auth/user-pmcs/subscriptions/" + checklistID + "/installed-releases/" + revisionID},
	}

	for _, route := range routes {
		route := route
		for _, auth := range []string{"absent", "malformed"} {
			auth := auth
			t.Run(route.name+"/"+auth, func(t *testing.T) {
				request := httptest.NewRequest(route.method, route.path, nil)
				if auth == "malformed" {
					request.Header.Set("X-Test-Auth", "malformed")
				}
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				requireStableAPIError(
					t,
					response,
					http.StatusUnauthorized,
					"authentication_required",
				)
			})
		}
	}
}

func TestHTTPContractUnknownResourcesUseSafeStable404s(t *testing.T) {
	userUID := newUserPmcsTestUser(t)
	router := newUserPmcsContractRouter(shared.DefaultConfig())
	checklistID := uuid.New()
	revisionID := uuid.New()
	draft := preparedTree(t, revisionID).Input
	publication := draft
	number := int32(1)
	publication.RevisionNumber = &number
	draftBody := mustJSON(t, draft)
	publicationBody := mustJSON(t, publication)
	stale := `"unknown-resource"`

	cases := []struct {
		userPmcsRouteContract
		body    []byte
		headers map[string]string
	}{
		{userPmcsRouteContract{"get checklist", http.MethodGet, "/auth/user-pmcs/checklists/" + checklistID.String()}, nil, nil},
		{userPmcsRouteContract{"delete checklist", http.MethodDelete, "/auth/user-pmcs/checklists/" + checklistID.String()}, nil, map[string]string{"If-Match": stale}},
		{userPmcsRouteContract{"put draft", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String()}, draftBody, map[string]string{"Content-Type": "application/json", "If-Match": stale}},
		{userPmcsRouteContract{"delete draft", http.MethodDelete, "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String()}, nil, map[string]string{"If-Match": stale}},
		{userPmcsRouteContract{"publish", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID.String() + "/publications/" + revisionID.String()}, publicationBody, map[string]string{"Content-Type": "application/json", "If-Match": stale}},
		{userPmcsRouteContract{"get revision", http.MethodGet, "/auth/user-pmcs/checklists/" + checklistID.String() + "/revisions/" + revisionID.String()}, nil, nil},
		{userPmcsRouteContract{"release", http.MethodPut, "/auth/user-pmcs/checklists/" + checklistID.String() + "/community-releases/" + revisionID.String()}, nil, map[string]string{"If-Match": stale}},
		{userPmcsRouteContract{"retire", http.MethodDelete, "/auth/user-pmcs/checklists/" + checklistID.String() + "/community-source"}, nil, map[string]string{"If-Match": stale}},
		{userPmcsRouteContract{"install", http.MethodPut, "/auth/user-pmcs/subscriptions/" + checklistID.String()}, nil, map[string]string{"If-None-Match": "*"}},
		{userPmcsRouteContract{"unsubscribe", http.MethodDelete, "/auth/user-pmcs/subscriptions/" + checklistID.String()}, nil, map[string]string{"If-Match": stale}},
		{userPmcsRouteContract{"accept update", http.MethodPut, "/auth/user-pmcs/subscriptions/" + checklistID.String() + "/installed-releases/" + revisionID.String()}, nil, map[string]string{"If-Match": stale}},
		{userPmcsRouteContract{"installed release", http.MethodGet, "/auth/user-pmcs/subscriptions/" + checklistID.String() + "/installed-releases/" + revisionID.String()}, nil, nil},
		{userPmcsRouteContract{"public release", http.MethodGet, "/user-pmcs/community/" + checklistID.String()}, nil, nil},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			response := performContractRequest(
				router,
				testCase.method,
				testCase.path,
				userUID,
				testCase.body,
				testCase.headers,
			)
			requireStableAPIError(
				t,
				response,
				http.StatusNotFound,
				"resource_not_found",
			)
		})
	}

	browse := performContractRequest(
		router,
		http.MethodGet,
		"/user-pmcs/community",
		"",
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, browse.Code)

	updates := performContractRequest(
		router,
		http.MethodGet,
		"/auth/user-pmcs/subscriptions/updates",
		userUID,
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, updates.Code)
}

func TestHTTPContractMutationBodyContentTypeAndConditionalFailures(
	t *testing.T,
) {
	userUID := newUserPmcsTestUser(t)
	config := shared.DefaultConfig()
	router := newUserPmcsContractRouter(config)
	checklistID := uuid.New()
	revisionID := uuid.New()
	draft := preparedTree(t, revisionID).Input
	publication := draft
	number := int32(1)
	publication.RevisionNumber = &number

	cases := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "create rejects absent content type",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String(),
			body:       mustJSON(t, draft),
			headers:    map[string]string{"If-None-Match": "*"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "create rejects malformed body",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String(),
			body:       []byte(`{"id":`),
			headers:    map[string]string{"Content-Type": "application/json", "If-None-Match": "*"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "create requires condition",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String(),
			body:       mustJSON(t, draft),
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "put draft requires condition",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String(),
			body:       mustJSON(t, draft),
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "publish requires condition",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/publications/" + revisionID.String(),
			body:       mustJSON(t, publication),
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "delete checklist requires condition",
			method:     http.MethodDelete,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String(),
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "delete draft requires condition",
			method:     http.MethodDelete,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String(),
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "release requires condition",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/community-releases/" + revisionID.String(),
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "retire requires condition",
			method:     http.MethodDelete,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/community-source",
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "install requires condition",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/subscriptions/" + checklistID.String(),
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "unsubscribe requires condition",
			method:     http.MethodDelete,
			path:       "/auth/user-pmcs/subscriptions/" + checklistID.String(),
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
		{
			name:       "accept update requires condition",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/subscriptions/" + checklistID.String() + "/installed-releases/" + revisionID.String(),
			wantStatus: http.StatusPreconditionRequired,
			wantCode:   "precondition_required",
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			response := performContractRequest(
				router,
				testCase.method,
				testCase.path,
				userUID,
				testCase.body,
				testCase.headers,
			)
			requireStableAPIError(
				t,
				response,
				testCase.wantStatus,
				testCase.wantCode,
			)
		})
	}

	smallConfig := config
	smallConfig.MaxMutationBodyBytes = 64
	smallRouter := newUserPmcsContractRouter(smallConfig)
	oversized := performContractRequest(
		smallRouter,
		http.MethodPut,
		"/auth/user-pmcs/checklists/"+uuid.NewString(),
		userUID,
		bytes.Repeat([]byte{'x'}, 65),
		map[string]string{
			"Content-Type":  "application/json",
			"If-None-Match": "*",
		},
	)
	requireStableAPIError(
		t,
		oversized,
		http.StatusRequestEntityTooLarge,
		"content_too_large",
	)
}

func TestAuthorizationOwnerCannotSubscribeAndCrossOwnerIsHidden(
	t *testing.T,
) {
	ctx := context.Background()
	fixture := newReleasedChecklistFixture(t, 1)
	released, err := fixture.repository.Release(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(
			fixture.checklist,
			fixture.aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)

	subscriptionRepository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 3),
		shared.DefaultConfig(),
	)
	_, err = subscriptionRepository.Install(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	requireAPIIntegrationError(t, err, 409, "invalid_transition")

	otherUID := newUserPmcsTestUser(t)
	router := newUserPmcsContractRouter(shared.DefaultConfig())
	crossDraft := preparedTree(t, uuid.New()).Input
	crossPublication := crossDraft
	crossNumber := int32(2)
	crossPublication.RevisionNumber = &crossNumber
	paths := []struct {
		userPmcsRouteContract
		body    []byte
		headers map[string]string
	}{
		{userPmcsRouteContract{"get", http.MethodGet, "/auth/user-pmcs/checklists/" + fixture.checklist.String()}, nil, nil},
		{userPmcsRouteContract{"create collision", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String()}, mustJSON(t, crossDraft), map[string]string{"Content-Type": "application/json", "If-None-Match": "*"}},
		{userPmcsRouteContract{"put draft", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/drafts/" + crossDraft.ID.String()}, mustJSON(t, crossDraft), map[string]string{"Content-Type": "application/json", "If-Match": shared.MakeChecklistETag(fixture.checklist, released.Aggregate.SyncVersion)}},
		{userPmcsRouteContract{"delete draft", http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/drafts/" + crossDraft.ID.String()}, nil, map[string]string{"If-Match": shared.MakeChecklistETag(fixture.checklist, released.Aggregate.SyncVersion)}},
		{userPmcsRouteContract{"publish", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/publications/" + crossDraft.ID.String()}, mustJSON(t, crossPublication), map[string]string{"Content-Type": "application/json", "If-Match": shared.MakeChecklistETag(fixture.checklist, released.Aggregate.SyncVersion)}},
		{userPmcsRouteContract{"delete", http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String()}, nil, map[string]string{"If-Match": shared.MakeChecklistETag(fixture.checklist, released.Aggregate.SyncVersion)}},
		{userPmcsRouteContract{"revision", http.MethodGet, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/revisions/" + fixture.revisions[0].Input.ID.String()}, nil, nil},
		{userPmcsRouteContract{"release", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/community-releases/" + fixture.revisions[0].Input.ID.String()}, nil, map[string]string{"If-Match": shared.MakeChecklistETag(fixture.checklist, released.Aggregate.SyncVersion)}},
		{userPmcsRouteContract{"retire", http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/community-source"}, nil, map[string]string{"If-Match": shared.MakeChecklistETag(fixture.checklist, released.Aggregate.SyncVersion)}},
		{userPmcsRouteContract{"unsubscribe without subscription", http.MethodDelete, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String()}, nil, map[string]string{"If-Match": `"missing"`}},
		{userPmcsRouteContract{"accept without subscription", http.MethodPut, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String() + "/installed-releases/" + fixture.revisions[0].Input.ID.String()}, nil, map[string]string{"If-Match": `"missing"`}},
		{userPmcsRouteContract{"read without subscription", http.MethodGet, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String() + "/installed-releases/" + fixture.revisions[0].Input.ID.String()}, nil, nil},
	}
	for _, path := range paths {
		response := performContractRequest(
			router,
			path.method,
			path.path,
			otherUID,
			path.body,
			path.headers,
		)
		requireStableAPIError(
			t,
			response,
			http.StatusNotFound,
			"resource_not_found",
		)
	}
}

func TestAuthorizationTombstoneRemainsPrivateAndContentFree(t *testing.T) {
	ctx := context.Background()
	ownerUID := newUserPmcsTestUser(t)
	otherUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		ctx,
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	deleted, err := repository.DeleteChecklist(
		ctx,
		ownerUID,
		checklistID,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	require.NotNil(t, deleted.Aggregate.DeletedAt)
	require.Nil(t, deleted.Aggregate.Draft)
	require.Nil(t, deleted.Aggregate.Publication)
	require.Nil(t, deleted.Aggregate.Community)

	router := newUserPmcsContractRouter(shared.DefaultConfig())
	ownerResponse := performContractRequest(
		router,
		http.MethodGet,
		"/auth/user-pmcs/checklists/"+checklistID.String(),
		ownerUID,
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, ownerResponse.Code)
	require.NotContains(t, ownerResponse.Body.String(), `"sections"`)
	require.NotContains(t, ownerResponse.Body.String(), draft.Input.Name)

	for name, response := range map[string]*httptest.ResponseRecorder{
		"cross owner": performContractRequest(
			router,
			http.MethodGet,
			"/auth/user-pmcs/checklists/"+checklistID.String(),
			otherUID,
			nil,
			nil,
		),
		"public": performContractRequest(
			router,
			http.MethodGet,
			"/user-pmcs/community/"+checklistID.String(),
			"",
			nil,
			nil,
		),
	} {
		t.Run(name, func(t *testing.T) {
			requireStableAPIError(
				t,
				response,
				http.StatusNotFound,
				"resource_not_found",
			)
		})
	}
}

func TestAuthorizationPublicVisibilityNoticeInvariantAndCurrentCreatorName(
	t *testing.T,
) {
	ctx := context.Background()
	fixture := newReleasedChecklistFixture(t, 2)
	released, err := fixture.repository.Release(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(
			fixture.checklist,
			fixture.aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)

	publicRepository := community.NewRepository(
		persistence.NewStore(testDB, 3),
		shared.DefaultConfig(),
	)
	current, err := publicRepository.GetCurrentRelease(ctx, fixture.checklist)
	require.NoError(t, err)
	require.Equal(t, fixture.revisions[0].Input.ID, current.Revision.ID)
	require.NotEqual(t, fixture.revisions[1].Input.ID, current.Revision.ID)
	privateDraft := preparedTree(t, uuid.New())
	_, err = newOwnedRepository().PutDraft(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		privateDraft,
		checklistPrecondition(
			fixture.checklist,
			released.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	current, err = publicRepository.GetCurrentRelease(ctx, fixture.checklist)
	require.NoError(t, err)
	require.NotEqual(t, privateDraft.Input.ID, current.Revision.ID)

	draft := preparedTree(t, uuid.New()).Input
	draft.Sections[0].Items[0].Notices[0].Type = nil
	number := int32(3)
	draft.RevisionNumber = &number
	_, err = shared.PreparePublication(draft, shared.DefaultConfig())
	requireAPIIntegrationError(t, err, 422, "validation_failed")

	updatedName := "renamed-" + uuid.NewString()[:8]
	_, err = testDB.ExecContext(
		ctx,
		`UPDATE users SET username = $1 WHERE uid = $2`,
		updatedName,
		fixture.ownerUID,
	)
	require.NoError(t, err)
	current, err = publicRepository.GetCurrentRelease(ctx, fixture.checklist)
	require.NoError(t, err)
	require.Equal(t, updatedName, current.CreatorDisplayName)
	require.NotNil(t, current.Revision.RevisionNumber)
	require.Equal(t, int32(1), *current.Revision.RevisionNumber)
	require.NotNil(t, released.Aggregate.Community.CurrentReleaseRevisionID)
	require.Equal(
		t,
		*released.Aggregate.Community.CurrentReleaseRevisionID,
		current.Revision.ID,
	)
}

func newUserPmcsContractRouter(config shared.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	config.PublicRequestsPerSecond = 1_000_000
	config.PublicRequestBurst = 1_000_000
	config.AuthenticatedReadsPerSecond = 1_000_000
	config.AuthenticatedReadBurst = 1_000_000
	config.AuthenticatedMutationsPerSecond = 1_000_000
	config.AuthenticatedMutationBurst = 1_000_000
	config.ReleasesPerUserPerHour = 1_000_000
	config.ReleaseUserBurst = 1_000_000
	config.ReleasesPerIPPerHour = 1_000_000
	config.ReleaseIPBurst = 1_000_000

	router := gin.New()
	router.Use(func(context *gin.Context) {
		switch context.GetHeader("X-Test-Auth") {
		case "malformed":
			context.Set("user", "not-a-bootstrap-user")
		case "valid":
			context.Set(
				"user",
				&bootstrap.User{
					UserID: context.GetHeader("X-Test-User"),
				},
			)
		}
		context.Next()
	})
	userpmcs.RegisterRoutes(
		userpmcs.Dependencies{DB: testDB, Config: config},
		router.Group(""),
		router.Group("/auth"),
	)
	return router
}

func performContractRequest(
	router http.Handler,
	method string,
	path string,
	userUID string,
	body []byte,
	headers map[string]string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if userUID != "" {
		request.Header.Set("X-Test-Auth", "valid")
		request.Header.Set("X-Test-User", userUID)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func requireStableAPIError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	code string,
) {
	t.Helper()
	require.Equal(t, status, response.Code, response.Body.String())
	var envelope struct {
		Status  int             `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Equal(t, status, envelope.Status)
	require.NotEmpty(t, strings.TrimSpace(envelope.Message))
	require.Equal(t, "null", string(envelope.Data))
	require.Equal(t, code, envelope.Error.Code)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return payload
}

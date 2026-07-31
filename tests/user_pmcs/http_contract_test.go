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

func TestHTTPContractSubscriptionReadsRequireInitializedAccount(t *testing.T) {
	router := newUserPmcsContractRouter(shared.DefaultConfig())
	uninitializedUID := "uninitialized-" + uuid.NewString()
	checklistID := uuid.NewString()
	revisionID := uuid.NewString()

	cases := []userPmcsRouteContract{
		{
			name:   "update discovery",
			method: http.MethodGet,
			path:   "/auth/user-pmcs/subscriptions/updates",
		},
		{
			name:   "pinned release",
			method: http.MethodGet,
			path: "/auth/user-pmcs/subscriptions/" + checklistID +
				"/installed-releases/" + revisionID,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			response := performContractRequest(
				router,
				testCase.method,
				testCase.path,
				uninitializedUID,
				nil,
				nil,
			)
			requireStableAPIError(
				t,
				response,
				http.StatusConflict,
				"account_not_initialized",
			)
		})
	}
}

func TestAuthorizationSafe404EnvelopesAreIndistinguishable(t *testing.T) {
	ctx := context.Background()
	config := shared.DefaultConfig()
	router := newUserPmcsContractRouter(config)
	fixture := newReleasedChecklistFixture(t, 1)
	_, err := fixture.repository.Release(
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
	otherUID := newUserPmcsTestUser(t)
	unknownChecklistID := uuid.New()
	unknownRevisionID := uuid.New()
	crossDraft := preparedTree(t, uuid.New()).Input
	unknownDraft := preparedTree(t, uuid.New()).Input
	crossPublication := preparePublication(t, crossDraft, 2).Input
	unknownPublication := preparePublication(t, unknownDraft, 1).Input
	stale := `"untrusted-validator"`

	type requestSpec struct {
		method  string
		path    string
		userUID string
		body    []byte
		headers map[string]string
	}
	request := func(spec requestSpec) *httptest.ResponseRecorder {
		return performContractRequest(
			router,
			spec.method,
			spec.path,
			spec.userUID,
			spec.body,
			spec.headers,
		)
	}
	ownedCases := []struct {
		name    string
		unknown requestSpec
		cross   requestSpec
	}{
		{
			"get checklist",
			requestSpec{http.MethodGet, "/auth/user-pmcs/checklists/" + unknownChecklistID.String(), otherUID, nil, nil},
			requestSpec{http.MethodGet, "/auth/user-pmcs/checklists/" + fixture.checklist.String(), otherUID, nil, nil},
		},
		{
			"delete checklist",
			requestSpec{http.MethodDelete, "/auth/user-pmcs/checklists/" + unknownChecklistID.String(), otherUID, nil, map[string]string{"If-Match": stale}},
			requestSpec{http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String(), otherUID, nil, map[string]string{"If-Match": stale}},
		},
		{
			"put draft",
			requestSpec{http.MethodPut, "/auth/user-pmcs/checklists/" + unknownChecklistID.String() + "/drafts/" + unknownDraft.ID.String(), otherUID, mustJSON(t, unknownDraft), map[string]string{"Content-Type": "application/json", "If-Match": stale}},
			requestSpec{http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/drafts/" + crossDraft.ID.String(), otherUID, mustJSON(t, crossDraft), map[string]string{"Content-Type": "application/json", "If-Match": stale}},
		},
		{
			"delete draft",
			requestSpec{http.MethodDelete, "/auth/user-pmcs/checklists/" + unknownChecklistID.String() + "/drafts/" + unknownRevisionID.String(), otherUID, nil, map[string]string{"If-Match": stale}},
			requestSpec{http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/drafts/" + fixture.revisions[0].Input.ID.String(), otherUID, nil, map[string]string{"If-Match": stale}},
		},
		{
			"publish",
			requestSpec{http.MethodPut, "/auth/user-pmcs/checklists/" + unknownChecklistID.String() + "/publications/" + unknownPublication.ID.String(), otherUID, mustJSON(t, unknownPublication), map[string]string{"Content-Type": "application/json", "If-Match": stale}},
			requestSpec{http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/publications/" + crossPublication.ID.String(), otherUID, mustJSON(t, crossPublication), map[string]string{"Content-Type": "application/json", "If-Match": stale}},
		},
		{
			"get revision",
			requestSpec{http.MethodGet, "/auth/user-pmcs/checklists/" + unknownChecklistID.String() + "/revisions/" + unknownRevisionID.String(), otherUID, nil, nil},
			requestSpec{http.MethodGet, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/revisions/" + fixture.revisions[0].Input.ID.String(), otherUID, nil, nil},
		},
		{
			"release",
			requestSpec{http.MethodPut, "/auth/user-pmcs/checklists/" + unknownChecklistID.String() + "/community-releases/" + unknownRevisionID.String(), otherUID, nil, map[string]string{"If-Match": stale}},
			requestSpec{http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/community-releases/" + fixture.revisions[0].Input.ID.String(), otherUID, nil, map[string]string{"If-Match": stale}},
		},
		{
			"retire",
			requestSpec{http.MethodDelete, "/auth/user-pmcs/checklists/" + unknownChecklistID.String() + "/community-source", otherUID, nil, map[string]string{"If-Match": stale}},
			requestSpec{http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/community-source", otherUID, nil, map[string]string{"If-Match": stale}},
		},
	}
	for _, testCase := range ownedCases {
		testCase := testCase
		t.Run("owned/"+testCase.name, func(t *testing.T) {
			requireIndistinguishableSafe404s(
				t,
				"checklist not found",
				request(testCase.unknown),
				request(testCase.cross),
			)
		})
	}

	privateChecklistID := uuid.New()
	privateDraft := preparedTree(t, uuid.New())
	privateCreated, err := newOwnedRepository().Create(
		ctx,
		fixture.ownerUID,
		privateChecklistID,
		privateDraft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	deletedChecklistID := uuid.New()
	deletedDraft := preparedTree(t, uuid.New())
	deletedCreated, err := newOwnedRepository().Create(
		ctx,
		fixture.ownerUID,
		deletedChecklistID,
		deletedDraft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	_, err = newOwnedRepository().DeleteChecklist(
		ctx,
		fixture.ownerUID,
		deletedChecklistID,
		checklistPrecondition(
			deletedChecklistID,
			deletedCreated.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	require.Positive(t, privateCreated.Aggregate.SyncVersion)

	t.Run("public current release", func(t *testing.T) {
		requireIndistinguishableSafe404s(
			t,
			"community checklist not found",
			request(requestSpec{http.MethodGet, "/user-pmcs/community/" + unknownChecklistID.String(), "", nil, nil}),
			request(requestSpec{http.MethodGet, "/user-pmcs/community/" + privateChecklistID.String(), "", nil, nil}),
			request(requestSpec{http.MethodGet, "/user-pmcs/community/" + deletedChecklistID.String(), "", nil, nil}),
		)
	})
	t.Run("install unavailable source", func(t *testing.T) {
		requireIndistinguishableSafe404s(
			t,
			"community checklist not found",
			request(requestSpec{http.MethodPut, "/auth/user-pmcs/subscriptions/" + unknownChecklistID.String(), otherUID, nil, map[string]string{"If-None-Match": "*"}}),
			request(requestSpec{http.MethodPut, "/auth/user-pmcs/subscriptions/" + privateChecklistID.String(), otherUID, nil, map[string]string{"If-None-Match": "*"}}),
			request(requestSpec{http.MethodPut, "/auth/user-pmcs/subscriptions/" + deletedChecklistID.String(), otherUID, nil, map[string]string{"If-None-Match": "*"}}),
		)
	})

	subscriptionRepository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 3),
		config,
	)
	installed, err := subscriptionRepository.Install(
		ctx,
		otherUID,
		fixture.checklist,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	_, err = subscriptionRepository.Unsubscribe(
		ctx,
		otherUID,
		fixture.checklist,
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeSubscriptionETag(
				fixture.checklist,
				installed.Subscription.SyncVersion,
			),
		},
	)
	require.NoError(t, err)
	t.Run("accept update missing subscription", func(t *testing.T) {
		requireIndistinguishableSafe404s(
			t,
			"subscription not found",
			request(requestSpec{http.MethodPut, "/auth/user-pmcs/subscriptions/" + unknownChecklistID.String() + "/installed-releases/" + unknownRevisionID.String(), otherUID, nil, map[string]string{"If-Match": stale}}),
			request(requestSpec{http.MethodPut, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String() + "/installed-releases/" + fixture.revisions[0].Input.ID.String(), otherUID, nil, map[string]string{"If-Match": stale}}),
		)
	})
	t.Run("pinned read missing subscription", func(t *testing.T) {
		requireIndistinguishableSafe404s(
			t,
			"installed checklist release not found",
			request(requestSpec{http.MethodGet, "/auth/user-pmcs/subscriptions/" + unknownChecklistID.String() + "/installed-releases/" + unknownRevisionID.String(), otherUID, nil, nil}),
			request(requestSpec{http.MethodGet, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String() + "/installed-releases/" + fixture.revisions[0].Input.ID.String(), otherUID, nil, nil}),
		)
	})
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
			name:       "create rejects empty body",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String(),
			headers:    map[string]string{"Content-Type": "application/json", "If-None-Match": "*"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "put draft rejects absent content type",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String(),
			body:       mustJSON(t, draft),
			headers:    map[string]string{"If-Match": `"current"`},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "put draft rejects empty body",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String(),
			headers:    map[string]string{"Content-Type": "application/json", "If-Match": `"current"`},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "put draft rejects malformed body",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String(),
			body:       []byte(`{"id":`),
			headers:    map[string]string{"Content-Type": "application/json", "If-Match": `"current"`},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "publish rejects absent content type",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/publications/" + revisionID.String(),
			body:       mustJSON(t, publication),
			headers:    map[string]string{"If-Match": `"current"`},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "publish rejects empty body",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/publications/" + revisionID.String(),
			headers:    map[string]string{"Content-Type": "application/json", "If-Match": `"current"`},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "publish rejects malformed body",
			method:     http.MethodPut,
			path:       "/auth/user-pmcs/checklists/" + checklistID.String() + "/publications/" + revisionID.String(),
			body:       []byte(`{"id":`),
			headers:    map[string]string{"Content-Type": "application/json", "If-Match": `"current"`},
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
	for _, mutation := range []struct {
		name    string
		path    string
		headers map[string]string
	}{
		{
			"create",
			"/auth/user-pmcs/checklists/" + uuid.NewString(),
			map[string]string{"Content-Type": "application/json", "If-None-Match": "*"},
		},
		{
			"put draft",
			"/auth/user-pmcs/checklists/" + checklistID.String() + "/drafts/" + revisionID.String(),
			map[string]string{"Content-Type": "application/json", "If-Match": `"current"`},
		},
		{
			"publish",
			"/auth/user-pmcs/checklists/" + checklistID.String() + "/publications/" + revisionID.String(),
			map[string]string{"Content-Type": "application/json", "If-Match": `"current"`},
		},
	} {
		t.Run(mutation.name+" rejects oversized body", func(t *testing.T) {
			oversized := performContractRequest(
				smallRouter,
				http.MethodPut,
				mutation.path,
				userUID,
				bytes.Repeat([]byte{'x'}, 65),
				mutation.headers,
			)
			requireStableAPIError(
				t,
				oversized,
				http.StatusRequestEntityTooLarge,
				"content_too_large",
			)
		})
	}
}

func TestHTTPContractConditionalHeaderMatrix(t *testing.T) {
	ctx := context.Background()
	config := shared.DefaultConfig()
	router := newUserPmcsContractRouter(config)
	fixture := newReleasedChecklistFixture(t, 3)
	firstRelease, err := fixture.repository.Release(
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
	subscriberUID := newUserPmcsTestUser(t)
	subscriptionRepository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 3),
		config,
	)
	installed, err := subscriptionRepository.Install(
		ctx,
		subscriberUID,
		fixture.checklist,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	secondRelease, err := fixture.repository.Release(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(
			fixture.checklist,
			firstRelease.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	currentDraft := preparedTree(t, uuid.New())
	_, err = newOwnedRepository().PutDraft(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		currentDraft,
		checklistPrecondition(
			fixture.checklist,
			secondRelease.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	stale := `"stale"`
	malformedPutDraft := preparedTree(t, uuid.New()).Input
	malformedPublicationDraft := preparedTree(t, uuid.New())
	malformedPublication := preparePublication(
		t,
		malformedPublicationDraft.Input,
		1,
	).Input

	malformedCases := []struct {
		name    string
		method  string
		path    string
		userUID string
		body    []byte
		headers map[string]string
	}{
		{"create", http.MethodPut, "/auth/user-pmcs/checklists/" + uuid.NewString(), fixture.ownerUID, mustJSON(t, preparedTree(t, uuid.New()).Input), map[string]string{"Content-Type": "application/json", "If-None-Match": stale}},
		{"put draft", http.MethodPut, "/auth/user-pmcs/checklists/" + uuid.NewString() + "/drafts/" + malformedPutDraft.ID.String(), fixture.ownerUID, mustJSON(t, malformedPutDraft), map[string]string{"Content-Type": "application/json", "If-Match": "not-an-etag"}},
		{"delete draft", http.MethodDelete, "/auth/user-pmcs/checklists/" + uuid.NewString() + "/drafts/" + uuid.NewString(), fixture.ownerUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"delete checklist", http.MethodDelete, "/auth/user-pmcs/checklists/" + uuid.NewString(), fixture.ownerUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"publish", http.MethodPut, "/auth/user-pmcs/checklists/" + uuid.NewString() + "/publications/" + malformedPublication.ID.String(), fixture.ownerUID, mustJSON(t, malformedPublication), map[string]string{"Content-Type": "application/json", "If-Match": "not-an-etag"}},
		{"release", http.MethodPut, "/auth/user-pmcs/checklists/" + uuid.NewString() + "/community-releases/" + uuid.NewString(), fixture.ownerUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"retire", http.MethodDelete, "/auth/user-pmcs/checklists/" + uuid.NewString() + "/community-source", fixture.ownerUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"install create header", http.MethodPut, "/auth/user-pmcs/subscriptions/" + uuid.NewString(), subscriberUID, nil, map[string]string{"If-None-Match": stale}},
		{"install existing header", http.MethodPut, "/auth/user-pmcs/subscriptions/" + uuid.NewString(), subscriberUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"install both headers", http.MethodPut, "/auth/user-pmcs/subscriptions/" + uuid.NewString(), subscriberUID, nil, map[string]string{"If-None-Match": "*", "If-Match": stale}},
		{"unsubscribe", http.MethodDelete, "/auth/user-pmcs/subscriptions/" + uuid.NewString(), subscriberUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"accept update", http.MethodPut, "/auth/user-pmcs/subscriptions/" + uuid.NewString() + "/installed-releases/" + uuid.NewString(), subscriberUID, nil, map[string]string{"If-Match": "not-an-etag"}},
		{"get checklist", http.MethodGet, "/auth/user-pmcs/checklists/" + fixture.checklist.String(), fixture.ownerUID, nil, map[string]string{"If-None-Match": "not-an-etag"}},
		{"get revision", http.MethodGet, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/revisions/" + fixture.revisions[0].Input.ID.String(), fixture.ownerUID, nil, map[string]string{"If-None-Match": "not-an-etag"}},
		{"public release", http.MethodGet, "/user-pmcs/community/" + fixture.checklist.String(), "", nil, map[string]string{"If-None-Match": "not-an-etag"}},
		{"installed release", http.MethodGet, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String() + "/installed-releases/" + fixture.revisions[0].Input.ID.String(), subscriberUID, nil, map[string]string{"If-None-Match": "not-an-etag"}},
	}
	for _, testCase := range malformedCases {
		testCase := testCase
		t.Run("malformed/"+testCase.name, func(t *testing.T) {
			response := performContractRequest(
				router,
				testCase.method,
				testCase.path,
				testCase.userUID,
				testCase.body,
				testCase.headers,
			)
			requireStableAPIError(
				t,
				response,
				http.StatusBadRequest,
				"invalid_precondition",
			)
		})
	}

	privateChecklistID := uuid.New()
	privateDraft := preparedTree(t, uuid.New())
	privateCreated, err := newOwnedRepository().Create(
		ctx,
		fixture.ownerUID,
		privateChecklistID,
		privateDraft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	require.Positive(t, privateCreated.Aggregate.SyncVersion)
	stalePutDraft := preparedTree(t, uuid.New()).Input

	staleCases := []struct {
		name    string
		method  string
		path    string
		userUID string
		body    []byte
		headers map[string]string
	}{
		{"create collision", http.MethodPut, "/auth/user-pmcs/checklists/" + privateChecklistID.String(), fixture.ownerUID, mustJSON(t, preparedTree(t, uuid.New()).Input), map[string]string{"Content-Type": "application/json", "If-None-Match": "*"}},
		{"put draft", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/drafts/" + stalePutDraft.ID.String(), fixture.ownerUID, mustJSON(t, stalePutDraft), map[string]string{"Content-Type": "application/json", "If-Match": stale}},
		{"delete draft", http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/drafts/" + currentDraft.Input.ID.String(), fixture.ownerUID, nil, map[string]string{"If-Match": stale}},
		{"delete checklist", http.MethodDelete, "/auth/user-pmcs/checklists/" + privateChecklistID.String(), fixture.ownerUID, nil, map[string]string{"If-Match": stale}},
		{"publish", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/publications/" + currentDraft.Input.ID.String(), fixture.ownerUID, mustJSON(t, preparePublication(t, currentDraft.Input, 3).Input), map[string]string{"Content-Type": "application/json", "If-Match": stale}},
		{"release", http.MethodPut, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/community-releases/" + fixture.revisions[2].Input.ID.String(), fixture.ownerUID, nil, map[string]string{"If-Match": stale}},
		{"retire", http.MethodDelete, "/auth/user-pmcs/checklists/" + fixture.checklist.String() + "/community-source", fixture.ownerUID, nil, map[string]string{"If-Match": stale}},
		{"install", http.MethodPut, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String(), subscriberUID, nil, map[string]string{"If-Match": stale}},
		{"unsubscribe", http.MethodDelete, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String(), subscriberUID, nil, map[string]string{"If-Match": stale}},
		{"accept update", http.MethodPut, "/auth/user-pmcs/subscriptions/" + fixture.checklist.String() + "/installed-releases/" + fixture.revisions[1].Input.ID.String(), subscriberUID, nil, map[string]string{"If-Match": stale}},
	}
	for _, testCase := range staleCases {
		testCase := testCase
		t.Run("stale/"+testCase.name, func(t *testing.T) {
			response := performContractRequest(
				router,
				testCase.method,
				testCase.path,
				testCase.userUID,
				testCase.body,
				testCase.headers,
			)
			requireStableAPIError(
				t,
				response,
				http.StatusPreconditionFailed,
				"stale_precondition",
			)
		})
	}

	require.Equal(
		t,
		shared.MakeSubscriptionETag(
			fixture.checklist,
			installed.Subscription.SyncVersion,
		),
		shared.MakeSubscriptionETag(fixture.checklist, 1),
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

	tombstoneETag := shared.MakeChecklistETag(
		checklistID,
		deleted.Aggregate.SyncVersion,
	)
	repeatedDelete := performContractRequest(
		router,
		http.MethodDelete,
		"/auth/user-pmcs/checklists/"+checklistID.String(),
		ownerUID,
		nil,
		map[string]string{"If-Match": tombstoneETag},
	)
	require.Equal(t, http.StatusOK, repeatedDelete.Code)
	require.Equal(t, tombstoneETag, repeatedDelete.Header().Get("ETag"))
	require.NotContains(t, repeatedDelete.Body.String(), `"sections"`)
	require.NotContains(t, repeatedDelete.Body.String(), draft.Input.Name)

	rejectedDraft := preparedTree(t, uuid.New()).Input
	rejectedPublication := preparePublication(t, rejectedDraft, 1).Input
	for _, mutation := range []struct {
		name    string
		method  string
		path    string
		body    []byte
		headers map[string]string
	}{
		{
			"create",
			http.MethodPut,
			"/auth/user-pmcs/checklists/" + checklistID.String(),
			mustJSON(t, rejectedDraft),
			map[string]string{
				"Content-Type":  "application/json",
				"If-None-Match": "*",
			},
		},
		{
			"put draft",
			http.MethodPut,
			"/auth/user-pmcs/checklists/" + checklistID.String() +
				"/drafts/" + rejectedDraft.ID.String(),
			mustJSON(t, rejectedDraft),
			map[string]string{
				"Content-Type": "application/json",
				"If-Match":     tombstoneETag,
			},
		},
		{
			"delete draft",
			http.MethodDelete,
			"/auth/user-pmcs/checklists/" + checklistID.String() +
				"/drafts/" + draft.Input.ID.String(),
			nil,
			map[string]string{"If-Match": tombstoneETag},
		},
		{
			"publish",
			http.MethodPut,
			"/auth/user-pmcs/checklists/" + checklistID.String() +
				"/publications/" + rejectedPublication.ID.String(),
			mustJSON(t, rejectedPublication),
			map[string]string{
				"Content-Type": "application/json",
				"If-Match":     tombstoneETag,
			},
		},
		{
			"release",
			http.MethodPut,
			"/auth/user-pmcs/checklists/" + checklistID.String() +
				"/community-releases/" + draft.Input.ID.String(),
			nil,
			map[string]string{"If-Match": tombstoneETag},
		},
		{
			"retire",
			http.MethodDelete,
			"/auth/user-pmcs/checklists/" + checklistID.String() +
				"/community-source",
			nil,
			map[string]string{"If-Match": tombstoneETag},
		},
	} {
		t.Run("owner tombstone rejects "+mutation.name, func(t *testing.T) {
			response := performContractRequest(
				router,
				mutation.method,
				mutation.path,
				ownerUID,
				mutation.body,
				mutation.headers,
			)
			requireStableAPIError(
				t,
				response,
				http.StatusPreconditionFailed,
				"stale_precondition",
			)
			require.NotContains(t, response.Body.String(), draft.Input.Name)
		})
	}
	historical := performContractRequest(
		router,
		http.MethodGet,
		"/auth/user-pmcs/checklists/"+checklistID.String()+
			"/revisions/"+draft.Input.ID.String(),
		ownerUID,
		nil,
		nil,
	)
	requireStableAPIError(
		t,
		historical,
		http.StatusNotFound,
		"resource_not_found",
	)
	install := performContractRequest(
		router,
		http.MethodPut,
		"/auth/user-pmcs/subscriptions/"+checklistID.String(),
		otherUID,
		nil,
		map[string]string{"If-None-Match": "*"},
	)
	requireStableAPIError(
		t,
		install,
		http.StatusNotFound,
		"resource_not_found",
	)

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

func TestAuthorizationSubscriptionTombstoneRouteMatrix(t *testing.T) {
	ctx := context.Background()
	config := shared.DefaultConfig()
	fixture := newReleasedChecklistFixture(t, 2)
	firstRelease, err := fixture.repository.Release(
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
	subscriberUID := newUserPmcsTestUser(t)
	repository := subscriptions.NewRepository(
		persistence.NewStore(testDB, 3),
		config,
	)
	installed, err := repository.Install(
		ctx,
		subscriberUID,
		fixture.checklist,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	_, err = fixture.repository.Release(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(
			fixture.checklist,
			firstRelease.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	tombstone, err := repository.Unsubscribe(
		ctx,
		subscriberUID,
		fixture.checklist,
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeSubscriptionETag(
				fixture.checklist,
				installed.Subscription.SyncVersion,
			),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, tombstone.Subscription.DeletedAt)
	require.Nil(t, tombstone.Subscription.InstalledRevisionID)
	tombstoneETag := shared.MakeSubscriptionETag(
		fixture.checklist,
		tombstone.Subscription.SyncVersion,
	)
	router := newUserPmcsContractRouter(config)

	updates := performContractRequest(
		router,
		http.MethodGet,
		"/auth/user-pmcs/subscriptions/updates",
		subscriberUID,
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, updates.Code)
	require.NotContains(t, updates.Body.String(), fixture.checklist.String())

	accept := performContractRequest(
		router,
		http.MethodPut,
		"/auth/user-pmcs/subscriptions/"+fixture.checklist.String()+
			"/installed-releases/"+fixture.revisions[1].Input.ID.String(),
		subscriberUID,
		nil,
		map[string]string{"If-Match": tombstoneETag},
	)
	requireStableAPIError(
		t,
		accept,
		http.StatusNotFound,
		"resource_not_found",
	)
	pinned := performContractRequest(
		router,
		http.MethodGet,
		"/auth/user-pmcs/subscriptions/"+fixture.checklist.String()+
			"/installed-releases/"+fixture.revisions[0].Input.ID.String(),
		subscriberUID,
		nil,
		nil,
	)
	requireStableAPIError(
		t,
		pinned,
		http.StatusNotFound,
		"resource_not_found",
	)

	repeatedUnsubscribe := performContractRequest(
		router,
		http.MethodDelete,
		"/auth/user-pmcs/subscriptions/"+fixture.checklist.String(),
		subscriberUID,
		nil,
		map[string]string{"If-Match": tombstoneETag},
	)
	require.Equal(t, http.StatusOK, repeatedUnsubscribe.Code)
	require.Equal(
		t,
		tombstoneETag,
		repeatedUnsubscribe.Header().Get("ETag"),
	)
	require.NotContains(
		t,
		repeatedUnsubscribe.Body.String(),
		`"installed_revision_id":"`,
	)

	createStyleReinstall := performContractRequest(
		router,
		http.MethodPut,
		"/auth/user-pmcs/subscriptions/"+fixture.checklist.String(),
		subscriberUID,
		nil,
		map[string]string{"If-None-Match": "*"},
	)
	requireStableAPIError(
		t,
		createStyleReinstall,
		http.StatusPreconditionFailed,
		"stale_precondition",
	)
	resubscribe := performContractRequest(
		router,
		http.MethodPut,
		"/auth/user-pmcs/subscriptions/"+fixture.checklist.String(),
		subscriberUID,
		nil,
		map[string]string{"If-Match": tombstoneETag},
	)
	require.Equal(t, http.StatusOK, resubscribe.Code)
	require.Contains(
		t,
		resubscribe.Body.String(),
		fixture.revisions[1].Input.ID.String(),
	)
	require.NotEqual(t, tombstoneETag, resubscribe.Header().Get("ETag"))
}

func TestAuthorizationPublicVisibilityNoticeInvariantAndCurrentCreatorName(
	t *testing.T,
) {
	ctx := context.Background()
	fixture := newReleasedChecklistFixture(t, 3)
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
	require.NotEqual(t, fixture.revisions[2].Input.ID, current.Revision.ID)
	require.Equal(
		t,
		"superseded",
		revisionState(t, fixture.revisions[1].Input.ID),
	)
	require.Equal(
		t,
		"published",
		revisionState(t, fixture.revisions[2].Input.ID),
	)
	privateDraft := preparedTree(t, uuid.New())
	withPrivateDraft, err := newOwnedRepository().PutDraft(
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

	router := newUserPmcsContractRouter(shared.DefaultConfig())
	publicResponse := performContractRequest(
		router,
		http.MethodGet,
		"/user-pmcs/community/"+fixture.checklist.String(),
		"",
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, publicResponse.Code)
	require.NotContains(
		t,
		publicResponse.Body.String(),
		fixture.revisions[1].Input.ID.String(),
	)
	require.NotContains(
		t,
		publicResponse.Body.String(),
		fixture.revisions[2].Input.ID.String(),
	)
	require.NotContains(
		t,
		publicResponse.Body.String(),
		privateDraft.Input.ID.String(),
	)

	beforeRejectedPublish, err := newOwnedRepository().Get(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
	)
	require.NoError(t, err)
	var publicationCountBefore int
	require.NoError(t, testDB.QueryRowContext(
		ctx,
		`SELECT count(*)
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1
		   AND state IN ('published', 'superseded')`,
		fixture.checklist,
	).Scan(&publicationCountBefore))
	incompleteCases := []struct {
		name          string
		mutate        func(*shared.RevisionInput)
		expectedField string
	}{
		{
			"missing notice type",
			func(input *shared.RevisionInput) {
				input.Sections[0].Items[0].Notices[0].Type = nil
			},
			"revision.sections[0].items[0].notices[0].type",
		},
		{
			"blank notice text",
			func(input *shared.RevisionInput) {
				input.Sections[0].Items[0].Notices[0].NoticeText = " "
			},
			"revision.sections[0].items[0].notices[0].notice_text",
		},
	}
	for _, testCase := range incompleteCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			var publication shared.RevisionInput
			require.NoError(
				t,
				json.Unmarshal(mustJSON(t, privateDraft.Input), &publication),
			)
			number := int32(4)
			publication.RevisionNumber = &number
			testCase.mutate(&publication)
			response := performContractRequest(
				router,
				http.MethodPut,
				"/auth/user-pmcs/checklists/"+fixture.checklist.String()+
					"/publications/"+publication.ID.String(),
				fixture.ownerUID,
				mustJSON(t, publication),
				map[string]string{
					"Content-Type": "application/json",
					"If-Match": shared.MakeChecklistETag(
						fixture.checklist,
						withPrivateDraft.Aggregate.SyncVersion,
					),
				},
			)
			requireValidationAPIError(
				t,
				response,
				testCase.expectedField,
			)
		})
	}
	afterRejectedPublish, err := newOwnedRepository().Get(
		ctx,
		fixture.ownerUID,
		fixture.checklist,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		beforeRejectedPublish.SyncVersion,
		afterRejectedPublish.SyncVersion,
	)
	require.NotNil(t, afterRejectedPublish.Draft)
	require.Equal(t, privateDraft.Input.ID, afterRejectedPublish.Draft.ID)
	require.NotNil(t, afterRejectedPublish.Publication)
	require.Equal(
		t,
		fixture.revisions[2].Input.ID,
		afterRejectedPublish.Publication.ID,
	)
	var publicationCountAfter int
	require.NoError(t, testDB.QueryRowContext(
		ctx,
		`SELECT count(*)
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1
		   AND state IN ('published', 'superseded')`,
		fixture.checklist,
	).Scan(&publicationCountAfter))
	require.Equal(t, publicationCountBefore, publicationCountAfter)

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
	_ = decodeStableAPIErrorEnvelope(t, response, status, code)
}

func decodeStableAPIErrorEnvelope(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	code string,
) map[string]any {
	t.Helper()
	require.Equal(t, status, response.Code, response.Body.String())
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Equal(t, 4, len(envelope), "complete error envelope changed")
	require.Equal(t, float64(status), envelope["status"])
	message, ok := envelope["message"].(string)
	require.True(t, ok)
	require.NotEmpty(t, strings.TrimSpace(message))
	require.Nil(t, envelope["data"])
	errorEnvelope, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{"code": code}, errorEnvelope)
	return envelope
}

func requireIndistinguishableSafe404s(
	t *testing.T,
	safeMessage string,
	responses ...*httptest.ResponseRecorder,
) {
	t.Helper()
	require.NotEmpty(t, responses)
	var expected map[string]any
	for index, response := range responses {
		envelope := decodeStableAPIErrorEnvelope(
			t,
			response,
			http.StatusNotFound,
			"resource_not_found",
		)
		require.Equal(t, safeMessage, envelope["message"])
		if index == 0 {
			expected = envelope
			continue
		}
		require.Equal(t, expected, envelope)
	}
}

func requireValidationAPIError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	expectedField string,
) {
	t.Helper()
	require.Equal(
		t,
		http.StatusUnprocessableEntity,
		response.Code,
		response.Body.String(),
	)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Equal(t, float64(http.StatusUnprocessableEntity), envelope["status"])
	require.Nil(t, envelope["data"])
	errorEnvelope, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "validation_failed", errorEnvelope["code"])
	details, ok := errorEnvelope["details"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, expectedField, details["field"])
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return payload
}

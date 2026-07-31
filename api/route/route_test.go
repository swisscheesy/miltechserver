package route

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	userpmcsshared "miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

func TestSetupRegistersPmcsSbsFaultRoutesUnderAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Setup(nil, router, nil, nil, nil)

	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults")
	requireRouteRegistered(t, router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults")
	requireRouteRegistered(t, router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults")
	requireRouteRegistered(t, router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults/bulk")
}

func TestSetupRegistersPmcsSbsImageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Setup(nil, router, nil, nil, nil)

	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/library/pmcs-sbs/image")
}

func TestSetupRegistersUserPmcsRoutesOnApprovedGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Setup(nil, router, nil, nil, nil)

	routes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/auth/user-pmcs/sync"},
		{method: http.MethodGet, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id"},
		{method: http.MethodPut, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id"},
		{method: http.MethodPut, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id/drafts/:revision_id"},
		{method: http.MethodDelete, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id/drafts/:revision_id"},
		{method: http.MethodPut, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id/publications/:revision_id"},
		{method: http.MethodGet, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id/revisions/:revision_id"},
		{method: http.MethodDelete, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id"},
		{method: http.MethodPut, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id/community-releases/:revision_id"},
		{method: http.MethodDelete, path: "/api/v1/auth/user-pmcs/checklists/:checklist_id/community-source"},
		{method: http.MethodGet, path: "/api/v1/user-pmcs/community"},
		{method: http.MethodGet, path: "/api/v1/user-pmcs/community/:checklist_id"},
		{method: http.MethodPut, path: "/api/v1/auth/user-pmcs/subscriptions/:checklist_id"},
		{method: http.MethodDelete, path: "/api/v1/auth/user-pmcs/subscriptions/:checklist_id"},
		{method: http.MethodGet, path: "/api/v1/auth/user-pmcs/subscriptions/updates"},
		{method: http.MethodPut, path: "/api/v1/auth/user-pmcs/subscriptions/:checklist_id/installed-releases/:revision_id"},
		{method: http.MethodGet, path: "/api/v1/auth/user-pmcs/subscriptions/:checklist_id/installed-releases/:revision_id"},
	}

	for _, route := range routes {
		requireRouteRegistered(t, router, route.method, route.path)
	}
}

func TestSetupUserPmcsPublicObservabilityDoesNotLogAuthoredQueryText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var accessOutput bytes.Buffer
	originalDefaultWriter := gin.DefaultWriter
	gin.DefaultWriter = &accessOutput
	t.Cleanup(func() {
		gin.DefaultWriter = originalDefaultWriter
	})
	originalErrorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = io.Discard
	t.Cleanup(func() {
		gin.DefaultErrorWriter = originalErrorWriter
	})

	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	config := userpmcsshared.DefaultConfig()
	config.PublicRequestsPerSecond = 1
	config.PublicRequestBurst = 1
	env := &bootstrap.Env{
		UserPmcs: bootstrap.UserPmcsConfig(config),
	}
	router := NewEngine()
	router.GET("/unrelated-access-log-test", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})
	Setup(nil, router, nil, env, nil)

	const authoredQuery = "authored-community-model-secret"
	firstRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community?model="+authoredQuery,
		nil,
	)
	firstRequest.RemoteAddr = "192.0.2.50:1000"
	router.ServeHTTP(httptest.NewRecorder(), firstRequest)

	secondRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community?model="+authoredQuery,
		nil,
	)
	secondRequest.RemoteAddr = "192.0.2.50:2000"
	secondRecorder := httptest.NewRecorder()
	router.ServeHTTP(secondRecorder, secondRequest)

	require.Equal(t, http.StatusTooManyRequests, secondRecorder.Code)
	require.NotContains(t, output.String(), authoredQuery)
	require.NotContains(t, accessOutput.String(), authoredQuery)
	require.Contains(
		t,
		accessOutput.String(),
		"GET",
	)
	require.Contains(
		t,
		accessOutput.String(),
		"/api/v1/user-pmcs/community",
	)

	const authoredPathSegment = "authored-dynamic-path-secret"
	invalidPathRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/user-pmcs/community/"+authoredPathSegment,
		nil,
	)
	invalidPathRequest.RemoteAddr = "192.0.2.51:1000"
	invalidPathRecorder := httptest.NewRecorder()
	router.ServeHTTP(invalidPathRecorder, invalidPathRequest)
	require.Equal(t, http.StatusBadRequest, invalidPathRecorder.Code)
	require.NotContains(t, output.String(), authoredPathSegment)
	require.NotContains(t, accessOutput.String(), authoredPathSegment)
	require.Contains(
		t,
		accessOutput.String(),
		"/api/v1/user-pmcs/community/:checklist_id",
	)

	unrelatedRequest := httptest.NewRequest(
		http.MethodGet,
		"/unrelated-access-log-test?filter=useful-structural-query",
		nil,
	)
	unrelatedRecorder := httptest.NewRecorder()
	router.ServeHTTP(unrelatedRecorder, unrelatedRequest)
	require.Equal(t, http.StatusNoContent, unrelatedRecorder.Code)
	require.Contains(t, accessOutput.String(), "useful-structural-query")
}

func requireRouteRegistered(t *testing.T, router *gin.Engine, method string, path string) {
	t.Helper()

	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			return
		}
	}
	require.Failf(t, "route not registered", "%s %s", method, path)
}

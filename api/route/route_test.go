package route

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetupRegistersPmcsSbsProgressRoutesUnderAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Setup(nil, router, nil, nil, nil)

	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment")
	requireRouteRegistered(t, router, http.MethodPatch, "/api/v1/auth/pmcs-sbs/equipment/:equipment_id/completions/batch")
	requireRouteRegistered(t, router, http.MethodPost, "/api/v1/auth/pmcs-sbs/sync")
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

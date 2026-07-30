package subscriptions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type subscriptionServiceStub struct {
	installResult *MutationResult
	installETag   string
	installed     *shared.InstalledChecklistRelease
	installedETag string
}

func (stub *subscriptionServiceStub) Install(context.Context, *bootstrap.User, string, string, string) (*MutationResult, string, error) {
	return stub.installResult, stub.installETag, nil
}
func (stub *subscriptionServiceStub) Unsubscribe(context.Context, *bootstrap.User, string, string) (*MutationResult, string, error) {
	return nil, "", nil
}
func (stub *subscriptionServiceStub) GetInstalledRelease(context.Context, *bootstrap.User, string, string) (*shared.InstalledChecklistRelease, string, error) {
	return stub.installed, stub.installedETag, nil
}

func TestSubscriptionPinnedHandlerUsesPrivateImmutableCacheAndConditionalGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	checklistID, revisionID := uuid.New(), uuid.New()
	stub := &subscriptionServiceStub{installed: &shared.InstalledChecklistRelease{ChecklistID: checklistID, Revision: shared.Revision{ID: revisionID, Models: []shared.ModelValue{}, Sections: []shared.Section{}}}, installedETag: `"pinned"`}
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "subscriber-1"}); c.Next() })
	RegisterRoutes(group, stub)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/user-pmcs/subscriptions/"+checklistID.String()+"/installed-releases/"+revisionID.String(), nil)
	request.Header.Set("If-None-Match", `"pinned"`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNotModified, response.Code)
	require.Equal(t, "private, max-age=31536000, immutable", response.Header().Get("Cache-Control"))
	require.Equal(t, `"pinned"`, response.Header().Get("ETag"))
}

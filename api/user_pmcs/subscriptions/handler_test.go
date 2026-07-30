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
	updates       *shared.SubscriptionUpdatePage
	acceptResult  *MutationResult
	acceptETag    string
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
func (stub *subscriptionServiceStub) ListUpdates(context.Context, *bootstrap.User, string, string) (*shared.SubscriptionUpdatePage, error) {
	return stub.updates, nil
}
func (stub *subscriptionServiceStub) AcceptUpdate(context.Context, *bootstrap.User, string, string, string) (*MutationResult, string, error) {
	return stub.acceptResult, stub.acceptETag, nil
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
	require.Equal(t, "private, no-cache", response.Header().Get("Cache-Control"))
	require.NotContains(t, response.Header().Get("Cache-Control"), "immutable")
	require.Equal(t, `"pinned"`, response.Header().Get("ETag"))
}

func TestSubscriptionPinnedHandlerUsesWeakIfNoneMatchSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	checklistID, revisionID := uuid.New(), uuid.New()
	etag := `"pinned"`
	stub := &subscriptionServiceStub{installed: &shared.InstalledChecklistRelease{ChecklistID: checklistID, Revision: shared.Revision{ID: revisionID, Models: []shared.ModelValue{}, Sections: []shared.Section{}}}, installedETag: etag}
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "subscriber-1"}); c.Next() })
	RegisterRoutes(group, stub)
	path := "/api/v1/auth/user-pmcs/subscriptions/" + checklistID.String() + "/installed-releases/" + revisionID.String()

	for _, test := range []struct {
		name       string
		headers    []string
		wantStatus int
	}{
		{name: "wildcard", headers: []string{"*"}, wantStatus: http.StatusNotModified},
		{name: "weak", headers: []string{`W/"pinned"`}, wantStatus: http.StatusNotModified},
		{name: "list", headers: []string{`"other", W/"pinned", "third"`}, wantStatus: http.StatusNotModified},
		{name: "repeated", headers: []string{`"other"`, `W/"pinned"`}, wantStatus: http.StatusNotModified},
		{name: "non-match", headers: []string{`W/"other"`}, wantStatus: http.StatusOK},
		{name: "mixed wildcard", headers: []string{"*", `"other"`}, wantStatus: http.StatusBadRequest},
		{name: "malformed", headers: []string{"not-an-entity-tag"}, wantStatus: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			for _, header := range test.headers {
				request.Header.Add("If-None-Match", header)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, test.wantStatus, response.Code)
			if test.wantStatus == http.StatusNotModified {
				require.Empty(t, response.Body.Bytes())
			}
			if test.wantStatus == http.StatusBadRequest {
				require.Empty(t, response.Header().Get("ETag"))
				require.Empty(t, response.Header().Get("Cache-Control"))
			}
			if test.wantStatus != http.StatusBadRequest {
				require.Equal(t, "private, no-cache", response.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestSubscriptionUpdateAndAcceptanceRoutesReturnCanonicalSubscriptionData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	checklistID, installedID, currentID := uuid.New(), uuid.New(), uuid.New()
	stub := &subscriptionServiceStub{
		updates: &shared.SubscriptionUpdatePage{Items: []shared.SubscriptionUpdate{{
			ChecklistID:              checklistID,
			SourceStatus:             "active",
			InstalledRevisionID:      installedID,
			InstalledRevisionNumber:  1,
			CurrentReleaseRevisionID: &currentID,
			CurrentReleaseNumber:     int32Pointer(2),
			UpdateAvailable:          true,
		}}},
		acceptResult: &MutationResult{Subscription: shared.Subscription{
			ChecklistID: checklistID, InstalledRevisionID: &currentID, SyncVersion: 2,
		}},
		acceptETag: `"accepted"`,
	}
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "subscriber-1"}); c.Next() })
	RegisterRoutes(group, stub)

	updatesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/user-pmcs/subscriptions/updates", nil)
	updatesResponse := httptest.NewRecorder()
	router.ServeHTTP(updatesResponse, updatesRequest)
	require.Equal(t, http.StatusOK, updatesResponse.Code)
	require.Contains(t, updatesResponse.Body.String(), `"update_available":true`)
	require.NotContains(t, updatesResponse.Body.String(), `"sections"`)

	acceptRequest := httptest.NewRequest(http.MethodPut, "/api/v1/auth/user-pmcs/subscriptions/"+checklistID.String()+"/installed-releases/"+currentID.String(), nil)
	acceptRequest.Header.Set("If-Match", `"current"`)
	acceptResponse := httptest.NewRecorder()
	router.ServeHTTP(acceptResponse, acceptRequest)
	require.Equal(t, http.StatusOK, acceptResponse.Code)
	require.Equal(t, `"accepted"`, acceptResponse.Header().Get("ETag"))
	require.Contains(t, acceptResponse.Body.String(), currentID.String())
}

func int32Pointer(value int32) *int32 { return &value }

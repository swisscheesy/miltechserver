package aggregates

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	listsResp *response.ShopListsWithItemsResponse
	err       error
}

func (s serviceStub) GetListsWithItems(context.Context, *bootstrap.User, string) (*response.ShopListsWithItemsResponse, error) {
	return s.listsResp, s.err
}
func (s serviceStub) GetVehicleMaintenanceSnapshot(context.Context, *bootstrap.User, string, SnapshotLimits) (*response.VehicleMaintenanceSnapshotResponse, error) {
	return nil, errors.New("unexpected vehicle snapshot call")
}
func (s serviceStub) GetShopSnapshot(context.Context, *bootstrap.User, string, ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	return nil, errors.New("unexpected shop snapshot call")
}
func (s serviceStub) GetBootstrap(context.Context, *bootstrap.User, BootstrapOptions) (*response.ShopsBootstrapResponse, error) {
	return nil, errors.New("unexpected bootstrap call")
}

func TestGetListsWithItemsRequiresUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/shop-1/lists-with-items", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

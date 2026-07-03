package aggregates

import (
	"context"
	"encoding/json"
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

func TestListsWithItemsRequiresUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/shop-1/lists-with-items", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestListsWithItemsReturnsStandardResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "user-1"})
		c.Next()
	})
	RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{
		listsResp: &response.ShopListsWithItemsResponse{
			Lists: []response.ShopListWithItems{},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/shop-1/lists-with-items", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var payload struct {
		Status int `json:"status"`
		Data   struct {
			Lists []response.ShopListWithItems `json:"lists"`
		} `json:"data"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, payload.Status)
	require.Equal(t, "Shop lists with items retrieved successfully", payload.Message)
	require.Empty(t, payload.Data.Lists)
}

func TestVehicleMaintenanceSnapshotRejectsInvalidSuppliedLimitValues(t *testing.T) {
	for _, path := range []string{
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot?services_limit=",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot?changes_limit=",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot?services_limit=0",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot?changes_limit=0",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot?services_limit=-1",
		"/api/v1/auth/shops/vehicles/vehicle-1/maintenance-snapshot?changes_limit=-1",
	} {
		t.Run(path, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("user", &bootstrap.User{UserID: "user-1"})
				c.Next()
			})
			RegisterRoutes(router.Group("/api/v1/auth"), serviceStub{})

			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			require.Equal(t, http.StatusBadRequest, resp.Code)
		})
	}
}

package core

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	ginGzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type overviewServiceStub struct {
	ShopService
	result *response.ShopEquipmentOverviewResponse
	err    error
}

func (stub overviewServiceStub) GetShopEquipmentOverview(context.Context, *bootstrap.User) (*response.ShopEquipmentOverviewResponse, error) {
	return stub.result, stub.err
}

func TestGetShopEquipmentOverviewReturnsGenericServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	router.Use(func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "user-1"}); c.Next() })
	handler := Handler{service: overviewServiceStub{err: errors.New("database connection failed")}}
	router.GET("/shops/equipment/overview", handler.GetShopEquipmentOverview)
	req := httptest.NewRequest(http.MethodGet, "/shops/equipment/overview", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusInternalServerError, resp.Code)
	var standardResponse response.StandardResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &standardResponse))
	require.Equal(t, response.StandardResponse{
		Status:  http.StatusInternalServerError,
		Message: ErrShopEquipmentOverviewUnavailable.Error(),
		Data:    nil,
	}, standardResponse)
	require.NotContains(t, resp.Body.String(), "database")
}

func TestGetShopEquipmentOverviewWritesGenericGzipErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler)
	router.Use(func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "user-1"}); c.Next() })
	handler := Handler{service: overviewServiceStub{err: ErrShopEquipmentOverviewUnavailable}}
	router.GET(
		"/shops/equipment/overview",
		ginGzip.Gzip(ginGzip.DefaultCompression),
		handler.GetShopEquipmentOverview,
	)

	req := httptest.NewRequest(http.MethodGet, "/shops/equipment/overview", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Equal(t, "gzip", resp.Header().Get("Content-Encoding"))
	gzipReader, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	decompressedBody, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())

	var standardResponse response.StandardResponse
	require.NoError(t, json.Unmarshal(decompressedBody, &standardResponse))
	require.Equal(t, response.StandardResponse{
		Status:  http.StatusInternalServerError,
		Message: ErrShopEquipmentOverviewUnavailable.Error(),
		Data:    nil,
	}, standardResponse)
	require.NotContains(t, string(decompressedBody), "database")
}

type shopsFailureStub struct{ ShopService }

func (shopsFailureStub) GetShopsByUser(context.Context, *bootstrap.User) ([]model.Shops, error) {
	return nil, errors.New("database private details")
}
func TestContractGetUserShopsPreservesLegacy404(t *testing.T) {
	for _, selector := range []string{"", "99", "2"} {
		r := gin.New()
		r.Use(middleware.ErrorHandler)
		g := r.Group("", shared.ContractMiddleware, func(c *gin.Context) { c.Set("user", &bootstrap.User{}) })
		handler := Handler{service: shopsFailureStub{}}
		g.GET("/shops", handler.GetUserShops)
		req := httptest.NewRequest("GET", "/shops", nil)
		req.Header.Set(shared.ContractHeader, selector)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var body response.StandardResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		if selector == "2" {
			require.Equal(t, 500, w.Code)
			require.Equal(t, "internal_error", body.Code)
		} else {
			require.Equal(t, 404, w.Code)
			require.Equal(t, "No item found", body.Message)
			require.Empty(t, body.Code)
		}
		require.NotContains(t, w.Body.String(), "private")
	}
}

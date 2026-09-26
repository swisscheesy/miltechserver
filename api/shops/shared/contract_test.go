package shared_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"miltechserver/api/middleware"
	"miltechserver/api/response"
	"miltechserver/api/shops/capabilities"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContractRouteErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, selector := range []string{"", "99", "2"} {
		for _, status := range []int{400, 401, 403, 404, 409, 422, 500, 503} {
			r := gin.New()
			r.Use(middleware.ErrorHandler)
			group := r.Group("", shared.ContractMiddleware)
			group.GET("/error", func(c *gin.Context) { response.Error(c, status, "invalid request") })
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/error", nil)
			req.Header.Set(shared.ContractHeader, selector)
			r.ServeHTTP(w, req)
			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != status {
				t.Fatalf("status %d", w.Code)
			}
			_, code := body["code"]
			if code != (selector == "2") {
				t.Fatalf("selector %q: %v", selector, body)
			}
		}
	}
}
func TestContractMiddlewareIsolation(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler)
	r.GET("/other", func(c *gin.Context) { c.Error(errors.New("legacy unrelated")) })
	r.Group("", shared.ContractMiddleware).GET("/shops", func(c *gin.Context) { c.Error(shared.ErrListNotFound) })
	for _, tc := range []struct {
		path, selector string
		status         int
		code           string
	}{{"/shops", "", 500, ""}, {"/shops", "99", 500, ""}, {"/shops", "2", 404, "list_not_found"}, {"/other", "2", 500, ""}, {"/missing", "2", 404, ""}} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", tc.path, nil)
		req.Header.Set(shared.ContractHeader, tc.selector)
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("%+v: %d", tc, w.Code)
		}
		if tc.path != "/missing" {
			var body response.StandardResponse
			json.Unmarshal(w.Body.Bytes(), &body)
			if body.Code != tc.code {
				t.Fatalf("%+v", body)
			}
		}
	}
}
func TestContractCapabilitiesDisabled(t *testing.T) {
	r := gin.New()
	g := r.Group("", shared.ContractMiddleware, func(c *gin.Context) { c.Set("user", &bootstrap.User{}) })
	capabilities.RegisterRoutes(g)
	for _, selector := range []string{"", "99", "2"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/shops/capabilities", nil)
		req.Header.Set(shared.ContractHeader, selector)
		r.ServeHTTP(w, req)
		if selector != "2" {
			if w.Code != 400 {
				t.Fatal(w.Code)
			}
			continue
		}
		var body struct{ Data map[string]interface{} }
		json.Unmarshal(w.Body.Bytes(), &body)
		if body.Data["contract_version"] != float64(2) {
			t.Fatal(body)
		}
		for _, key := range []string{"typed_errors", "atomic_notification_save", "service_dates", "service_reads", "message_sync"} {
			if body.Data[key] != false {
				t.Fatal(key)
			}
		}
	}
}

func TestContractAuthenticationBeforeRouteScope(t *testing.T) {
	r := gin.New()
	auth := r.Group("/api/v1/auth", middleware.AuthenticationMiddleware(nil))
	auth.Group("", shared.ContractMiddleware).GET("/shops/:shop_id", func(c *gin.Context) { t.Fatal("unauthorized handler reached") })
	auth.GET("/other", func(c *gin.Context) { t.Fatal("unauthorized handler reached") })
	for _, tc := range []struct{ path, selector, code string }{{"/api/v1/auth/shops/a", "2", "unauthorized"}, {"/api/v1/auth/shops/a", "99", ""}, {"/api/v1/auth/shops/a", "", ""}, {"/api/v1/auth/other", "2", ""}} {
		req := httptest.NewRequest("GET", tc.path, nil)
		req.Header.Set(shared.ContractHeader, tc.selector)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var body response.StandardResponse
		json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != 401 || body.Code != tc.code {
			t.Fatalf("%+v %+v", tc, body)
		}
	}
}

func TestContractLegacyPublicMessageDistinctions(t *testing.T) {
	for _, selector := range []string{"", "99", "2"} {
		r := gin.New()
		r.Use(middleware.ErrorHandler)
		r.Group("", shared.ContractMiddleware).GET("/error", func(c *gin.Context) {
			c.Error(fmt.Errorf("failed to get shop list: %w", errors.New("shop list not found")))
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/error", nil)
		req.Header.Set(shared.ContractHeader, selector)
		r.ServeHTTP(w, req)
		var body response.StandardResponse
		json.Unmarshal(w.Body.Bytes(), &body)
		expected := "failed to get shop list: shop list not found"
		if selector == "2" {
			expected = "shop list not found"
		}
		if body.Message != expected {
			t.Fatalf("%q: %q", selector, body.Message)
		}
	}
}

func TestContractSanitizesInternalErrorAndPreservesLegacyEnvelope(t *testing.T) {
	for _, selector := range []string{"", "99", "2"} {
		r := gin.New()
		r.Use(middleware.ErrorHandler)
		g := r.Group("", shared.ContractMiddleware)
		g.GET("/error", func(c *gin.Context) { c.Error(errors.New("driver password=secret table=private")) })
		g.GET("/validation", func(c *gin.Context) { shared.WriteValidationError(c, "invalid request") })
		for _, path := range []string{"/error", "/validation"} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set(shared.ContractHeader, selector)
			r.ServeHTTP(w, req)
			if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "private") {
				t.Fatal("internal details leaked")
			}
			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			_, hasCode := body["code"]
			if hasCode != (selector == "2") {
				t.Fatal(body)
			}
			if path == "/validation" && selector != "2" {
				if body["message"] != "invalid request" || body["details"] != "invalid request" {
					t.Fatal(body)
				}
			}
		}
	}
}

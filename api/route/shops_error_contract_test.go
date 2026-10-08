package route

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"miltechserver/api/middleware"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"miltechserver/internal/testsql"
)

func TestSetupWithNilConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	require.NotPanics(t, func() { Setup(nil, router, nil, nil, nil) })
}

// The real registered GET handler must serialize its repository error even
// though authenticated routes do not inherit the public group's ErrorHandler.
func TestProductionShopsQueuedErrorWritesOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, globalHandler := range []bool{false, true} {
		for _, selector := range []string{"", "99", "2"} {
			for _, firstQueuedError := range []bool{false, true} {
				t.Run(fmt.Sprintf("global=%t/selector=%q/first=%t", globalHandler, selector, firstQueuedError), func(t *testing.T) {
					db := testsql.Open(t, &testsql.Behavior{QueryErr: errors.New("private-error-sentinel")})
					router := gin.New()
					if globalHandler {
						router.Use(middleware.ErrorHandler)
					}
					authenticate := func(c *gin.Context) {
						c.Set("user", &bootstrap.User{UserID: "controlled-user"})
						if firstQueuedError {
							_ = c.Error(shared.ErrListNotFound)
						}
					}
					setupRoutes(db, router, authenticate, nil, nil, nil)
					request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/11111111-1111-1111-1111-111111111111", nil)
					request.Header.Set(shared.ContractHeader, selector)
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, request)
					if firstQueuedError && selector == "2" {
						require.Equal(t, http.StatusNotFound, recorder.Code)
						require.JSONEq(t, `{"status":404,"message":"list not found","code":"list_not_found","data":null}`, recorder.Body.String())
					} else {
						require.Equal(t, http.StatusInternalServerError, recorder.Code)
						if firstQueuedError {
							require.JSONEq(t, `{"status":500,"message":"list not found","data":null}`, recorder.Body.String())
						} else if selector == "2" {
							require.JSONEq(t, `{"status":500,"message":"Unable to complete Shops request","code":"internal_error","data":null}`, recorder.Body.String())
						} else {
							require.JSONEq(t, `{"status":500,"message":"Unable to complete Shops request","data":null}`, recorder.Body.String())
						}
					}
					require.NotContains(t, recorder.Body.String(), "private-error-sentinel")
				})
			}
		}
	}
}

func TestProductionShopsPreservesWrittenBody(t *testing.T) {
	const originalBody = `{"status":202,"message":"already accepted","data":{"id":"existing"}}`
	for _, globalHandler := range []bool{false, true} {
		for _, selector := range []string{"", "2"} {
			t.Run(fmt.Sprintf("global=%t/selector=%q", globalHandler, selector), func(t *testing.T) {
				db := testsql.Open(t, &testsql.Behavior{QueryErr: errors.New("private-error-sentinel")})
				router := gin.New()
				if globalHandler {
					router.Use(middleware.ErrorHandler)
				}
				setupRoutes(db, router, func(c *gin.Context) {
					c.Set("user", &bootstrap.User{UserID: "controlled-user"})
					c.Data(http.StatusAccepted, "application/json", []byte(originalBody))
				}, nil, nil, nil)
				request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/11111111-1111-1111-1111-111111111111", nil)
				request.Header.Set(shared.ContractHeader, selector)
				prewritten := httptest.NewRecorder()
				router.ServeHTTP(prewritten, request)
				require.Equal(t, http.StatusAccepted, prewritten.Code)
				require.JSONEq(t, originalBody, prewritten.Body.String())
			})
		}
	}
}

func TestProductionShopsNilConfigurationDisablesCapabilities(t *testing.T) {
	// A nil database makes an accidental readiness probe fail immediately.
	router := gin.New()
	setupRoutes(nil, router, func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "controlled-user"})
	}, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops/capabilities", nil)
	request.Header.Set(shared.ContractHeader, "2")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"status":200,"message":"","data":{"atomic_notification_save":false,"contract_version":2,"message_sync":false,"service_dates":false,"service_reads":false,"typed_errors":false}}`, recorder.Body.String())
}

func TestProductionShopsNegotiatedFailures(t *testing.T) {
	for _, selector := range []string{"", "2"} {
		for _, test := range []struct {
			name                         string
			queryErr                     error
			method, path, body           string
			legacyStatus, contractStatus int
			code                         string
		}{
			{"forbidden", nil, http.MethodGet, "/api/v1/auth/shops/11111111-1111-1111-1111-111111111111", "", 500, 403, "denied"},
			{"missing", shared.ErrShopNotFound, http.MethodGet, "/api/v1/auth/shops/11111111-1111-1111-1111-111111111111", "", 500, 404, "shop_not_found"},
			{"validation", nil, http.MethodPost, "/api/v1/auth/shops", "{", 400, 400, "invalid"},
		} {
			t.Run(test.name+"/selector="+selector, func(t *testing.T) {
				db := testsql.Open(t, &testsql.Behavior{Columns: []string{"exists"}, QueryErr: test.queryErr})
				router := gin.New()
				setupRoutes(db, router, func(c *gin.Context) {
					c.Set("user", &bootstrap.User{UserID: "controlled-user"})
				}, nil, nil, nil)
				request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set(shared.ContractHeader, selector)
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)
				if selector == "2" {
					require.Equal(t, test.contractStatus, recorder.Code)
					require.Contains(t, recorder.Body.String(), `"code":"`+test.code+`"`)
				} else {
					require.Equal(t, test.legacyStatus, recorder.Code)
					require.NotContains(t, recorder.Body.String(), `"code"`)
				}
			})
		}
	}
}

package user_general

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type serviceStub struct {
	upsertErr  error
	deleteErr  error
	updateErr  error
	deletedUID string
}

func (s *serviceStub) UpsertUser(*bootstrap.User, auth.UserDto) error {
	return s.upsertErr
}

func (s *serviceStub) DeleteUser(_ context.Context, uid string) error {
	s.deletedUID = uid
	return s.deleteErr
}

func (s *serviceStub) UpdateUserDisplayName(string, string) error {
	return s.updateErr
}

func TestUpsertUserUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	stub := &serviceStub{}

	registerHandlers(group, stub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/user/general/refresh", bytes.NewBufferString("{}"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestUpsertUserBadBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	stub := &serviceStub{}

	registerHandlers(group, stub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/user/general/refresh", bytes.NewBufferString("{"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDeleteUserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "user"})
		c.Next()
	})
	stub := &serviceStub{deleteErr: ErrUserNotFound}

	registerHandlers(group, stub)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/user/general/delete_user", bytes.NewBufferString(`{"uid":"user"}`))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestDeleteUserIgnoresBodyUIDAndUsesAuthenticatedUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "caller"})
		c.Next()
	})
	stub := &serviceStub{}
	registerHandlers(group, stub)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/auth/user/general/delete_user",
		strings.NewReader(`{"uid":"victim"}`),
	)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "caller", stub.deletedUID)
}

func TestDeleteUserAcceptsBodylessAuthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "caller"})
		c.Next()
	})
	stub := &serviceStub{}
	registerHandlers(group, stub)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/auth/user/general/delete_user",
		nil,
	)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "caller", stub.deletedUID)
}

func TestDeleteUserRejectsInvalidAuthenticatedContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		c.Set("user", "not-a-bootstrap-user")
		c.Next()
	})
	stub := &serviceStub{}
	registerHandlers(group, stub)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/auth/user/general/delete_user",
		nil,
	)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
	require.Empty(t, stub.deletedUID)
}

func TestUpdateDisplayNameInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	stub := &serviceStub{}

	registerHandlers(group, stub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/user/general/dn_change", bytes.NewBufferString("{"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

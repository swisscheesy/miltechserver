package pmcs_sbs_progress

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	listResp        *FaultListResponse
	faultResp       *FaultResponse
	err             error
	capturedUser    *bootstrap.User
	capturedRequest interface{}
}

func (s *serviceStub) ListFaults(user *bootstrap.User, equipmentID string) (*FaultListResponse, error) {
	s.capturedUser = user
	s.capturedRequest = equipmentID
	return s.listResp, s.err
}

func (s *serviceStub) UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error) {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         FaultRequest
	}{equipmentID: equipmentID, req: req}
	return s.faultResp, s.err
}

func (s *serviceStub) DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         DeleteFaultRequest
	}{equipmentID: equipmentID, req: req}
	return s.err
}

func TestHandlersRequireAuth(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, nil)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestHandlersRejectInvalidAuthContext(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
	}{
		{name: "wrong type", value: "user-1"},
		{name: "nil user", value: (*bootstrap.User)(nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			group := router.Group("/api/v1/auth")
			group.Use(func(c *gin.Context) {
				c.Set("user", tc.value)
				c.Next()
			})
			registerHandlers(group, &serviceStub{})

			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, routeUser())

			require.Equal(t, http.StatusUnauthorized, resp.Code)
		})
	}
}

func TestListFaultsSuccess(t *testing.T) {
	stub := &serviceStub{listResp: &FaultListResponse{Faults: []FaultResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "user-1", stub.capturedUser.UserID)
	require.Equal(t, "vehicle-1", stub.capturedRequest)
}

func TestUpsertFaultSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{faultResp: &FaultResponse{
		EquipmentID:      "vehicle-1",
		SectionID:        "before",
		ItemIndex:        0,
		ItemNo:           "1",
		Status:           "x",
		FaultText:        "leak",
		CorrectiveAction: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", FaultRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		Status:    "X",
		FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         FaultRequest
	})
	require.True(t, ok)
	require.Equal(t, "vehicle-1", captured.equipmentID)
	require.Equal(t, "X", captured.req.Status)

	var body struct {
		Status int           `json:"status"`
		Data   FaultResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "x", body.Data.Status)
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDeleteFaultSuccess(t *testing.T) {
	stub := &serviceStub{}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", DeleteFaultRequest{
		SectionID: "before",
		ItemIndex: 0,
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         DeleteFaultRequest
	})
	require.True(t, ok)
	require.Equal(t, "vehicle-1", captured.equipmentID)
	require.Equal(t, "before", captured.req.SectionID)
}

func TestServiceErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "unauthorized", err: ErrUnauthorized, want: http.StatusUnauthorized},
		{name: "invalid id", err: ErrInvalidID, want: http.StatusBadRequest},
		{name: "invalid request", err: ErrInvalidRequest, want: http.StatusBadRequest},
		{name: "invalid status", err: ErrInvalidStatus, want: http.StatusBadRequest},
		{name: "not found", err: ErrNotFound, want: http.StatusNotFound},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{err: tc.err})
			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/faults", nil, routeUser())
			require.Equal(t, tc.want, resp.Code)
		})
	}
}

func newRouteTestRouter(stub Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID != "" {
			c.Set("user", &bootstrap.User{UserID: userID, Email: userID + "@example.com", Username: userID})
		}
		c.Next()
	})
	registerHandlers(group, stub)
	return router
}

func routeUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Email: "user-1@example.com", Username: "user-1"}
}

func doRouteJSON(router *gin.Engine, method string, path string, body interface{}, user *bootstrap.User) *httptest.ResponseRecorder {
	var payload bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&payload).Encode(body)
		if err != nil {
			panic(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("X-User-ID", user.UserID)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

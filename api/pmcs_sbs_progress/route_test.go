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
	listResp        *EquipmentListResponse
	aggregateResp   *EquipmentAggregateResponse
	equipmentResp   *EquipmentResponse
	completionResp  *CompletionResponse
	batchResp       *BatchCompletionsResponse
	faultResp       *FaultResponse
	syncResp        *SyncResponse
	err             error
	capturedUser    *bootstrap.User
	capturedRequest interface{}
}

func (s *serviceStub) ListEquipment(user *bootstrap.User) (*EquipmentListResponse, error) {
	s.capturedUser = user
	return s.listResp, s.err
}

func (s *serviceStub) GetEquipment(user *bootstrap.User, equipmentID string) (*EquipmentAggregateResponse, error) {
	s.capturedUser = user
	s.capturedRequest = equipmentID
	return s.aggregateResp, s.err
}

func (s *serviceStub) UpsertEquipment(user *bootstrap.User, equipmentID string, req EquipmentRequest) (*EquipmentResponse, error) {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         EquipmentRequest
	}{equipmentID: equipmentID, req: req}
	return s.equipmentResp, s.err
}

func (s *serviceStub) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	s.capturedUser = user
	s.capturedRequest = equipmentID
	return s.err
}

func (s *serviceStub) UpsertCompletion(user *bootstrap.User, equipmentID string, req CompletionRequest) (*CompletionResponse, error) {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         CompletionRequest
	}{equipmentID: equipmentID, req: req}
	return s.completionResp, s.err
}

func (s *serviceStub) BatchCompletions(user *bootstrap.User, equipmentID string, req BatchCompletionsRequest) (*BatchCompletionsResponse, error) {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         BatchCompletionsRequest
	}{equipmentID: equipmentID, req: req}
	return s.batchResp, s.err
}

func (s *serviceStub) DeleteCompletion(user *bootstrap.User, equipmentID string, req DeleteCompletionRequest) error {
	s.capturedUser = user
	s.capturedRequest = struct {
		equipmentID string
		req         DeleteCompletionRequest
	}{equipmentID: equipmentID, req: req}
	return s.err
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

func (s *serviceStub) Sync(user *bootstrap.User, req SyncRequest) (*SyncResponse, error) {
	s.capturedUser = user
	s.capturedRequest = req
	return s.syncResp, s.err
}

func TestHandlersRequireAuth(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, nil)

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

			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())

			require.Equal(t, http.StatusUnauthorized, resp.Code)
		})
	}
}

func TestListEquipmentSuccess(t *testing.T) {
	stub := &serviceStub{listResp: &EquipmentListResponse{Equipment: []EquipmentResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "user-1", stub.capturedUser.UserID)
}

func TestGetEquipmentMapsNotFound(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{err: ErrNotFound})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", nil, routeUser())

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestUpsertEquipmentSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{equipmentResp: &EquipmentResponse{
		ID:              "550e8400-e29b-41d4-a716-446655440000",
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
		Serial:          "SER123",
		Nomenclature:    "Truck, Utility",
		Model:           "M1152A1",
		Uic:             "WABC01",
		CreatedAt:       now,
		UpdatedAt:       now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", EquipmentRequest{
		EquipmentManual: "pmcs_sbs/hmmwv/basic.json",
		Admin:           "A12",
		Serial:          "SER123",
		Nomenclature:    " Truck, Utility ",
		Model:           " M1152A1 ",
		Uic:             "WABC01",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         EquipmentRequest
	})
	require.True(t, ok)
	require.Equal(t, " Truck, Utility ", captured.req.Nomenclature)
	require.Equal(t, " M1152A1 ", captured.req.Model)

	var body struct {
		Status int               `json:"status"`
		Data   EquipmentResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "Truck, Utility", body.Data.Nomenclature)
	require.Equal(t, "M1152A1", body.Data.Model)
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUpsertCompletionSuccess(t *testing.T) {
	stub := &serviceStub{completionResp: &CompletionResponse{EquipmentID: "550e8400-e29b-41d4-a716-446655440000", SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a", IsComplete: true}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/completions", CompletionRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		StepID:    "1-a",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestBatchCompletionsSuccess(t *testing.T) {
	stub := &serviceStub{batchResp: &BatchCompletionsResponse{UpsertedCount: 2, DeletedCount: 1}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPatch, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/completions/batch", BatchCompletionsRequest{
		UpsertCompletions: []CompletionRequest{
			{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-a"},
			{SectionID: "before", ItemIndex: 0, ItemNo: "1", StepID: "1-b"},
		},
		DeleteCompletions: []DeleteCompletionRequest{
			{SectionID: "before", ItemIndex: 0, StepID: "1-c"},
		},
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(struct {
		equipmentID string
		req         BatchCompletionsRequest
	})
	require.True(t, ok)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", captured.equipmentID)
	require.Len(t, captured.req.UpsertCompletions, 2)
	require.Len(t, captured.req.DeleteCompletions, 1)
}

func TestUpsertFaultMapsInvalidStatus(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{err: ErrInvalidStatus})

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/550e8400-e29b-41d4-a716-446655440000/faults", FaultRequest{
		SectionID: "before",
		ItemIndex: 0,
		ItemNo:    "1",
		Status:    "BAD",
		FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestSyncSuccess(t *testing.T) {
	stub := &serviceStub{syncResp: &SyncResponse{DeletedEquipmentIDs: []string{}}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPost, "/api/v1/auth/pmcs-sbs/sync", SyncRequest{}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestUnexpectedServiceErrorReturnsGeneric500(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{err: errors.New("db exploded")})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.NotContains(t, resp.Body.String(), "db exploded")
}

func TestRouteTestRouterUsesAuthGroupAndInjectsRouteUser(t *testing.T) {
	stub := &serviceStub{listResp: &EquipmentListResponse{Equipment: []EquipmentResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, stub.capturedUser)
	require.Equal(t, routeUser(), stub.capturedUser)
}

func TestRegisterRoutesUsesAuthGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		c.Set("user", routeUser())
		c.Next()
	})
	registerHandlers(group, &serviceStub{listResp: &EquipmentListResponse{Equipment: []EquipmentResponse{}, Count: 0}})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment", nil, routeUser())
	require.Equal(t, http.StatusOK, resp.Code)
}

func newRouteTestRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/auth")
	group.Use(func(c *gin.Context) {
		if c.GetHeader("X-User-ID") != "" {
			c.Set("user", routeUser())
		}
		c.Next()
	})
	registerHandlers(group, svc)
	return router
}

func routeUser() *bootstrap.User {
	return &bootstrap.User{UserID: "user-1", Username: "tester", Email: "user-1@example.com"}
}

func doRouteJSON(router *gin.Engine, method string, path string, body interface{}, user *bootstrap.User) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("X-User-ID", user.UserID)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

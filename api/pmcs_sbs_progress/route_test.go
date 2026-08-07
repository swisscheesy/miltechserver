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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type serviceStub struct {
	inspectionResp *InspectionResponse
	listResp       *InspectionListResponse
	faultResp      *FaultResponse
	bulkDeleteResp *BulkDeleteFaultResponse
	commentResp    *CommentResponse
	err            error

	capturedUser        *bootstrap.User
	capturedEquipmentID string
	capturedPmcsID      string
	capturedCommentID   string
	capturedRequest     interface{}
}

func (s *serviceStub) EnsureInspection(user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.inspectionResp, s.err
}

func (s *serviceStub) GetInspection(user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	return s.inspectionResp, s.err
}

func (s *serviceStub) ListInspections(user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedRequest = req
	return s.listResp, s.err
}

func (s *serviceStub) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID string) error {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	return s.err
}

func (s *serviceStub) UpsertFault(user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.faultResp, s.err
}

func (s *serviceStub) DeleteFault(user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.err
}

func (s *serviceStub) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.bulkDeleteResp, s.err
}

func (s *serviceStub) CreateComment(user *bootstrap.User, equipmentID string, pmcsID string, req CreateCommentRequest) (*CommentResponse, error) {
	s.capturedUser = user
	s.capturedEquipmentID = equipmentID
	s.capturedPmcsID = pmcsID
	s.capturedRequest = req
	return s.commentResp, s.err
}

func (s *serviceStub) UpdateComment(user *bootstrap.User, commentID string, req UpdateCommentRequest) (*CommentResponse, error) {
	s.capturedUser = user
	s.capturedCommentID = commentID
	s.capturedRequest = req
	return s.commentResp, s.err
}

func (s *serviceStub) DeleteComment(user *bootstrap.User, commentID string) (*CommentResponse, error) {
	s.capturedUser = user
	s.capturedCommentID = commentID
	return s.commentResp, s.err
}

const routeTestPmcsID = "11111111-1111-1111-1111-111111111111"

func TestHandlersRequireAuth(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, nil)

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

			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

			require.Equal(t, http.StatusUnauthorized, resp.Code)
		})
	}
}

func TestUpsertInspectionSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{inspectionResp: &InspectionResponse{
		ID: uuid.MustParse(routeTestPmcsID), EquipmentID: "vehicle-1", GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"), PerformedDate: now, Faults: []FaultResponse{},
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, InspectionRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: now,
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, routeTestPmcsID, stub.capturedPmcsID)
}

func TestRouteInspectionLegacyGuideRequestReachesServiceAsGuideShaped(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{inspectionResp: &InspectionResponse{
		ID: uuid.MustParse(routeTestPmcsID), SourceType: "guide", GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"), PerformedDate: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, map[string]any{
		"guide_manual":   "pmcs_sbs/hmmwv/file.json",
		"performed_date": now.Format(time.RFC3339Nano),
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(InspectionRequest)
	require.True(t, ok)
	require.Equal(t, "", captured.SourceType)
	require.Equal(t, "pmcs_sbs/hmmwv/file.json", captured.GuideManual)
	require.Empty(t, captured.CustomChecklistID)
	require.Empty(t, captured.CustomRevisionID)
	require.Nil(t, captured.CustomRevisionNumber)
	require.Empty(t, captured.CustomChecklistName)
}

func TestRouteCustomFaultCarriesCompleteProvenanceAndSectionTitle(t *testing.T) {
	now := time.Now().UTC()
	revisionNumber := int32(7)
	stub := &serviceStub{faultResp: &FaultResponse{
		PmcsID: uuid.MustParse(routeTestPmcsID), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", FaultRequest{
		InspectionSourceRequest: InspectionSourceRequest{
			SourceType: "custom", CustomChecklistID: "22222222-2222-2222-2222-222222222222", CustomRevisionID: "33333333-3333-3333-3333-333333333333", CustomRevisionNumber: &revisionNumber, CustomChecklistName: "Weekly Generator PMCS",
		},
		PerformedDate: now, SectionID: "before", SectionTitle: "Before operation", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(FaultRequest)
	require.True(t, ok)
	require.Equal(t, "custom", captured.SourceType)
	require.Equal(t, "22222222-2222-2222-2222-222222222222", captured.CustomChecklistID)
	require.Equal(t, "33333333-3333-3333-3333-333333333333", captured.CustomRevisionID)
	require.Equal(t, &revisionNumber, captured.CustomRevisionNumber)
	require.Equal(t, "Weekly Generator PMCS", captured.CustomChecklistName)
	require.Equal(t, "Before operation", captured.SectionTitle)
}

func TestRouteResponsesKeepGuideAndCustomProvenanceMutuallyExclusive(t *testing.T) {
	now := time.Now().UTC()
	customChecklistID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	customRevisionID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	revisionNumber := int32(7)

	for _, tc := range []struct {
		name string
		resp *InspectionResponse
		want []string
		omit []string
	}{
		{
			name: "custom", resp: &InspectionResponse{ID: uuid.MustParse(routeTestPmcsID), SourceType: "custom", CustomChecklistID: &customChecklistID, CustomRevisionID: &customRevisionID, CustomRevisionNumber: &revisionNumber, CustomChecklistName: stringPointer("Weekly Generator PMCS"), PerformedDate: now},
			want: []string{"source_type", "custom_checklist_id", "custom_revision_id", "custom_revision_number", "custom_checklist_name"}, omit: []string{"guide_manual"},
		},
		{
			name: "guide", resp: &InspectionResponse{ID: uuid.MustParse(routeTestPmcsID), SourceType: "guide", GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"), PerformedDate: now},
			want: []string{"source_type", "guide_manual"}, omit: []string{"custom_checklist_id", "custom_revision_id", "custom_revision_number", "custom_checklist_name"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{inspectionResp: tc.resp})
			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

			require.Equal(t, http.StatusOK, resp.Code)
			var body struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
			for _, field := range tc.want {
				require.Contains(t, body.Data, field)
			}
			for _, field := range tc.omit {
				require.NotContains(t, body.Data, field)
			}
		})
	}
}

func TestRouteSourceValidationKeepsBadRequestEnvelope(t *testing.T) {
	stub := &serviceStub{err: ErrInvalidRequest}
	router := newRouteTestRouter(stub)
	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, InspectionRequest{}, routeUser())

	require.Equal(t, http.StatusBadRequest, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "invalid request", body["message"])
	require.NotContains(t, body, "data")
}

func TestRouteRejectsInvalidRawUTF8BeforeDecoding(t *testing.T) {
	paths := []string{
		"/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/" + routeTestPmcsID,
		"/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/" + routeTestPmcsID + "/faults",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{})
			resp := doRouteRawJSON(router, http.MethodPut, path, []byte{'{', '"', 's', 'o', 'u', 'r', 'c', 'e', '_', 't', 'y', 'p', 'e', '"', ':', '"', 0xff, '"', '}'}, routeUser())

			require.Equal(t, http.StatusBadRequest, resp.Code)
			require.JSONEq(t, `{"message":"invalid request body"}`, resp.Body.String())
		})
	}
}

func TestRouteRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	validInspection := []byte(`{"guide_manual":"pmcs_sbs/hmmwv/file.json","performed_date":"2026-08-07T00:00:00Z"}`)
	cases := []struct {
		name string
		body []byte
	}{
		{name: "unknown field", body: append(validInspection[:len(validInspection)-1], []byte(`,"unexpected":true}`)...)},
		{name: "trailing JSON", body: append(validInspection, []byte(` {}`)...)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{})
			resp := doRouteRawJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, tc.body, routeUser())

			require.Equal(t, http.StatusBadRequest, resp.Code)
			require.JSONEq(t, `{"message":"invalid request body"}`, resp.Body.String())
		})
	}
}

func TestGetInspectionSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{inspectionResp: &InspectionResponse{
		ID: uuid.MustParse(routeTestPmcsID), EquipmentID: "vehicle-1", GuideManual: stringPointer("pmcs_sbs/hmmwv/file.json"), PerformedDate: now,
		Faults: []FaultResponse{{PmcsID: uuid.MustParse(routeTestPmcsID), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now}},
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	var body struct {
		Data InspectionResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Len(t, body.Data.Faults, 1)
}

func TestListInspectionsSuccess(t *testing.T) {
	stub := &serviceStub{listResp: &InspectionListResponse{Inspections: []InspectionSummaryResponse{}, Count: 0}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs", nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
}

func TestDeleteInspectionSuccess(t *testing.T) {
	stub := &serviceStub{}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, routeTestPmcsID, stub.capturedPmcsID)
}

func TestUpsertFaultSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{faultResp: &FaultResponse{
		PmcsID: uuid.MustParse(routeTestPmcsID), SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "x", FaultText: "leak", CreatedAt: now, UpdatedAt: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", FaultRequest{
		InspectionSourceRequest: InspectionSourceRequest{GuideManual: "pmcs_sbs/hmmwv/file.json"}, PerformedDate: now, SectionID: "before", ItemIndex: 0, ItemNo: "1", Status: "X", FaultText: "leak",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(FaultRequest)
	require.True(t, ok)
	require.Equal(t, "X", captured.Status)

	var body struct {
		Data FaultResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "x", body.Data.Status)
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := newRouteTestRouter(&serviceStub{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDeleteFaultSuccess(t *testing.T) {
	stub := &serviceStub{}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults", DeleteFaultRequest{
		SectionID: "before", ItemIndex: 0,
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	captured, ok := stub.capturedRequest.(DeleteFaultRequest)
	require.True(t, ok)
	require.Equal(t, "before", captured.SectionID)
}

func TestBulkDeleteFaultsSuccess(t *testing.T) {
	stub := &serviceStub{bulkDeleteResp: &BulkDeleteFaultResponse{RequestedCount: 2, DeletedCount: 1}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/faults/bulk", BulkDeleteFaultRequest{
		Faults: []BulkDeleteFaultItemRequest{
			{SectionID: "before", ItemIndex: 0},
			{SectionID: "after", ItemIndex: 2},
		},
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)

	var body struct {
		Message        string `json:"message"`
		RequestedCount int    `json:"requested_count"`
		DeletedCount   int    `json:"deleted_count"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "Faults deleted", body.Message)
	require.Equal(t, 2, body.RequestedCount)
	require.Equal(t, 1, body.DeletedCount)
}

func TestCreateCommentSuccess(t *testing.T) {
	now := time.Now().UTC()
	stub := &serviceStub{commentResp: &CommentResponse{
		ID: uuid.New(), PmcsID: uuid.MustParse(routeTestPmcsID), AuthorID: "user-1", Text: "looks good", CreatedAt: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPost, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/comments", CreateCommentRequest{
		Text: "looks good",
	}, routeUser())

	require.Equal(t, http.StatusCreated, resp.Code)
	require.Equal(t, "vehicle-1", stub.capturedEquipmentID)
	require.Equal(t, routeTestPmcsID, stub.capturedPmcsID)
	captured, ok := stub.capturedRequest.(CreateCommentRequest)
	require.True(t, ok)
	require.Equal(t, "looks good", captured.Text)
}

func TestUpdateCommentSuccess(t *testing.T) {
	commentID := uuid.New().String()
	now := time.Now().UTC()
	stub := &serviceStub{commentResp: &CommentResponse{
		ID: uuid.MustParse(commentID), PmcsID: uuid.MustParse(routeTestPmcsID), AuthorID: "user-1", Text: "edited", CreatedAt: now,
	}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodPut, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/comments/"+commentID, UpdateCommentRequest{
		Text: "edited",
	}, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, commentID, stub.capturedCommentID)
}

func TestDeleteCommentSuccess(t *testing.T) {
	commentID := uuid.New().String()
	stub := &serviceStub{commentResp: &CommentResponse{ID: uuid.MustParse(commentID)}}
	router := newRouteTestRouter(stub)

	resp := doRouteJSON(router, http.MethodDelete, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID+"/comments/"+commentID, nil, routeUser())

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, commentID, stub.capturedCommentID)
}

func TestServiceErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "unauthorized", err: ErrUnauthorized, want: http.StatusUnauthorized},
		{name: "invalid id", err: ErrInvalidID, want: http.StatusBadRequest},
		{name: "invalid pmcs id", err: ErrInvalidPmcsID, want: http.StatusBadRequest},
		{name: "invalid guide manual", err: ErrInvalidGuideManual, want: http.StatusBadRequest},
		{name: "invalid request", err: ErrInvalidRequest, want: http.StatusBadRequest},
		{name: "invalid status", err: ErrInvalidStatus, want: http.StatusBadRequest},
		{name: "inspection conflict", err: ErrInspectionConflict, want: http.StatusConflict},
		{name: "not found", err: ErrNotFound, want: http.StatusNotFound},
		{name: "inspection not found", err: ErrInspectionNotFound, want: http.StatusNotFound},
		{name: "comment not found", err: ErrCommentNotFound, want: http.StatusNotFound},
		{name: "invalid comment text", err: ErrInvalidCommentText, want: http.StatusBadRequest},
		{name: "forbidden", err: ErrForbidden, want: http.StatusForbidden},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newRouteTestRouter(&serviceStub{err: tc.err})
			resp := doRouteJSON(router, http.MethodGet, "/api/v1/auth/pmcs-sbs/equipment/vehicle-1/pmcs/"+routeTestPmcsID, nil, routeUser())
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

func doRouteRawJSON(router *gin.Engine, method string, path string, body []byte, user *bootstrap.User) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req.Header.Set("X-User-ID", user.UserID)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

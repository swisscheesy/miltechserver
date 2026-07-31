package shared_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseExistingPrecondition(t *testing.T) {
	current := shared.MakeChecklistETag(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), 7)

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{name: "accepts one strong ETag", header: current},
		{name: "rejects missing header", wantErr: true},
		{name: "rejects weak ETag", header: "W/" + current, wantErr: true},
		{name: "rejects multiple ETags", header: current + ", \"other\"", wantErr: true},
		{name: "rejects malformed quoted value", header: "not-quoted", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := shared.ParseExistingPrecondition(test.header)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, shared.PreconditionMatch, got.Mode)
			require.Equal(t, current, got.ETag)
			require.True(t, got.Matches(current))
			require.False(t, got.Matches(shared.MakeChecklistETag(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), 7)))
		})
	}
}

func TestParseCreatePrecondition(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{name: "accepts wildcard", header: "*"},
		{name: "rejects missing header", wantErr: true},
		{name: "rejects strong ETag", header: "\"current\"", wantErr: true},
		{name: "rejects multiple values", header: "*, \"current\"", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := shared.ParseCreatePrecondition(test.header)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, shared.PreconditionCreate, got.Mode)
			require.Empty(t, got.ETag)
		})
	}
}

func TestPreconditionErrorsAreTyped(t *testing.T) {
	tests := []struct {
		name       string
		parse      func(string) (shared.Precondition, error)
		header     string
		wantStatus int
		wantCode   string
	}{
		{name: "missing existing precondition requires a header", parse: shared.ParseExistingPrecondition, wantStatus: http.StatusPreconditionRequired, wantCode: "precondition_required"},
		{name: "malformed existing precondition is invalid", parse: shared.ParseExistingPrecondition, header: "not-quoted", wantStatus: http.StatusBadRequest, wantCode: "invalid_precondition"},
		{name: "missing create precondition requires a header", parse: shared.ParseCreatePrecondition, wantStatus: http.StatusPreconditionRequired, wantCode: "precondition_required"},
		{name: "malformed create precondition is invalid", parse: shared.ParseCreatePrecondition, header: "\"current\"", wantStatus: http.StatusBadRequest, wantCode: "invalid_precondition"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.parse(test.header)
			var apiError *shared.APIError
			require.ErrorAs(t, err, &apiError)
			require.Equal(t, test.wantStatus, apiError.Status)
			require.Equal(t, test.wantCode, apiError.Code)
		})
	}
}

func TestDecodeStrictJSON(t *testing.T) {
	type request struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name            string
		contentType     string
		contentEncoding string
		body            []byte
		maxBytes        int64
		wantStatus      int
		wantCode        string
		wantName        string
	}{
		{name: "decodes one application JSON value", contentType: "application/json", body: []byte(`{"name":"checklist"}`), maxBytes: 1024, wantName: "checklist"},
		{name: "rejects unknown field", contentType: "application/json", body: []byte(`{"name":"checklist","unexpected":true}`), maxBytes: 1024, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "rejects trailing JSON", contentType: "application/json", body: []byte(`{"name":"checklist"} {}`), maxBytes: 1024, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "rejects compressed body", contentType: "application/json", contentEncoding: "gzip", body: []byte(`{"name":"checklist"}`), maxBytes: 1024, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "rejects non JSON media type", contentType: "text/plain", body: []byte(`{"name":"checklist"}`), maxBytes: 1024, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "rejects invalid UTF-8", contentType: "application/json", body: []byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'}, maxBytes: 1024, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "rejects bodies above the configured limit", contentType: "application/json", body: []byte(`{"name":"checklist"}`), maxBytes: 4, wantStatus: http.StatusRequestEntityTooLarge, wantCode: "content_too_large"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(test.body))
			context.Request.Header.Set("Content-Type", test.contentType)
			if test.contentEncoding != "" {
				context.Request.Header.Set("Content-Encoding", test.contentEncoding)
			}

			var got request
			err := shared.DecodeStrictJSON(context, &got, test.maxBytes)
			if test.wantStatus != 0 {
				require.Equal(t, test.wantStatus, err.Status)
				require.Equal(t, test.wantCode, err.Code)
				return
			}

			require.Nil(t, err)
			require.Equal(t, test.wantName, got.Name)
		})
	}
}

func TestWriteAPIErrorOmitsCauseAndEmptyDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	shared.WriteAPIError(context, &shared.APIError{
		Status:  http.StatusPreconditionFailed,
		Code:    "stale_precondition",
		Message: "resource changed",
		Cause:   errors.New("sensitive backend failure"),
	})

	require.Equal(t, http.StatusPreconditionFailed, recorder.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, float64(http.StatusPreconditionFailed), body["status"])
	require.Equal(t, "resource changed", body["message"])
	require.Nil(t, body["data"])
	errorBody := body["error"].(map[string]any)
	require.Equal(t, "stale_precondition", errorBody["code"])
	require.NotContains(t, recorder.Body.String(), "sensitive backend failure")
	_, hasDetails := errorBody["details"]
	require.False(t, hasDetails)
}

func TestDefaultConfigAndConfigFromEnv(t *testing.T) {
	want := shared.Config{
		MaxOwnedChecklists:     250,
		MaxActiveSubscriptions: 500,
		MaxChecklistModels:     100,
		MaxSections:            100,
		MaxSectionModels:       100,
		MaxSectionModelsTotal:  1000,
		MaxItemsPerSection:     500,
		MaxItemsTotal:          2000,
		MaxNoticesPerItem:      100,
		MaxNoticesTotal:        4000,
		MaxStepsPerItem:        250,
		MaxStepsTotal:          10000,
		MaxMutationBodyBytes:   8 * 1024 * 1024,
		MaxDeltaResponseBytes:  20 * 1024 * 1024,
		DeltaDefaultLimit:      10,
		DeltaMaxLimit:          25,
		UpdatesDefaultLimit:    50,
		UpdatesMaxLimit:        100,
		CommunityDefaultLimit:  20,
		CommunityMaxLimit:      50,
		TransactionMaxAttempts: 3,

		PublicRequestsPerSecond:         2,
		PublicRequestBurst:              20,
		AuthenticatedReadsPerSecond:     10,
		AuthenticatedReadBurst:          30,
		AuthenticatedMutationsPerSecond: 2,
		AuthenticatedMutationBurst:      10,
		ReleasesPerUserPerHour:          12,
		ReleaseUserBurst:                3,
		ReleasesPerIPPerHour:            60,
		ReleaseIPBurst:                  10,
		LimiterIdleMinutes:              15,
	}
	require.Equal(t, want, shared.DefaultConfig())

	config, err := shared.ConfigFromEnv(&bootstrap.Env{UserPmcs: bootstrap.UserPmcsConfig{
		MaxOwnedChecklists:     1,
		MaxActiveSubscriptions: 2,
		MaxChecklistModels:     3,
		MaxSections:            4,
		MaxSectionModels:       5,
		MaxSectionModelsTotal:  6,
		MaxItemsPerSection:     7,
		MaxItemsTotal:          8,
		MaxNoticesPerItem:      9,
		MaxNoticesTotal:        10,
		MaxStepsPerItem:        11,
		MaxStepsTotal:          12,
		MaxMutationBodyBytes:   13,
		MaxDeltaResponseBytes:  14,
		DeltaDefaultLimit:      15,
		DeltaMaxLimit:          16,
		UpdatesDefaultLimit:    17,
		UpdatesMaxLimit:        18,
		CommunityDefaultLimit:  19,
		CommunityMaxLimit:      20,
		TransactionMaxAttempts: 21,

		PublicRequestsPerSecond:         22,
		PublicRequestBurst:              23,
		AuthenticatedReadsPerSecond:     24,
		AuthenticatedReadBurst:          25,
		AuthenticatedMutationsPerSecond: 26,
		AuthenticatedMutationBurst:      27,
		ReleasesPerUserPerHour:          28,
		ReleaseUserBurst:                29,
		ReleasesPerIPPerHour:            30,
		ReleaseIPBurst:                  31,
		LimiterIdleMinutes:              32,
	}})
	require.NoError(t, err)
	require.Equal(t, 1, config.MaxOwnedChecklists)
	require.Equal(t, int64(13), config.MaxMutationBodyBytes)
	require.Equal(t, 21, config.TransactionMaxAttempts)
	require.Equal(t, 22, config.PublicRequestsPerSecond)
	require.Equal(t, 32, config.LimiterIdleMinutes)

	_, err = shared.ConfigFromEnv(&bootstrap.Env{UserPmcs: bootstrap.UserPmcsConfig{MaxOwnedChecklists: -1}})
	require.Error(t, err)
}

package user_pmcs

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"miltechserver/bootstrap"
)

func TestSlogObserverEmitsOnlyStructuralObservationFields(t *testing.T) {
	var output bytes.Buffer
	observer := newSlogObserver(slog.New(slog.NewJSONHandler(&output, nil)))

	observer.Observe(Observation{
		Operation:      "PUT /api/v1/auth/user-pmcs/checklists/:checklist_id",
		Status:         http.StatusUnprocessableEntity,
		Code:           "validation_failed",
		Duration:       125 * time.Millisecond,
		DBDuration:     75 * time.Millisecond,
		EncodeDuration: 5 * time.Millisecond,
		RetryCount:     2,
		NodeCount:      45,
		RequestBytes:   1024,
		ResponseBytes:  256,
	})

	var event map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &event))
	require.Equal(t, "user PMCS operation", event["msg"])
	require.Equal(t, "PUT /api/v1/auth/user-pmcs/checklists/:checklist_id", event["operation"])
	require.Equal(t, float64(http.StatusUnprocessableEntity), event["status"])
	require.Equal(t, "validation_failed", event["code"])
	require.Equal(t, float64(2), event["retry_count"])
	require.Equal(t, float64(45), event["node_count"])
	require.Equal(t, float64(1024), event["request_bytes"])
	require.Equal(t, float64(256), event["response_bytes"])
	require.Contains(t, event, "duration")
	require.Contains(t, event, "db_duration")
	require.Contains(t, event, "encode_duration")

	encoded := output.String()
	require.NotContains(t, encoded, "checklist title")
	require.NotContains(t, encoded, "person@example.mil")
	require.NotContains(t, encoded, "Bearer secret-token")
}

func TestObservationMiddlewareDoesNotLogBodiesAuthClaimsOrEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const (
		authoredBody = `{"name":"authored checklist secret"}`
		email        = "person@example.mil"
		token        = "Bearer secret-token"
	)
	var output bytes.Buffer
	observer := newSlogObserver(slog.New(slog.NewJSONHandler(&output, nil)))
	clock := &fakeClock{now: time.Unix(5_000, 0)}

	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{
			UserID: "verified-user",
			Email:  email,
		})
		context.Next()
	})
	router.Use(observeRequests(observer, clock.Now))
	router.PUT(
		"/api/v1/auth/user-pmcs/checklists/:checklist_id",
		func(context *gin.Context) {
			clock.Advance(25 * time.Millisecond)
			context.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "authored response secret",
			})
		},
	)

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		strings.NewReader(authoredBody),
	)
	request.Header.Set("Authorization", token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	var event map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &event))
	require.Equal(t, "PUT /api/v1/auth/user-pmcs/checklists/:checklist_id", event["operation"])
	require.Equal(t, float64(http.StatusUnprocessableEntity), event["status"])
	require.Equal(t, float64(len(authoredBody)), event["request_bytes"])
	require.Greater(t, event["response_bytes"].(float64), float64(0))
	require.Equal(t, float64(25*time.Millisecond), event["duration"])

	encoded := output.String()
	require.NotContains(t, encoded, authoredBody)
	require.NotContains(t, encoded, "authored checklist secret")
	require.NotContains(t, encoded, "authored response secret")
	require.NotContains(t, encoded, email)
	require.NotContains(t, encoded, token)
}

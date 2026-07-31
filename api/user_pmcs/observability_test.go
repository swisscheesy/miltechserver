package user_pmcs

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
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

func TestObservationMiddlewareCapturesAPIErrorCodeAndEncodingDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	observer := &capturingObserver{}
	router := gin.New()
	router.Use(observeRequests(observer, time.Now))
	router.POST("/validation", func(context *gin.Context) {
		shared.WriteAPIError(
			context,
			shared.NewValidationFailed(
				"authored response secret",
				map[string]any{"field": "authored field secret"},
			),
		)
	})

	request := httptest.NewRequest(http.MethodPost, "/validation", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	observation := observer.single(t)
	require.Equal(t, "validation_failed", observation.Code)
	require.Equal(t, http.StatusUnprocessableEntity, observation.Status)
	require.Positive(t, observation.EncodeDuration)
	require.Zero(t, observation.DBDuration)
	require.Zero(t, observation.RetryCount)
	require.Zero(t, observation.NodeCount)
}

func TestObservationMiddlewareCapturesRetryingMutationMeasurements(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openObservationDatabase(t)
	observer := &capturingObserver{}
	router := gin.New()
	router.Use(observeRequests(observer, time.Now))
	router.PUT("/mutation", func(context *gin.Context) {
		attempts := 0
		_, err := persistence.WithWriteTx(
			context.Request.Context(),
			database,
			2,
			func(_ *sql.Tx) (struct{}, error) {
				attempts++
				if attempts == 1 {
					return struct{}{}, &pq.Error{
						Code: pq.ErrorCode("40P01"),
					}
				}
				return struct{}{}, nil
			},
		)
		require.NoError(t, err)
		shared.WriteJSON(context, http.StatusOK, gin.H{"status": "saved"})
	})

	request := httptest.NewRequest(http.MethodPut, "/mutation", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	observation := observer.single(t)
	require.Positive(t, observation.DBDuration)
	require.Positive(t, observation.EncodeDuration)
	require.Equal(t, 1, observation.RetryCount)
}

func TestObservationMiddlewareCapturesFullTreeResponseNodeCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	observer := &capturingObserver{}
	router := gin.New()
	router.Use(observeRequests(observer, time.Now))
	router.GET("/full-tree", func(context *gin.Context) {
		shared.WriteJSON(context, http.StatusOK, shared.ChecklistAggregate{
			Draft: &shared.Revision{
				ID:     uuid.New(),
				Models: []shared.ModelValue{{DisplayText: "M1"}},
				Sections: []shared.Section{{
					ID:     uuid.New(),
					Models: []shared.ModelValue{{DisplayText: "section M1"}},
					Items: []shared.Item{{
						ID:             uuid.New(),
						Notices:        []shared.NoticeInput{{ID: uuid.New()}},
						ProcedureSteps: []shared.ProcedureStepInput{{ID: uuid.New()}},
					}},
				}},
			},
		})
	})

	request := httptest.NewRequest(http.MethodGet, "/full-tree", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	observation := observer.single(t)
	require.Equal(t, 7, observation.NodeCount)
	require.Positive(t, observation.EncodeDuration)
}

type capturingObserver struct {
	observations []Observation
}

func (observer *capturingObserver) Observe(observation Observation) {
	observer.observations = append(observer.observations, observation)
}

func (observer *capturingObserver) single(t *testing.T) Observation {
	t.Helper()
	require.Len(t, observer.observations, 1)
	return observer.observations[0]
}

type observationDriver struct{}

type observationConnection struct{}

type observationTransaction struct{}

func (observationDriver) Open(string) (driver.Conn, error) {
	return observationConnection{}, nil
}

func (observationConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (observationConnection) Close() error {
	return nil
}

func (observationConnection) Begin() (driver.Tx, error) {
	return observationTransaction{}, nil
}

func (observationConnection) BeginTx(
	context.Context,
	driver.TxOptions,
) (driver.Tx, error) {
	return observationTransaction{}, nil
}

func (observationTransaction) Commit() error {
	return nil
}

func (observationTransaction) Rollback() error {
	return nil
}

var observationDriverCounter atomic.Uint64
var observationDriverRegistrationMu sync.Mutex

func openObservationDatabase(t *testing.T) *sql.DB {
	t.Helper()

	observationDriverRegistrationMu.Lock()
	driverName := fmt.Sprintf(
		"user-pmcs-observation-%d",
		observationDriverCounter.Add(1),
	)
	sql.Register(driverName, observationDriver{})
	observationDriverRegistrationMu.Unlock()

	database, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return database
}

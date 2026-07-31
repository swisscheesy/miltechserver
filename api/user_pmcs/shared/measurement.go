package shared

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RequestMeasurements struct {
	mu             sync.Mutex
	dbDuration     time.Duration
	encodeDuration time.Duration
	retryCount     int
	nodeCount      int
	code           string
}

type MeasurementSnapshot struct {
	DBDuration     time.Duration
	EncodeDuration time.Duration
	RetryCount     int
	NodeCount      int
	Code           string
}

type measurementContextKey struct{}

func WithRequestMeasurements(
	ctx context.Context,
) (context.Context, *RequestMeasurements) {
	measurements := &RequestMeasurements{}
	return context.WithValue(ctx, measurementContextKey{}, measurements), measurements
}

func (measurements *RequestMeasurements) Snapshot() MeasurementSnapshot {
	if measurements == nil {
		return MeasurementSnapshot{}
	}
	measurements.mu.Lock()
	defer measurements.mu.Unlock()
	return MeasurementSnapshot{
		DBDuration:     measurements.dbDuration,
		EncodeDuration: measurements.encodeDuration,
		RetryCount:     measurements.retryCount,
		NodeCount:      measurements.nodeCount,
		Code:           measurements.code,
	}
}

func RecordDBDuration(ctx context.Context, duration time.Duration) {
	if measurements := requestMeasurements(ctx); measurements != nil {
		measurements.mu.Lock()
		measurements.dbDuration += duration
		measurements.mu.Unlock()
	}
}

func RecordEncodeDuration(ctx context.Context, duration time.Duration) {
	if measurements := requestMeasurements(ctx); measurements != nil {
		measurements.mu.Lock()
		measurements.encodeDuration += duration
		measurements.mu.Unlock()
	}
}

func RecordRetry(ctx context.Context) {
	if measurements := requestMeasurements(ctx); measurements != nil {
		measurements.mu.Lock()
		measurements.retryCount++
		measurements.mu.Unlock()
	}
}

func RecordNodeCount(ctx context.Context, count int) {
	if count <= 0 {
		return
	}
	if measurements := requestMeasurements(ctx); measurements != nil {
		measurements.mu.Lock()
		measurements.nodeCount = count
		measurements.mu.Unlock()
	}
}

func RecordErrorCode(ctx context.Context, code string) {
	if measurements := requestMeasurements(ctx); measurements != nil {
		measurements.mu.Lock()
		measurements.code = code
		measurements.mu.Unlock()
	}
}

func WriteJSON(context *gin.Context, status int, value any) {
	RecordNodeCount(
		requestContext(context),
		TreeNodeCount(value),
	)
	startedAt := time.Now()
	context.JSON(status, value)
	RecordEncodeDuration(
		requestContext(context),
		time.Since(startedAt),
	)
}

func requestContext(ginContext *gin.Context) context.Context {
	if ginContext == nil || ginContext.Request == nil {
		return context.Background()
	}
	return ginContext.Request.Context()
}

func requestMeasurements(ctx context.Context) *RequestMeasurements {
	measurements, _ := ctx.Value(measurementContextKey{}).(*RequestMeasurements)
	return measurements
}

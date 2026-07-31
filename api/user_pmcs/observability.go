package user_pmcs

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"miltechserver/api/user_pmcs/shared"
)

type Observation struct {
	Operation      string
	Status         int
	Code           string
	Duration       time.Duration
	DBDuration     time.Duration
	EncodeDuration time.Duration
	RetryCount     int
	NodeCount      int
	RequestBytes   int64
	ResponseBytes  int
}

type Observer interface {
	Observe(Observation)
}

type slogObserver struct {
	logger *slog.Logger
}

func newSlogObserver(logger *slog.Logger) Observer {
	if logger == nil {
		logger = slog.Default()
	}
	return slogObserver{logger: logger}
}

func defaultObserver() Observer {
	return newSlogObserver(slog.Default())
}

func (observer slogObserver) Observe(observation Observation) {
	observer.logger.Info(
		"user PMCS operation",
		"operation", observation.Operation,
		"status", observation.Status,
		"code", observation.Code,
		"duration", observation.Duration,
		"db_duration", observation.DBDuration,
		"encode_duration", observation.EncodeDuration,
		"retry_count", observation.RetryCount,
		"node_count", observation.NodeCount,
		"request_bytes", observation.RequestBytes,
		"response_bytes", observation.ResponseBytes,
	)
}

func observeRequests(
	observer Observer,
	now func() time.Time,
) gin.HandlerFunc {
	if observer == nil {
		observer = defaultObserver()
	}
	if now == nil {
		now = time.Now
	}
	return func(context *gin.Context) {
		startedAt := now()
		requestContext, measurements := shared.WithRequestMeasurements(
			context.Request.Context(),
		)
		context.Request = context.Request.WithContext(requestContext)
		context.Next()

		requestBytes := context.Request.ContentLength
		if requestBytes < 0 {
			requestBytes = 0
		}
		responseBytes := context.Writer.Size()
		if responseBytes < 0 {
			responseBytes = 0
		}
		measurement := measurements.Snapshot()
		observer.Observe(Observation{
			Operation:      context.Request.Method + " " + context.FullPath(),
			Status:         context.Writer.Status(),
			Code:           measurement.Code,
			Duration:       now().Sub(startedAt),
			DBDuration:     measurement.DBDuration,
			EncodeDuration: measurement.EncodeDuration,
			RetryCount:     measurement.RetryCount,
			NodeCount:      measurement.NodeCount,
			RequestBytes:   requestBytes,
			ResponseBytes:  responseBytes,
		})
	}
}

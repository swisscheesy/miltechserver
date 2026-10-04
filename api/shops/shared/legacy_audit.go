package shared

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"log/slog"
)

type auditCorrelationContextKey struct{}

// WithAuditCorrelation keeps one server-minted identifier for the entire mutation flow.
func WithAuditCorrelation(ctx context.Context) context.Context {
	if _, ok := ctx.Value(auditCorrelationContextKey{}).(string); ok {
		return ctx
	}
	return context.WithValue(ctx, auditCorrelationContextKey{}, uuid.NewString())
}

// WarnLegacyAuditFailure reports delivery gaps without exposing database errors or item values.
func WarnLegacyAuditFailure(ctx context.Context, operation, shopID, vehicleID, notificationID, actorID string, err error) {
	ctx = WithAuditCorrelation(ctx)
	category := "database_failure"
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		category = "request_canceled"
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		category = "request_deadline"
	}
	slog.WarnContext(ctx, "legacy_notification_audit_failed", "operation", operation, "shop_id", shopID, "vehicle_id", vehicleID, "notification_id", notificationID, "actor_id", actorID, "correlation_id", ctx.Value(auditCorrelationContextKey{}), "failure_category", category)
}

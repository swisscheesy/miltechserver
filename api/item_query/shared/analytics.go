package shared

import "context"

type AnalyticsTracker interface {
	IncrementItemSearchSuccess(ctx context.Context, niin string, nomenclature string) error
}

type NoOpTracker struct{}

func (NoOpTracker) IncrementItemSearchSuccess(context.Context, string, string) error {
	return nil
}

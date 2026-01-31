package shared

type AnalyticsTracker interface {
	IncrementItemSearchSuccess(niin string, nomenclature string) error
}

type NoOpTracker struct{}

func (NoOpTracker) IncrementItemSearchSuccess(string, string) error {
	return nil
}

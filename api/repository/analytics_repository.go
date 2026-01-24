package repository

type AnalyticsRepository interface {
	IncrementCounter(eventType string, entityKey string, entityLabel string) error
}

package service

type AnalyticsService interface {
	IncrementItemSearchSuccess(niin string) error
	IncrementPMCSManualDownload(entityKey string, entityLabel string) error
	IncrementCounter(eventType string, entityKey string, entityLabel string) error
}

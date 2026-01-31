package analytics

type Repository interface {
	IncrementCounter(eventType string, entityKey string, entityLabel string) error
}

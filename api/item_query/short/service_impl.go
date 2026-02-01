package short

import (
	"log/slog"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/item_query/shared"
)

type analyticsEvent struct {
	niin         string
	nomenclature string
}

type ServiceImpl struct {
	repo       Repository
	analytics  shared.AnalyticsTracker
	analyticsQ chan analyticsEvent
}

func NewService(repo Repository, analytics shared.AnalyticsTracker) *ServiceImpl {
	s := &ServiceImpl{
		repo:       repo,
		analytics:  analytics,
		analyticsQ: make(chan analyticsEvent, 100),
	}
	go s.processAnalytics()
	return s
}

// processAnalytics runs in the background, processing analytics events without blocking requests.
func (s *ServiceImpl) processAnalytics() {
	for event := range s.analyticsQ {
		if err := s.analytics.IncrementItemSearchSuccess(event.niin, event.nomenclature); err != nil {
			slog.Warn("Failed to increment analytics for item search", "niin", event.niin, "error", err)
		}
	}
}

func (service *ServiceImpl) FindShortByNiin(niin string) (model.NiinLookup, error) {
	val, err := service.repo.ShortItemSearchNiin(niin)
	if err != nil {
		return model.NiinLookup{}, err
	}

	normalizedNiin := normalizeNiinPointer(val.Niin, niin)
	nomenclature := normalizeNiinPointer(val.ItemName, "")
	service.trackItemSearchSuccess(normalizedNiin, nomenclature)
	return val, nil
}

func (service *ServiceImpl) FindShortByPart(part string) ([]model.NiinLookup, error) {
	results, err := service.repo.ShortItemSearchPart(part)
	if err != nil {
		return []model.NiinLookup{}, err
	}

	uniqueNiins := make(map[string]string)
	for _, result := range results {
		normalizedNiin := normalizeNiinPointer(result.Niin, "")
		if normalizedNiin == "" {
			continue
		}
		if _, exists := uniqueNiins[normalizedNiin]; exists {
			continue
		}
		nomenclature := normalizeNiinPointer(result.ItemName, "")
		uniqueNiins[normalizedNiin] = nomenclature
	}

	for normalizedNiin, nomenclature := range uniqueNiins {
		service.trackItemSearchSuccess(normalizedNiin, nomenclature)
	}

	return results, nil
}

func normalizeNiinPointer(niin *string, fallback string) string {
	if niin != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*niin))
		if normalized != "" {
			return normalized
		}
	}

	normalizedFallback := strings.ToUpper(strings.TrimSpace(fallback))
	if normalizedFallback == "" {
		return ""
	}
	return normalizedFallback
}

// trackItemSearchSuccess sends analytics events to a buffered channel for async processing.
// If the queue is full, events are dropped to prevent blocking the request.
func (service *ServiceImpl) trackItemSearchSuccess(niin string, nomenclature string) {
	if service.analytics == nil || niin == "" {
		return
	}
	select {
	case service.analyticsQ <- analyticsEvent{niin: niin, nomenclature: nomenclature}:
	default:
		slog.Warn("Analytics queue full, dropping event", "niin", niin)
	}
}

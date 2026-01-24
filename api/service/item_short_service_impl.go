package service

import (
	"log/slog"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
)

type ItemShortServiceImpl struct {
	ItemQueryRepository repository.ItemQueryRepository
	AnalyticsService    AnalyticsService
}

func NewItemQueryServiceImpl(
	itemQueryRepository repository.ItemQueryRepository,
	analyticsService AnalyticsService,
) *ItemShortServiceImpl {
	return &ItemShortServiceImpl{
		ItemQueryRepository: itemQueryRepository,
		AnalyticsService:    analyticsService,
	}
}

func (service *ItemShortServiceImpl) FindShortByNiin(niin string) (model.NiinLookup, error) {
	val, err := service.ItemQueryRepository.ShortItemSearchNiin(niin)
	if err != nil {
		return model.NiinLookup{}, err
	} else {
		normalizedNiin := normalizeNiinPointer(val.Niin, niin)
		service.trackItemSearchSuccess(normalizedNiin)
		return val, nil
	}
}

func (service *ItemShortServiceImpl) FindShortByPart(part string) ([]model.NiinLookup, error) {
	results, err := service.ItemQueryRepository.ShortItemSearchPart(part)
	if err != nil {
		return []model.NiinLookup{}, err
	}

	uniqueNiins := make(map[string]struct{})
	for _, result := range results {
		normalizedNiin := normalizeNiinPointer(result.Niin, "")
		if normalizedNiin == "" {
			continue
		}
		if _, exists := uniqueNiins[normalizedNiin]; exists {
			continue
		}
		uniqueNiins[normalizedNiin] = struct{}{}
	}

	for normalizedNiin := range uniqueNiins {
		service.trackItemSearchSuccess(normalizedNiin)
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

func (service *ItemShortServiceImpl) trackItemSearchSuccess(niin string) {
	if service.AnalyticsService == nil || niin == "" {
		return
	}
	if analyticsErr := service.AnalyticsService.IncrementItemSearchSuccess(niin); analyticsErr != nil {
		slog.Warn("Failed to increment analytics for item search", "niin", niin, "error", analyticsErr)
	}
}

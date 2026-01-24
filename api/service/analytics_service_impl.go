package service

import (
	"strings"

	"miltechserver/api/repository"
)

const (
	analyticsEventItemSearchSuccess = "item_search_success"
	analyticsEventPMCSManualDownload = "pmcs_manual_download"
)

type AnalyticsServiceImpl struct {
	AnalyticsRepository repository.AnalyticsRepository
}

func NewAnalyticsServiceImpl(analyticsRepository repository.AnalyticsRepository) *AnalyticsServiceImpl {
	return &AnalyticsServiceImpl{AnalyticsRepository: analyticsRepository}
}

func (service *AnalyticsServiceImpl) IncrementItemSearchSuccess(niin string) error {
	normalized := normalizeAnalyticsKey(niin)
	if normalized == "" {
		return nil
	}
	return service.IncrementCounter(analyticsEventItemSearchSuccess, normalized, normalized)
}

func (service *AnalyticsServiceImpl) IncrementPMCSManualDownload(entityKey string, entityLabel string) error {
	normalizedKey := normalizeAnalyticsKey(sanitizePMCSKey(entityKey))
	normalizedLabel := normalizeAnalyticsKey(entityLabel)
	if normalizedKey == "" {
		return nil
	}
	if normalizedLabel == "" {
		normalizedLabel = normalizedKey
	}
	return service.IncrementCounter(analyticsEventPMCSManualDownload, normalizedKey, normalizedLabel)
}

func (service *AnalyticsServiceImpl) IncrementCounter(eventType string, entityKey string, entityLabel string) error {
	if strings.TrimSpace(eventType) == "" {
		return nil
	}
	return service.AnalyticsRepository.IncrementCounter(eventType, entityKey, entityLabel)
}

func normalizeAnalyticsKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed)
}

func sanitizePMCSKey(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	lowercased := strings.ToLower(value)
	lowercased = strings.ReplaceAll(lowercased, "_", "")
	lowercased = strings.ReplaceAll(lowercased, "checklist", "")
	lowercased = strings.ReplaceAll(lowercased, "packet", "")
	return strings.TrimSpace(lowercased)
}

package analytics

import "strings"

const (
	analyticsEventItemSearchSuccess  = "item_search_success"
	analyticsEventPMCSManualDownload = "pmcs_manual_download"
	analyticsEventPSMagDownload      = "ps_mag_download"
)

type ServiceImpl struct {
	repo Repository
}

func (service *ServiceImpl) IncrementItemSearchSuccess(niin string, nomenclature string) error {
	normalizedKey := normalizeAnalyticsKey(niin)
	normalizedLabel := normalizeAnalyticsKey(nomenclature)
	if normalizedKey == "" {
		return nil
	}
	if normalizedLabel == "" {
		normalizedLabel = normalizedKey
	}
	return service.IncrementCounter(analyticsEventItemSearchSuccess, normalizedKey, normalizedLabel)
}

func (service *ServiceImpl) IncrementPMCSManualDownload(entityKey string, entityLabel string) error {
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

func (service *ServiceImpl) IncrementCounter(eventType string, entityKey string, entityLabel string) error {
	if strings.TrimSpace(eventType) == "" {
		return nil
	}
	return service.repo.IncrementCounter(eventType, entityKey, entityLabel)
}

func normalizeAnalyticsKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed)
}

// formatPSMagLabel derives a human-readable label from a PS Magazine filename.
// Example: "PS_Magazine_Issue_004_September_1951.pdf" → "Issue 004 September 1951"
func formatPSMagLabel(filename string) string {
	label := strings.TrimPrefix(filename, "PS_Magazine_")
	if idx := strings.LastIndex(label, "."); idx != -1 {
		label = label[:idx]
	}
	label = strings.ReplaceAll(label, "_", " ")
	return strings.TrimSpace(label)
}

func (service *ServiceImpl) IncrementPSMagDownload(filename string) error {
	normalizedKey := normalizeAnalyticsKey(filename)
	if normalizedKey == "" {
		return nil
	}
	label := normalizeAnalyticsKey(formatPSMagLabel(filename))
	if label == "" {
		label = normalizedKey
	}
	return service.IncrementCounter(analyticsEventPSMagDownload, normalizedKey, label)
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

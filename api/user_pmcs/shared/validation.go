package shared

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/google/uuid"
)

const (
	shortFieldGraphemeLimit = 200
	longFieldGraphemeLimit  = 4000
	shortFieldByteLimit     = 8 * 1024
	longFieldByteLimit      = 64 * 1024
)

type TreeCounts struct {
	ChecklistModels int
	Sections        int
	SectionModels   int
	Items           int
	Notices         int
	ProcedureSteps  int
}

type PreparedRevision struct {
	Input  RevisionInput
	Hash   [32]byte
	Counts TreeCounts
}

func PrepareDraft(input RevisionInput, config Config) (PreparedRevision, error) {
	return prepareRevision(input, config, false)
}

func PreparePublication(input RevisionInput, config Config) (PreparedRevision, error) {
	return prepareRevision(input, config, true)
}

func prepareRevision(input RevisionInput, config Config, publication bool) (PreparedRevision, error) {
	if err := config.validate(); err != nil {
		return PreparedRevision{}, err
	}
	if err := validateMutationBodySize(input, config.MaxMutationBodyBytes); err != nil {
		return PreparedRevision{}, err
	}
	if err := validateRevisionUTF8(input); err != nil {
		return PreparedRevision{}, err
	}

	normalized, err := cloneAndNormalizeRevision(input)
	if err != nil {
		return PreparedRevision{}, err
	}
	counts, err := validateRevision(normalized, config, publication)
	if err != nil {
		return PreparedRevision{}, err
	}
	hash, err := CanonicalRevisionHash(normalized)
	if err != nil {
		return PreparedRevision{}, err
	}
	return PreparedRevision{
		Input:  normalized,
		Hash:   hash,
		Counts: counts,
	}, nil
}

func validateMutationBodySize(input RevisionInput, maximum int64) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return NewValidationFailed("revision cannot be encoded", nil)
	}
	if int64(len(payload)) > maximum {
		return NewContentTooLarge("revision body exceeds the configured limit", map[string]any{
			"maximum_bytes": maximum,
		})
	}
	return nil
}

func cloneAndNormalizeRevision(input RevisionInput) (RevisionInput, error) {
	cloned := input
	cloned.Models = make([]ModelInput, len(input.Models))
	for index, model := range input.Models {
		normalized, err := NormalizeModel(model.DisplayText)
		if err != nil {
			return RevisionInput{}, err
		}
		cloned.Models[index] = ModelInput{
			DisplayText:    model.DisplayText,
			NormalizedText: normalized,
		}
	}

	cloned.Sections = make([]SectionInput, len(input.Sections))
	for sectionIndex, section := range input.Sections {
		clonedSection := section
		clonedSection.Models = make([]ModelInput, len(section.Models))
		for modelIndex, model := range section.Models {
			normalized, err := NormalizeModel(model.DisplayText)
			if err != nil {
				return RevisionInput{}, err
			}
			clonedSection.Models[modelIndex] = ModelInput{
				DisplayText:    model.DisplayText,
				NormalizedText: normalized,
			}
		}

		clonedSection.Items = make([]ItemInput, len(section.Items))
		for itemIndex, item := range section.Items {
			clonedItem := item
			clonedItem.Notices = make([]NoticeInput, len(item.Notices))
			for noticeIndex, notice := range item.Notices {
				clonedNotice := notice
				if notice.Type != nil {
					noticeType := *notice.Type
					clonedNotice.Type = &noticeType
				}
				clonedItem.Notices[noticeIndex] = clonedNotice
			}
			clonedItem.ProcedureSteps = append(
				[]ProcedureStepInput(nil),
				item.ProcedureSteps...,
			)
			clonedSection.Items[itemIndex] = clonedItem
		}
		cloned.Sections[sectionIndex] = clonedSection
	}
	return cloned, nil
}

func validateRevision(input RevisionInput, config Config, publication bool) (TreeCounts, error) {
	if input.ID == uuid.Nil {
		return TreeCounts{}, validationError("revision.id", "must be a non-zero UUID")
	}
	if err := validateShortField("revision.name", input.Name); err != nil {
		return TreeCounts{}, err
	}
	if err := validateLongField("revision.description", input.Description); err != nil {
		return TreeCounts{}, err
	}
	if publication && isBlank(input.Name) {
		return TreeCounts{}, validationError("revision.name", "must not be blank")
	}
	if publication && len(input.Models) == 0 {
		return TreeCounts{}, validationError("revision.models", "must contain at least one model")
	}
	if publication && len(input.Sections) == 0 {
		return TreeCounts{}, validationError("revision.sections", "must contain at least one section")
	}

	counts := TreeCounts{
		ChecklistModels: len(input.Models),
		Sections:        len(input.Sections),
	}
	if err := enforceCeiling("revision.models", counts.ChecklistModels, config.MaxChecklistModels); err != nil {
		return TreeCounts{}, err
	}
	if err := enforceCeiling("revision.sections", counts.Sections, config.MaxSections); err != nil {
		return TreeCounts{}, err
	}
	if err := validateModels("revision.models", input.Models, publication); err != nil {
		return TreeCounts{}, err
	}
	if err := validatePositions("revision.sections", len(input.Sections), func(index int) int32 {
		return input.Sections[index].Position
	}); err != nil {
		return TreeCounts{}, err
	}

	seenUUIDs := map[uuid.UUID]string{input.ID: "revision.id"}
	for sectionIndex, section := range input.Sections {
		sectionPath := fmt.Sprintf("revision.sections[%d]", sectionIndex)
		if err := validateAndTrackUUID(seenUUIDs, section.ID, sectionPath+".id"); err != nil {
			return TreeCounts{}, err
		}
		if err := validateShortField(sectionPath+".title", section.Title); err != nil {
			return TreeCounts{}, err
		}
		if publication && isBlank(section.Title) {
			return TreeCounts{}, validationError(sectionPath+".title", "must not be blank")
		}
		if publication && len(section.Items) == 0 {
			return TreeCounts{}, validationError(sectionPath+".items", "must contain at least one item")
		}
		if err := enforceCeiling(sectionPath+".models", len(section.Models), config.MaxSectionModels); err != nil {
			return TreeCounts{}, err
		}
		if err := enforceCeiling(sectionPath+".items", len(section.Items), config.MaxItemsPerSection); err != nil {
			return TreeCounts{}, err
		}
		if err := validateModels(sectionPath+".models", section.Models, publication); err != nil {
			return TreeCounts{}, err
		}
		if err := validatePositions(sectionPath+".items", len(section.Items), func(index int) int32 {
			return section.Items[index].Position
		}); err != nil {
			return TreeCounts{}, err
		}
		counts.SectionModels += len(section.Models)
		counts.Items += len(section.Items)

		for itemIndex, item := range section.Items {
			itemPath := fmt.Sprintf("%s.items[%d]", sectionPath, itemIndex)
			if err := validateAndTrackUUID(seenUUIDs, item.ID, itemPath+".id"); err != nil {
				return TreeCounts{}, err
			}
			if err := validateShortField(itemPath+".interval", item.Interval); err != nil {
				return TreeCounts{}, err
			}
			if err := validateLongField(itemPath+".item_to_be_checked_or_serviced", item.ItemToBeCheckedOrServiced); err != nil {
				return TreeCounts{}, err
			}
			if err := validateShortField(itemPath+".performed_by", item.PerformedBy); err != nil {
				return TreeCounts{}, err
			}
			if publication && isBlank(item.Interval) {
				return TreeCounts{}, validationError(itemPath+".interval", "must not be blank")
			}
			if publication && isBlank(item.ItemToBeCheckedOrServiced) {
				return TreeCounts{}, validationError(itemPath+".item_to_be_checked_or_serviced", "must not be blank")
			}
			if publication && len(item.ProcedureSteps) == 0 {
				return TreeCounts{}, validationError(itemPath+".procedure_steps", "must contain at least one procedure step")
			}
			if err := enforceCeiling(itemPath+".notices", len(item.Notices), config.MaxNoticesPerItem); err != nil {
				return TreeCounts{}, err
			}
			if err := enforceCeiling(itemPath+".procedure_steps", len(item.ProcedureSteps), config.MaxStepsPerItem); err != nil {
				return TreeCounts{}, err
			}
			if err := validatePositions(itemPath+".notices", len(item.Notices), func(index int) int32 {
				return item.Notices[index].Position
			}); err != nil {
				return TreeCounts{}, err
			}
			if err := validatePositions(itemPath+".procedure_steps", len(item.ProcedureSteps), func(index int) int32 {
				return item.ProcedureSteps[index].Position
			}); err != nil {
				return TreeCounts{}, err
			}
			counts.Notices += len(item.Notices)
			counts.ProcedureSteps += len(item.ProcedureSteps)

			for noticeIndex, notice := range item.Notices {
				noticePath := fmt.Sprintf("%s.notices[%d]", itemPath, noticeIndex)
				if err := validateAndTrackUUID(seenUUIDs, notice.ID, noticePath+".id"); err != nil {
					return TreeCounts{}, err
				}
				if err := validateLongField(noticePath+".notice_text", notice.NoticeText); err != nil {
					return TreeCounts{}, err
				}
				if notice.Type != nil && !isSupportedNoticeType(*notice.Type) {
					return TreeCounts{}, validationError(noticePath+".type", "must be warning, caution, or note")
				}
				if publication && notice.Type == nil {
					return TreeCounts{}, validationError(noticePath+".type", "must be present")
				}
				if publication && isBlank(notice.NoticeText) {
					return TreeCounts{}, validationError(noticePath+".notice_text", "must not be blank")
				}
			}

			for stepIndex, step := range item.ProcedureSteps {
				stepPath := fmt.Sprintf("%s.procedure_steps[%d]", itemPath, stepIndex)
				if err := validateAndTrackUUID(seenUUIDs, step.ID, stepPath+".id"); err != nil {
					return TreeCounts{}, err
				}
				if err := validateLongField(stepPath+".step_text", step.StepText); err != nil {
					return TreeCounts{}, err
				}
				if err := validateLongField(stepPath+".fault_found_if", step.FaultFoundIf); err != nil {
					return TreeCounts{}, err
				}
				if publication && isBlank(step.StepText) {
					return TreeCounts{}, validationError(stepPath+".step_text", "must not be blank")
				}
			}
		}
	}

	totalCeilings := []struct {
		path    string
		actual  int
		maximum int
	}{
		{"revision.section_models", counts.SectionModels, config.MaxSectionModelsTotal},
		{"revision.items", counts.Items, config.MaxItemsTotal},
		{"revision.notices", counts.Notices, config.MaxNoticesTotal},
		{"revision.procedure_steps", counts.ProcedureSteps, config.MaxStepsTotal},
	}
	for _, ceiling := range totalCeilings {
		if err := enforceCeiling(ceiling.path, ceiling.actual, ceiling.maximum); err != nil {
			return TreeCounts{}, err
		}
	}
	return counts, nil
}

func validateRevisionUTF8(input RevisionInput) error {
	fields := []struct {
		path  string
		value string
	}{
		{"revision.name", input.Name},
		{"revision.description", input.Description},
	}
	for modelIndex, model := range input.Models {
		fields = append(fields, struct {
			path  string
			value string
		}{fmt.Sprintf("revision.models[%d].display_text", modelIndex), model.DisplayText})
	}
	for sectionIndex, section := range input.Sections {
		sectionPath := fmt.Sprintf("revision.sections[%d]", sectionIndex)
		fields = append(fields, struct {
			path  string
			value string
		}{sectionPath + ".title", section.Title})
		for modelIndex, model := range section.Models {
			fields = append(fields, struct {
				path  string
				value string
			}{fmt.Sprintf("%s.models[%d].display_text", sectionPath, modelIndex), model.DisplayText})
		}
		for itemIndex, item := range section.Items {
			itemPath := fmt.Sprintf("%s.items[%d]", sectionPath, itemIndex)
			fields = append(fields,
				struct {
					path  string
					value string
				}{itemPath + ".interval", item.Interval},
				struct {
					path  string
					value string
				}{itemPath + ".item_to_be_checked_or_serviced", item.ItemToBeCheckedOrServiced},
				struct {
					path  string
					value string
				}{itemPath + ".performed_by", item.PerformedBy},
			)
			for noticeIndex, notice := range item.Notices {
				noticePath := fmt.Sprintf("%s.notices[%d]", itemPath, noticeIndex)
				if notice.Type != nil {
					fields = append(fields, struct {
						path  string
						value string
					}{noticePath + ".type", *notice.Type})
				}
				fields = append(fields, struct {
					path  string
					value string
				}{noticePath + ".notice_text", notice.NoticeText})
			}
			for stepIndex, step := range item.ProcedureSteps {
				stepPath := fmt.Sprintf("%s.procedure_steps[%d]", itemPath, stepIndex)
				fields = append(fields,
					struct {
						path  string
						value string
					}{stepPath + ".step_text", step.StepText},
					struct {
						path  string
						value string
					}{stepPath + ".fault_found_if", step.FaultFoundIf},
				)
			}
		}
	}
	for _, field := range fields {
		if !utf8.ValidString(field.value) {
			return validationError(field.path, "must contain valid UTF-8")
		}
	}
	return nil
}

func validateModels(path string, models []ModelInput, requireNonblank bool) error {
	seen := make(map[string]struct{}, len(models))
	for index, model := range models {
		modelPath := fmt.Sprintf("%s[%d]", path, index)
		if err := validateShortField(modelPath+".display_text", model.DisplayText); err != nil {
			return err
		}
		if err := validateShortField(modelPath+".normalized_text", model.NormalizedText); err != nil {
			return err
		}
		if requireNonblank && (isBlank(model.DisplayText) || model.NormalizedText == "") {
			return validationError(modelPath, "model values must not be blank")
		}
		if _, duplicate := seen[model.NormalizedText]; duplicate {
			return validationError(path, "normalized model values must be unique")
		}
		seen[model.NormalizedText] = struct{}{}
	}
	return nil
}

func validateShortField(path, value string) error {
	return validateTextField(path, value, shortFieldGraphemeLimit, shortFieldByteLimit)
}

func validateLongField(path, value string) error {
	return validateTextField(path, value, longFieldGraphemeLimit, longFieldByteLimit)
}

func validateTextField(path, value string, graphemeLimit, byteLimit int) error {
	if len(value) > byteLimit {
		return validationError(path, fmt.Sprintf("must not exceed %d bytes", byteLimit))
	}
	if countGraphemes(value) > graphemeLimit {
		return validationError(path, fmt.Sprintf("must not exceed %d graphemes", graphemeLimit))
	}
	return nil
}

func countGraphemes(value string) int {
	count := 0
	iterator := graphemes.FromString(value)
	for iterator.Next() {
		count++
	}
	return count
}

func validateAndTrackUUID(seen map[uuid.UUID]string, id uuid.UUID, path string) error {
	if id == uuid.Nil {
		return validationError(path, "must be a non-zero UUID")
	}
	if existingPath, duplicate := seen[id]; duplicate {
		return validationError(path, "UUID is already used by "+existingPath)
	}
	seen[id] = path
	return nil
}

func validatePositions(path string, count int, position func(int) int32) error {
	seen := make([]bool, count+1)
	for index := 0; index < count; index++ {
		value := position(index)
		if value < 1 || int(value) > count || seen[value] {
			return validationError(path, "positions must be unique and contiguous from one")
		}
		seen[value] = true
	}
	return nil
}

func enforceCeiling(path string, actual, maximum int) error {
	if actual > maximum {
		return NewContentTooLarge("revision tree exceeds the configured node limit", map[string]any{
			"field":   path,
			"actual":  actual,
			"maximum": maximum,
		})
	}
	return nil
}

func validationError(path, reason string) *APIError {
	return NewValidationFailed("revision validation failed", map[string]any{
		"field":  path,
		"reason": reason,
	})
}

func isSupportedNoticeType(value string) bool {
	switch value {
	case "warning", "caution", "note":
		return true
	default:
		return false
	}
}

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

package shared_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/shared"
)

const (
	shortGraphemeLimit = 200
	longGraphemeLimit  = 4000
	shortByteLimit     = 8 * 1024
	longByteLimit      = 64 * 1024
)

var (
	revisionID = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	sectionID  = uuid.MustParse("20000000-0000-0000-0000-000000000001")
	itemID     = uuid.MustParse("30000000-0000-0000-0000-000000000001")
	noticeID   = uuid.MustParse("40000000-0000-0000-0000-000000000001")
	stepID     = uuid.MustParse("50000000-0000-0000-0000-000000000001")
)

type unicodeFixture struct {
	NormalizationCases []struct {
		Name       string `json:"name"`
		Input      string `json:"input"`
		Normalized string `json:"normalized"`
	} `json:"normalization_cases"`
	GraphemeCases []struct {
		Name  string `json:"name"`
		Input string `json:"input"`
		Count int    `json:"count"`
	} `json:"grapheme_cases"`
}

func TestNormalizeModelUnicodeV16Fixtures(t *testing.T) {
	fixture := loadUnicodeFixture(t)
	for _, testCase := range fixture.NormalizationCases {
		t.Run(testCase.Name, func(t *testing.T) {
			got, err := shared.NormalizeModel(testCase.Input)
			if err != nil {
				t.Fatalf("NormalizeModel() error = %v", err)
			}
			if got != testCase.Normalized {
				t.Fatalf("NormalizeModel() = %q, want %q", got, testCase.Normalized)
			}
		})
	}
}

func TestNormalizeModelRejectsInvalidUTF8(t *testing.T) {
	_, err := shared.NormalizeModel(string([]byte{0xff}))
	requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
}

func TestPublicationCountsUnicodeV16GraphemeFixtures(t *testing.T) {
	fixture := loadUnicodeFixture(t)
	for _, testCase := range fixture.GraphemeCases {
		t.Run(testCase.Name, func(t *testing.T) {
			input := validPublication()
			input.Name = testCase.Input + strings.Repeat("a", shortGraphemeLimit-testCase.Count)
			_, err := shared.PreparePublication(input, shared.DefaultConfig())
			if err != nil {
				t.Fatalf("PreparePublication() at %d graphemes error = %v", shortGraphemeLimit, err)
			}

			input.Name += "x"
			_, err = shared.PreparePublication(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
}

func TestPublicationShortFieldGraphemeBoundaries(t *testing.T) {
	fields := []fieldMutation{
		{name: "checklist name", mutate: func(input *shared.RevisionInput, value string) { input.Name = value }},
		{name: "checklist model display", mutate: func(input *shared.RevisionInput, value string) { input.Models[0].DisplayText = value }},
		{name: "section title", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Title = value }},
		{name: "section model display", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Models[0].DisplayText = value }},
		{name: "item interval", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Items[0].Interval = value }},
		{name: "performed by", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Items[0].PerformedBy = value }},
	}
	assertFieldGraphemeBoundaries(t, fields, shortGraphemeLimit)
}

func TestPublicationLongFieldGraphemeBoundaries(t *testing.T) {
	fields := []fieldMutation{
		{name: "checklist description", mutate: func(input *shared.RevisionInput, value string) { input.Description = value }},
		{name: "item text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ItemToBeCheckedOrServiced = value
		}},
		{name: "notice text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].Notices[0].NoticeText = value
		}},
		{name: "procedure step text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ProcedureSteps[0].StepText = value
		}},
		{name: "fault found if", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ProcedureSteps[0].FaultFoundIf = value
		}},
	}
	assertFieldGraphemeBoundaries(t, fields, longGraphemeLimit)
}

func TestPublicationShortFieldByteBoundaries(t *testing.T) {
	fields := []fieldMutation{
		{name: "checklist name", mutate: func(input *shared.RevisionInput, value string) { input.Name = value }},
		{name: "checklist model display", mutate: func(input *shared.RevisionInput, value string) { input.Models[0].DisplayText = value }},
		{name: "section title", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Title = value }},
		{name: "section model display", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Models[0].DisplayText = value }},
		{name: "item interval", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Items[0].Interval = value }},
		{name: "performed by", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Items[0].PerformedBy = value }},
	}
	assertFieldByteBoundaries(t, fields, shortByteLimit)
}

func TestPublicationLongFieldByteBoundaries(t *testing.T) {
	fields := []fieldMutation{
		{name: "checklist description", mutate: func(input *shared.RevisionInput, value string) { input.Description = value }},
		{name: "item text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ItemToBeCheckedOrServiced = value
		}},
		{name: "notice text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].Notices[0].NoticeText = value
		}},
		{name: "procedure step text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ProcedureSteps[0].StepText = value
		}},
		{name: "fault found if", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ProcedureSteps[0].FaultFoundIf = value
		}},
	}
	assertFieldByteBoundaries(t, fields, longByteLimit)
}

func TestPublicationRejectsNormalizedModelByteOverflow(t *testing.T) {
	// U+023A lowercases to U+2C65, growing from two UTF-8 bytes to three.
	// Combining marks keep the whole value within the 200-grapheme ceiling.
	value := strings.Repeat("\u023a", shortGraphemeLimit) +
		strings.Repeat("\u0301", (shortByteLimit-(shortGraphemeLimit*2))/2)
	if len(value) != shortByteLimit {
		t.Fatalf("test fixture bytes = %d, want %d", len(value), shortByteLimit)
	}

	input := validPublication()
	input.Models[0].DisplayText = value
	_, err := shared.PreparePublication(input, shared.DefaultConfig())
	requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
}

func TestDraftRejectsInvalidUTF8InEveryAuthoredField(t *testing.T) {
	invalid := string([]byte{0xff})
	invalidNoticeType := invalid
	fields := []fieldMutation{
		{name: "checklist name", mutate: func(input *shared.RevisionInput, value string) { input.Name = value }},
		{name: "checklist description", mutate: func(input *shared.RevisionInput, value string) { input.Description = value }},
		{name: "checklist model display", mutate: func(input *shared.RevisionInput, value string) { input.Models[0].DisplayText = value }},
		{name: "section title", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Title = value }},
		{name: "section model display", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Models[0].DisplayText = value }},
		{name: "item interval", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Items[0].Interval = value }},
		{name: "item text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ItemToBeCheckedOrServiced = value
		}},
		{name: "performed by", mutate: func(input *shared.RevisionInput, value string) { input.Sections[0].Items[0].PerformedBy = value }},
		{name: "notice text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].Notices[0].NoticeText = value
		}},
		{name: "procedure step text", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ProcedureSteps[0].StepText = value
		}},
		{name: "fault found if", mutate: func(input *shared.RevisionInput, value string) {
			input.Sections[0].Items[0].ProcedureSteps[0].FaultFoundIf = value
		}},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			input := validPublication()
			field.mutate(&input, invalid)
			_, err := shared.PrepareDraft(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
	t.Run("notice type", func(t *testing.T) {
		input := validPublication()
		input.Sections[0].Items[0].Notices[0].Type = &invalidNoticeType
		_, err := shared.PrepareDraft(input, shared.DefaultConfig())
		requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
	})
}

func TestDraftPermitsIncompleteAndBlankAuthoredStructure(t *testing.T) {
	input := shared.RevisionInput{ID: revisionID}
	prepared, err := shared.PrepareDraft(input, shared.DefaultConfig())
	if err != nil {
		t.Fatalf("PrepareDraft() minimal draft error = %v", err)
	}
	if prepared.Input.ID != revisionID {
		t.Fatalf("PrepareDraft() ID = %s, want %s", prepared.Input.ID, revisionID)
	}

	input = validPublication()
	input.Name = " "
	input.Description = ""
	input.Models[0].DisplayText = "\t"
	input.Sections[0].Title = "\t"
	input.Sections[0].Models[0].DisplayText = "\u2003"
	input.Sections[0].Items[0].Interval = ""
	input.Sections[0].Items[0].ItemToBeCheckedOrServiced = "\u2003"
	input.Sections[0].Items[0].Notices[0].Type = nil
	input.Sections[0].Items[0].Notices[0].NoticeText = ""
	input.Sections[0].Items[0].ProcedureSteps[0].StepText = ""
	if _, err := shared.PrepareDraft(input, shared.DefaultConfig()); err != nil {
		t.Fatalf("PrepareDraft() blank authored fields error = %v", err)
	}
}

func TestDraftRejectsPresentUnsupportedNoticeType(t *testing.T) {
	input := validPublication()
	unsupported := "danger"
	input.Sections[0].Items[0].Notices[0].Type = &unsupported
	_, err := shared.PrepareDraft(input, shared.DefaultConfig())
	requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
}

func TestPublicationCompletenessRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*shared.RevisionInput)
	}{
		{name: "blank checklist name", mutate: func(input *shared.RevisionInput) { input.Name = "\u2003" }},
		{name: "no checklist models", mutate: func(input *shared.RevisionInput) { input.Models = nil }},
		{name: "blank checklist model", mutate: func(input *shared.RevisionInput) { input.Models[0].DisplayText = " " }},
		{name: "no sections", mutate: func(input *shared.RevisionInput) { input.Sections = nil }},
		{name: "blank section title", mutate: func(input *shared.RevisionInput) { input.Sections[0].Title = "\t" }},
		{name: "blank section model", mutate: func(input *shared.RevisionInput) { input.Sections[0].Models[0].DisplayText = "\n" }},
		{name: "no items", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items = nil }},
		{name: "blank item interval", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Interval = " " }},
		{name: "blank item text", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ItemToBeCheckedOrServiced = "\u00a0" }},
		{name: "nil notice type", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Notices[0].Type = nil }},
		{name: "unsupported notice type", mutate: func(input *shared.RevisionInput) {
			value := "danger"
			input.Sections[0].Items[0].Notices[0].Type = &value
		}},
		{name: "blank notice text", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Notices[0].NoticeText = " " }},
		{name: "no procedure steps", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps = nil }},
		{name: "blank procedure step text", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps[0].StepText = "\n" }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := validPublication()
			testCase.mutate(&input)
			_, err := shared.PreparePublication(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
}

func TestPublicationRejectsDuplicateNormalizedModels(t *testing.T) {
	for _, scope := range []string{"checklist", "section"} {
		t.Run(scope, func(t *testing.T) {
			input := validPublication()
			models := []shared.ModelInput{
				{DisplayText: " M1152A1 "},
				{DisplayText: "m1152a1"},
			}
			if scope == "checklist" {
				input.Models = models
			} else {
				input.Sections[0].Models = models
			}
			_, err := shared.PreparePublication(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
}

func TestDraftRejectsNilAndDuplicateUUIDsAcrossTree(t *testing.T) {
	nilCases := []struct {
		name   string
		mutate func(*shared.RevisionInput)
	}{
		{name: "revision", mutate: func(input *shared.RevisionInput) { input.ID = uuid.Nil }},
		{name: "section", mutate: func(input *shared.RevisionInput) { input.Sections[0].ID = uuid.Nil }},
		{name: "item", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ID = uuid.Nil }},
		{name: "notice", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Notices[0].ID = uuid.Nil }},
		{name: "procedure step", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps[0].ID = uuid.Nil }},
	}
	for _, testCase := range nilCases {
		t.Run("nil "+testCase.name, func(t *testing.T) {
			input := validPublication()
			testCase.mutate(&input)
			_, err := shared.PrepareDraft(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}

	t.Run("duplicate across node types", func(t *testing.T) {
		input := validPublication()
		input.Sections[0].Items[0].ID = input.Sections[0].ID
		_, err := shared.PrepareDraft(input, shared.DefaultConfig())
		requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
	})
	t.Run("duplicate same node type", func(t *testing.T) {
		input := validPublication()
		second := input.Sections[0]
		second.Position = 2
		input.Sections = append(input.Sections, second)
		_, err := shared.PrepareDraft(input, shared.DefaultConfig())
		requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
	})
}

func TestDraftRequiresContiguousOneBasedSiblingPositions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*shared.RevisionInput)
	}{
		{name: "sections", mutate: func(input *shared.RevisionInput) { input.Sections[0].Position = 2 }},
		{name: "items", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Position = 2 }},
		{name: "notices", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Notices[0].Position = 2 }},
		{name: "procedure steps", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps[0].Position = 2 }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := validPublication()
			testCase.mutate(&input)
			_, err := shared.PrepareDraft(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
}

func TestDraftAcceptsOutOfOrderContiguousPositions(t *testing.T) {
	input := validPublication()
	second := cloneSection(2)
	input.Sections = []shared.SectionInput{second, input.Sections[0]}
	if _, err := shared.PrepareDraft(input, shared.DefaultConfig()); err != nil {
		t.Fatalf("PrepareDraft() out-of-order contiguous positions error = %v", err)
	}
}

func TestDraftPerParentNodeCeilings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*shared.RevisionInput)
		config func(*shared.Config)
	}{
		{
			name: "checklist models",
			mutate: func(input *shared.RevisionInput) {
				input.Models = append(input.Models, shared.ModelInput{DisplayText: "M1165A1"})
			},
			config: func(config *shared.Config) { config.MaxChecklistModels = 1 },
		},
		{
			name:   "sections",
			mutate: func(input *shared.RevisionInput) { input.Sections = append(input.Sections, cloneSection(2)) },
			config: func(config *shared.Config) { config.MaxSections = 1 },
		},
		{
			name: "section models",
			mutate: func(input *shared.RevisionInput) {
				input.Sections[0].Models = append(input.Sections[0].Models, shared.ModelInput{DisplayText: "M1165A1"})
			},
			config: func(config *shared.Config) { config.MaxSectionModels = 1 },
		},
		{
			name: "items per section",
			mutate: func(input *shared.RevisionInput) {
				input.Sections[0].Items = append(input.Sections[0].Items, cloneItem(2, 2))
			},
			config: func(config *shared.Config) { config.MaxItemsPerSection = 1 },
		},
		{
			name: "notices per item",
			mutate: func(input *shared.RevisionInput) {
				input.Sections[0].Items[0].Notices = append(input.Sections[0].Items[0].Notices, cloneNotice(2, 2))
			},
			config: func(config *shared.Config) { config.MaxNoticesPerItem = 1 },
		},
		{
			name: "steps per item",
			mutate: func(input *shared.RevisionInput) {
				input.Sections[0].Items[0].ProcedureSteps = append(input.Sections[0].Items[0].ProcedureSteps, cloneStep(2, 2))
			},
			config: func(config *shared.Config) { config.MaxStepsPerItem = 1 },
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config := shared.DefaultConfig()
			testCase.config(&config)
			if _, err := shared.PrepareDraft(validPublication(), config); err != nil {
				t.Fatalf("PrepareDraft() at exact ceiling error = %v", err)
			}
			input := validPublication()
			testCase.mutate(&input)
			_, err := shared.PrepareDraft(input, config)
			requireAPIError(t, err, http.StatusRequestEntityTooLarge, "content_too_large")
		})
	}
}

func TestDraftTotalNodeCeilings(t *testing.T) {
	tests := []struct {
		name   string
		config func(*shared.Config)
	}{
		{name: "section models total", config: func(config *shared.Config) { config.MaxSectionModelsTotal = 1 }},
		{name: "items total", config: func(config *shared.Config) { config.MaxItemsTotal = 1 }},
		{name: "notices total", config: func(config *shared.Config) { config.MaxNoticesTotal = 1 }},
		{name: "steps total", config: func(config *shared.Config) { config.MaxStepsTotal = 1 }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config := shared.DefaultConfig()
			testCase.config(&config)
			if _, err := shared.PrepareDraft(validPublication(), config); err != nil {
				t.Fatalf("PrepareDraft() at exact total ceiling error = %v", err)
			}
			input := validPublication()
			input.Sections = append(input.Sections, cloneSection(2))
			_, err := shared.PrepareDraft(input, config)
			requireAPIError(t, err, http.StatusRequestEntityTooLarge, "content_too_large")
		})
	}
}

func TestDraftMutationBodyByteCeiling(t *testing.T) {
	input := validPublication()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	config := shared.DefaultConfig()
	config.MaxMutationBodyBytes = int64(len(payload))
	if _, err := shared.PrepareDraft(input, config); err != nil {
		t.Fatalf("PrepareDraft() at exact body ceiling error = %v", err)
	}

	config.MaxMutationBodyBytes--
	_, err = shared.PrepareDraft(input, config)
	requireAPIError(t, err, http.StatusRequestEntityTooLarge, "content_too_large")
}

func TestDraftPreparationFillsNormalizedModelsCountsAndHashWithoutMutatingCaller(t *testing.T) {
	input := validPublication()
	input.Models[0].DisplayText = " M1152\u2003A1 "
	input.Sections[0].Models[0].DisplayText = "\tHEMTT\n"

	prepared, err := shared.PrepareDraft(input, shared.DefaultConfig())
	if err != nil {
		t.Fatalf("PrepareDraft() error = %v", err)
	}
	if prepared.Input.Models[0].NormalizedText != "m1152 a1" {
		t.Fatalf("checklist normalized model = %q", prepared.Input.Models[0].NormalizedText)
	}
	if prepared.Input.Sections[0].Models[0].NormalizedText != "hemtt" {
		t.Fatalf("section normalized model = %q", prepared.Input.Sections[0].Models[0].NormalizedText)
	}
	if input.Models[0].NormalizedText != "" || input.Sections[0].Models[0].NormalizedText != "" {
		t.Fatal("PrepareDraft() mutated caller-owned model values")
	}
	wantCounts := shared.TreeCounts{
		ChecklistModels: 1,
		Sections:        1,
		SectionModels:   1,
		Items:           1,
		Notices:         1,
		ProcedureSteps:  1,
	}
	if prepared.Counts != wantCounts {
		t.Fatalf("PrepareDraft() counts = %+v, want %+v", prepared.Counts, wantCounts)
	}
	hash, err := shared.CanonicalRevisionHash(prepared.Input)
	if err != nil {
		t.Fatalf("CanonicalRevisionHash() error = %v", err)
	}
	if prepared.Hash != hash {
		t.Fatalf("PrepareDraft() hash = %x, want %x", prepared.Hash, hash)
	}
}

type fieldMutation struct {
	name   string
	mutate func(*shared.RevisionInput, string)
}

func assertFieldGraphemeBoundaries(t *testing.T, fields []fieldMutation, limit int) {
	t.Helper()
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			input := validPublication()
			field.mutate(&input, strings.Repeat("x", limit))
			if _, err := shared.PreparePublication(input, shared.DefaultConfig()); err != nil {
				t.Fatalf("PreparePublication() at %d graphemes error = %v", limit, err)
			}

			field.mutate(&input, strings.Repeat("x", limit+1))
			_, err := shared.PreparePublication(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
}

func assertFieldByteBoundaries(t *testing.T, fields []fieldMutation, limit int) {
	t.Helper()
	atLimit := "\u00e9" + strings.Repeat("\u0301", (limit-2)/2)
	overLimit := "a" + strings.Repeat("\u0301", limit/2)
	if len(atLimit) != limit || len(overLimit) != limit+1 {
		t.Fatalf("invalid byte boundary fixtures: got %d and %d", len(atLimit), len(overLimit))
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			input := validPublication()
			field.mutate(&input, atLimit)
			if _, err := shared.PreparePublication(input, shared.DefaultConfig()); err != nil {
				t.Fatalf("PreparePublication() at %d bytes error = %v", limit, err)
			}

			field.mutate(&input, overLimit)
			_, err := shared.PreparePublication(input, shared.DefaultConfig())
			requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
		})
	}
}

func loadUnicodeFixture(t *testing.T) unicodeFixture {
	t.Helper()
	content, err := os.ReadFile("testdata/unicode_v16.json")
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var fixture unicodeFixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return fixture
}

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want status %d code %q", status, code)
	}
	var apiError *shared.APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error type = %T, want *shared.APIError: %v", err, err)
	}
	if apiError.Status != status || apiError.Code != code {
		t.Fatalf("API error = status %d code %q, want status %d code %q",
			apiError.Status, apiError.Code, status, code)
	}
}

func validPublication() shared.RevisionInput {
	noticeType := "warning"
	return shared.RevisionInput{
		ID:          revisionID,
		Name:        "M1152A1 PMCS",
		Description: "Preventive maintenance checks and services",
		Models: []shared.ModelInput{
			{DisplayText: "M1152A1"},
		},
		Sections: []shared.SectionInput{
			{
				ID:       sectionID,
				Position: 1,
				Title:    "Before operation",
				Models: []shared.ModelInput{
					{DisplayText: "M1152A1"},
				},
				Items: []shared.ItemInput{
					{
						ID:                        itemID,
						Position:                  1,
						Interval:                  "Before",
						ItemToBeCheckedOrServiced: "Check engine oil level",
						PerformedBy:               "Operator",
						Notices: []shared.NoticeInput{
							{
								ID:         noticeID,
								Position:   1,
								Type:       &noticeType,
								NoticeText: "Wear gloves",
							},
						},
						ProcedureSteps: []shared.ProcedureStepInput{
							{
								ID:           stepID,
								Position:     1,
								StepText:     "Inspect the dipstick",
								FaultFoundIf: "Oil is below the safe range",
							},
						},
					},
				},
			},
		},
	}
}

func cloneSection(position int32) shared.SectionInput {
	section := validPublication().Sections[0]
	section.ID = uuid.MustParse("20000000-0000-0000-0000-000000000002")
	section.Position = position
	section.Items = []shared.ItemInput{cloneItem(1, 2)}
	return section
}

func cloneItem(position, identity int32) shared.ItemInput {
	item := validPublication().Sections[0].Items[0]
	item.ID = uuid.MustParse("30000000-0000-0000-0000-00000000000" + string(rune('0'+identity)))
	item.Position = position
	item.Notices = []shared.NoticeInput{cloneNotice(1, identity)}
	item.ProcedureSteps = []shared.ProcedureStepInput{cloneStep(1, identity)}
	return item
}

func cloneNotice(position, identity int32) shared.NoticeInput {
	notice := validPublication().Sections[0].Items[0].Notices[0]
	notice.ID = uuid.MustParse("40000000-0000-0000-0000-00000000000" + string(rune('0'+identity)))
	notice.Position = position
	return notice
}

func cloneStep(position, identity int32) shared.ProcedureStepInput {
	step := validPublication().Sections[0].Items[0].ProcedureSteps[0]
	step.ID = uuid.MustParse("50000000-0000-0000-0000-00000000000" + string(rune('0'+identity)))
	step.Position = position
	return step
}

package shared_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/shared"
)

func TestCanonicalHashIgnoresModelSetAndPositionedSliceOrder(t *testing.T) {
	first := validPublication()
	first.Models = append(first.Models, shared.ModelInput{DisplayText: "M1165A1"})
	first.Sections[0].Models = append(first.Sections[0].Models, shared.ModelInput{DisplayText: "M1165A1"})
	first.Sections = append(first.Sections, cloneSection(2))
	first.Sections[0].Items = append(first.Sections[0].Items, cloneItem(2, 2))
	first.Sections[0].Items[0].Notices = append(
		first.Sections[0].Items[0].Notices,
		cloneNotice(2, 2),
	)
	first.Sections[0].Items[0].ProcedureSteps = append(
		first.Sections[0].Items[0].ProcedureSteps,
		cloneStep(2, 2),
	)

	second := first
	second.Models = reversedModels(first.Models)
	second.Sections = []shared.SectionInput{first.Sections[1], first.Sections[0]}
	second.Sections[1].Models = reversedModels(first.Sections[0].Models)
	second.Sections[1].Items = []shared.ItemInput{
		first.Sections[0].Items[1],
		first.Sections[0].Items[0],
	}
	second.Sections[1].Items[1].Notices = []shared.NoticeInput{
		first.Sections[0].Items[0].Notices[1],
		first.Sections[0].Items[0].Notices[0],
	}
	second.Sections[1].Items[1].ProcedureSteps = []shared.ProcedureStepInput{
		first.Sections[0].Items[0].ProcedureSteps[1],
		first.Sections[0].Items[0].ProcedureSteps[0],
	}

	firstHash := mustCanonicalHash(t, first)
	secondHash := mustCanonicalHash(t, second)
	if firstHash != secondHash {
		t.Fatalf("canonical hashes differ by set/slice order: %x != %x", firstHash, secondHash)
	}
}

func TestCanonicalHashChangesForEveryAuthoredString(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*shared.RevisionInput)
	}{
		{name: "checklist name", mutate: func(input *shared.RevisionInput) { input.Name += "!" }},
		{name: "description", mutate: func(input *shared.RevisionInput) { input.Description += "!" }},
		{name: "checklist model display", mutate: func(input *shared.RevisionInput) { input.Models[0].DisplayText += "!" }},
		{name: "section title", mutate: func(input *shared.RevisionInput) { input.Sections[0].Title += "!" }},
		{name: "section model display", mutate: func(input *shared.RevisionInput) { input.Sections[0].Models[0].DisplayText += "!" }},
		{name: "item interval", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Interval += "!" }},
		{name: "item text", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ItemToBeCheckedOrServiced += "!" }},
		{name: "performed by", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].PerformedBy += "!" }},
		{name: "notice type", mutate: func(input *shared.RevisionInput) {
			value := "note"
			input.Sections[0].Items[0].Notices[0].Type = &value
		}},
		{name: "notice text", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Notices[0].NoticeText += "!" }},
		{name: "procedure step text", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps[0].StepText += "!" }},
		{name: "fault found if", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps[0].FaultFoundIf += "!" }},
	}
	baseline := mustCanonicalHash(t, validPublication())
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := validPublication()
			testCase.mutate(&input)
			if got := mustCanonicalHash(t, input); got == baseline {
				t.Fatalf("CanonicalRevisionHash() unchanged after changing %s", testCase.name)
			}
		})
	}
}

func TestCanonicalHashChangesForEveryClientUUID(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*shared.RevisionInput)
	}{
		{name: "revision", mutate: func(input *shared.RevisionInput) { input.ID = uuid.MustParse("10000000-0000-0000-0000-000000000099") }},
		{name: "section", mutate: func(input *shared.RevisionInput) {
			input.Sections[0].ID = uuid.MustParse("20000000-0000-0000-0000-000000000099")
		}},
		{name: "item", mutate: func(input *shared.RevisionInput) {
			input.Sections[0].Items[0].ID = uuid.MustParse("30000000-0000-0000-0000-000000000099")
		}},
		{name: "notice", mutate: func(input *shared.RevisionInput) {
			input.Sections[0].Items[0].Notices[0].ID = uuid.MustParse("40000000-0000-0000-0000-000000000099")
		}},
		{name: "procedure step", mutate: func(input *shared.RevisionInput) {
			input.Sections[0].Items[0].ProcedureSteps[0].ID = uuid.MustParse("50000000-0000-0000-0000-000000000099")
		}},
	}
	baseline := mustCanonicalHash(t, validPublication())
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := validPublication()
			testCase.mutate(&input)
			if got := mustCanonicalHash(t, input); got == baseline {
				t.Fatalf("CanonicalRevisionHash() unchanged after changing %s UUID", testCase.name)
			}
		})
	}
}

func TestCanonicalHashChangesForEverySiblingPosition(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*shared.RevisionInput)
	}{
		{name: "section", mutate: func(input *shared.RevisionInput) { input.Sections[0].Position = 2 }},
		{name: "item", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Position = 2 }},
		{name: "notice", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].Notices[0].Position = 2 }},
		{name: "procedure step", mutate: func(input *shared.RevisionInput) { input.Sections[0].Items[0].ProcedureSteps[0].Position = 2 }},
	}
	baseline := mustCanonicalHash(t, validPublication())
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := validPublication()
			testCase.mutate(&input)
			if got := mustCanonicalHash(t, input); got == baseline {
				t.Fatalf("CanonicalRevisionHash() unchanged after changing %s position", testCase.name)
			}
		})
	}
}

func TestCanonicalHashExcludesRevisionNumberAndCallerNormalizedState(t *testing.T) {
	first := validPublication()
	firstRevisionNumber := int32(1)
	first.RevisionNumber = &firstRevisionNumber
	first.Models[0].NormalizedText = "caller-state-one"
	first.Sections[0].Models[0].NormalizedText = "caller-state-two"

	second := validPublication()
	secondRevisionNumber := int32(99)
	second.RevisionNumber = &secondRevisionNumber
	second.Models[0].NormalizedText = "different-caller-state"
	second.Sections[0].Models[0].NormalizedText = "also-different"

	firstHash := mustCanonicalHash(t, first)
	secondHash := mustCanonicalHash(t, second)
	if firstHash != secondHash {
		t.Fatalf("canonical hash includes revision number or caller normalized state: %x != %x", firstHash, secondHash)
	}
}

func TestCanonicalHashUsesUnambiguousScalarFraming(t *testing.T) {
	first := validPublication()
	first.Name = "ab"
	first.Description = "c"

	second := validPublication()
	second.Name = "a"
	second.Description = "bc"

	if firstHash, secondHash := mustCanonicalHash(t, first), mustCanonicalHash(t, second); firstHash == secondHash {
		t.Fatalf("canonical hashes collide across scalar boundaries: %x", firstHash)
	}
}

func TestCanonicalHashIsStableAcrossRepeatedCalls(t *testing.T) {
	input := validPublication()
	want := mustCanonicalHash(t, input)
	for iteration := 0; iteration < 100; iteration++ {
		if got := mustCanonicalHash(t, input); got != want {
			t.Fatalf("iteration %d hash = %x, want %x", iteration, got, want)
		}
	}
}

func TestCanonicalHashRejectsInvalidUTF8BeforeCanonicalization(t *testing.T) {
	input := validPublication()
	input.Name = string([]byte{0xff})
	_, err := shared.CanonicalRevisionHash(input)
	requireAPIError(t, err, http.StatusUnprocessableEntity, "validation_failed")
}

func mustCanonicalHash(t *testing.T, input shared.RevisionInput) [32]byte {
	t.Helper()
	hash, err := shared.CanonicalRevisionHash(input)
	if err != nil {
		t.Fatalf("CanonicalRevisionHash() error = %v", err)
	}
	return hash
}

func reversedModels(models []shared.ModelInput) []shared.ModelInput {
	result := make([]shared.ModelInput, len(models))
	for index := range models {
		result[len(models)-1-index] = models[index]
	}
	return result
}

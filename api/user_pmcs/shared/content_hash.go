package shared

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"sort"
)

const (
	hashTagRevisionID byte = iota + 1
	hashTagRevisionName
	hashTagRevisionDescription
	hashTagChecklistModelCount
	hashTagModelDisplay
	hashTagModelNormalized
	hashTagSectionCount
	hashTagSectionID
	hashTagSectionPosition
	hashTagSectionTitle
	hashTagSectionModelCount
	hashTagItemCount
	hashTagItemID
	hashTagItemPosition
	hashTagItemInterval
	hashTagItemText
	hashTagItemPerformedBy
	hashTagNoticeCount
	hashTagNoticeID
	hashTagNoticePosition
	hashTagNoticeTypePresent
	hashTagNoticeType
	hashTagNoticeText
	hashTagStepCount
	hashTagStepID
	hashTagStepPosition
	hashTagStepText
	hashTagStepFaultFoundIf
)

func CanonicalRevisionHash(input RevisionInput) ([32]byte, error) {
	canonical, err := canonicalizeRevision(input)
	if err != nil {
		return [32]byte{}, err
	}

	digest := sha256.New()
	writeRevision(digest, canonical)
	var result [32]byte
	copy(result[:], digest.Sum(nil))
	return result, nil
}

func canonicalizeRevision(input RevisionInput) (RevisionInput, error) {
	if err := validateRevisionUTF8(input); err != nil {
		return RevisionInput{}, err
	}
	canonical, err := cloneAndNormalizeRevision(input)
	if err != nil {
		return RevisionInput{}, err
	}

	sortModels(canonical.Models)
	sort.Slice(canonical.Sections, func(first, second int) bool {
		if canonical.Sections[first].Position != canonical.Sections[second].Position {
			return canonical.Sections[first].Position < canonical.Sections[second].Position
		}
		return canonical.Sections[first].ID.String() < canonical.Sections[second].ID.String()
	})
	for sectionIndex := range canonical.Sections {
		section := &canonical.Sections[sectionIndex]
		sortModels(section.Models)
		sort.Slice(section.Items, func(first, second int) bool {
			if section.Items[first].Position != section.Items[second].Position {
				return section.Items[first].Position < section.Items[second].Position
			}
			return section.Items[first].ID.String() < section.Items[second].ID.String()
		})
		for itemIndex := range section.Items {
			item := &section.Items[itemIndex]
			sort.Slice(item.Notices, func(first, second int) bool {
				if item.Notices[first].Position != item.Notices[second].Position {
					return item.Notices[first].Position < item.Notices[second].Position
				}
				return item.Notices[first].ID.String() < item.Notices[second].ID.String()
			})
			sort.Slice(item.ProcedureSteps, func(first, second int) bool {
				if item.ProcedureSteps[first].Position != item.ProcedureSteps[second].Position {
					return item.ProcedureSteps[first].Position < item.ProcedureSteps[second].Position
				}
				return item.ProcedureSteps[first].ID.String() < item.ProcedureSteps[second].ID.String()
			})
		}
	}
	return canonical, nil
}

func sortModels(models []ModelInput) {
	sort.Slice(models, func(first, second int) bool {
		if models[first].NormalizedText != models[second].NormalizedText {
			return models[first].NormalizedText < models[second].NormalizedText
		}
		return models[first].DisplayText < models[second].DisplayText
	})
}

func writeRevision(digest hash.Hash, revision RevisionInput) {
	writeScalar(digest, hashTagRevisionID, revision.ID[:])
	writeString(digest, hashTagRevisionName, revision.Name)
	writeString(digest, hashTagRevisionDescription, revision.Description)
	writeCount(digest, hashTagChecklistModelCount, len(revision.Models))
	for _, model := range revision.Models {
		writeModel(digest, model)
	}

	writeCount(digest, hashTagSectionCount, len(revision.Sections))
	for _, section := range revision.Sections {
		writeScalar(digest, hashTagSectionID, section.ID[:])
		writePosition(digest, hashTagSectionPosition, section.Position)
		writeString(digest, hashTagSectionTitle, section.Title)
		writeCount(digest, hashTagSectionModelCount, len(section.Models))
		for _, model := range section.Models {
			writeModel(digest, model)
		}

		writeCount(digest, hashTagItemCount, len(section.Items))
		for _, item := range section.Items {
			writeScalar(digest, hashTagItemID, item.ID[:])
			writePosition(digest, hashTagItemPosition, item.Position)
			writeString(digest, hashTagItemInterval, item.Interval)
			writeString(digest, hashTagItemText, item.ItemToBeCheckedOrServiced)
			writeString(digest, hashTagItemPerformedBy, item.PerformedBy)

			writeCount(digest, hashTagNoticeCount, len(item.Notices))
			for _, notice := range item.Notices {
				writeScalar(digest, hashTagNoticeID, notice.ID[:])
				writePosition(digest, hashTagNoticePosition, notice.Position)
				if notice.Type == nil {
					writeScalar(digest, hashTagNoticeTypePresent, []byte{0})
				} else {
					writeScalar(digest, hashTagNoticeTypePresent, []byte{1})
					writeString(digest, hashTagNoticeType, *notice.Type)
				}
				writeString(digest, hashTagNoticeText, notice.NoticeText)
			}

			writeCount(digest, hashTagStepCount, len(item.ProcedureSteps))
			for _, step := range item.ProcedureSteps {
				writeScalar(digest, hashTagStepID, step.ID[:])
				writePosition(digest, hashTagStepPosition, step.Position)
				writeString(digest, hashTagStepText, step.StepText)
				writeString(digest, hashTagStepFaultFoundIf, step.FaultFoundIf)
			}
		}
	}
}

func writeModel(digest hash.Hash, model ModelInput) {
	writeString(digest, hashTagModelDisplay, model.DisplayText)
	writeString(digest, hashTagModelNormalized, model.NormalizedText)
}

func writeString(digest hash.Hash, tag byte, value string) {
	writeScalar(digest, tag, []byte(value))
}

func writeCount(digest hash.Hash, tag byte, value int) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(value))
	writeScalar(digest, tag, encoded[:])
}

func writePosition(digest hash.Hash, tag byte, value int32) {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], uint32(value))
	writeScalar(digest, tag, encoded[:])
}

func writeScalar(digest hash.Hash, tag byte, value []byte) {
	_, _ = digest.Write([]byte{tag})
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write(value)
}

package vehicles

import "fmt"

// VehicleMetadataUpdate preserves omission/null before generated model defaults.
type VehicleMetadataUpdate struct {
	VehicleID                                string
	Admin, Niin, Model, Serial, Uoc, Comment *string
	Mileage, Hours                           *int32
}

type VehicleUpdateInput struct {
	Metadata                     VehicleMetadataUpdate
	TrackedMileage, TrackedHours *int32
}

func isTrackedUsageUpdate(input VehicleUpdateInput) bool {
	// Legacy usage payloads carry stale Admin/base snapshots; their role cannot
	// turn those snapshots into a request to overwrite metadata.
	metadata := input.Metadata
	return (input.TrackedMileage != nil || input.TrackedHours != nil) &&
		absentOrEmpty(metadata.Niin) && absentOrEmpty(metadata.Model) &&
		absentOrEmpty(metadata.Serial) && absentOrEmpty(metadata.Uoc) && absentOrEmpty(metadata.Comment)
}

func absentOrEmpty(value *string) bool { return value == nil || *value == "" }

func validateBaseUsage(mileage, hours *int32) error {
	if mileage != nil && *mileage < 0 {
		return fmt.Errorf("%w: mileage cannot be negative", ErrInvalidUsageAdjustment)
	}
	if hours != nil && *hours < 0 {
		return fmt.Errorf("%w: hours cannot be negative", ErrInvalidUsageAdjustment)
	}
	return nil
}

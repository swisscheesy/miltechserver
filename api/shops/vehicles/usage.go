package vehicles

import "time"

type UsageAdjustmentOperation string

const (
	UsageOperationAdd      UsageAdjustmentOperation = "add"
	UsageOperationSubtract UsageAdjustmentOperation = "subtract"
	maximumUsageAdjustment int32                    = 10_000
)

type UsageAdjustment struct {
	VehicleID         string
	Operation         UsageAdjustmentOperation
	MileageAdjustment *int32
	HoursAdjustment   *int32
	LastUpdated       time.Time
}

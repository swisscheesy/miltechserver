package response

import "time"

// NotificationSaveReceipt confirms a commit without replaying stale draft data.
type NotificationSaveReceipt struct {
	OperationID    string    `json:"operation_id"`
	NotificationID string    `json:"notification_id"`
	CommittedAt    time.Time `json:"committed_at"`
	Replayed       bool      `json:"replayed"`
}

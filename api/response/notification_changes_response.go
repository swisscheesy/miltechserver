package response

import "time"

// NotificationChangeWithUsername represents a notification change record with the username of who made the change
type NotificationChangeWithUsername struct {
	ID                string                 `json:"id"`
	NotificationID    string                 `json:"notification_id"`
	ShopID            string                 `json:"shop_id"`
	VehicleID         string                 `json:"vehicle_id"`
	ChangedBy         string                 `json:"changed_by"`
	ChangedByUsername string                 `json:"changed_by_username"`
	ChangedAt         time.Time              `json:"changed_at"`
	ChangeType        string                 `json:"change_type"`
	FieldChanges      map[string]interface{} `json:"field_changes"`
	NotificationTitle string                 `json:"notification_title"`
}

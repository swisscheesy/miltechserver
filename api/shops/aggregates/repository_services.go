package aggregates

import (
	"context"
	"database/sql"
	"fmt"

	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
)

// buildRecentChangesQuery holds the SELECT column list, joins to resolve display fields, ORDER
// BY, and LIMIT shared across scopes. whereClause and membershipJoin carry the only real
// variance between the vehicle-scoped and shop-scoped callers.
func buildRecentChangesQuery(whereClause, membershipJoin string, limitPlaceholder int) string {
	return fmt.Sprintf(`
SELECT
	c.id,
	c.notification_id,
	c.shop_id,
	c.vehicle_id,
	c.changed_by,
	COALESCE(u.username, 'Unknown User') AS changed_by_username,
	c.changed_at,
	c.change_type,
	c.field_changes,
	COALESCE(n.title, c.notification_title, 'Deleted Notification') AS notification_title,
	c.notification_type,
	COALESCE(v.admin, c.vehicle_admin) AS vehicle_admin,
	CASE WHEN c.notification_id IS NULL OR c.vehicle_id IS NULL THEN true ELSE false END AS is_deleted
FROM shop_vehicle_notification_changes c
%s
LEFT JOIN users u ON c.changed_by = u.uid
LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
LEFT JOIN shop_vehicle v ON c.vehicle_id = v.id
WHERE %s
ORDER BY c.changed_at DESC, c.id ASC
LIMIT NULLIF($%d, 0)`, membershipJoin, whereClause, limitPlaceholder)
}

func (repo *RepositoryImpl) GetVehicleRecentChanges(ctx context.Context, vehicleID string, limit int) ([]response.NotificationChangeWithUsername, error) {
	query := buildRecentChangesQuery("c.vehicle_id = $1", "", 2)

	rows, err := repo.db.QueryContext(ctx, query, vehicleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicle notification changes: %w", err)
	}
	defer rows.Close()

	changes := []response.NotificationChangeWithUsername{}
	for rows.Next() {
		change, err := scanNotificationChange(rows)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate vehicle notification changes: %w", err)
	}

	return changes, nil
}

func scanNotificationChange(scanner rowScanner) (response.NotificationChangeWithUsername, error) {
	var change response.NotificationChangeWithUsername
	var notificationID sql.NullString
	var vehicleID sql.NullString
	var changedBy sql.NullString
	var notificationType sql.NullString
	var vehicleAdmin sql.NullString
	var fieldChanges []byte

	err := scanner.Scan(
		&change.ID,
		&notificationID,
		&change.ShopID,
		&vehicleID,
		&changedBy,
		&change.ChangedByUsername,
		&change.ChangedAt,
		&change.ChangeType,
		&fieldChanges,
		&change.NotificationTitle,
		&notificationType,
		&vehicleAdmin,
		&change.IsDeleted,
	)
	if err != nil {
		return response.NotificationChangeWithUsername{}, fmt.Errorf("failed to scan notification change: %w", err)
	}

	change.NotificationID = shared.NullStringPtr(notificationID)
	change.VehicleID = shared.NullStringPtr(vehicleID)
	change.ChangedBy = shared.NullStringPtr(changedBy)
	change.NotificationType = shared.NullStringPtr(notificationType)
	change.VehicleAdmin = shared.NullStringPtr(vehicleAdmin)
	change.FieldChanges = map[string]interface{}{}
	if len(fieldChanges) > 0 {
		change.FieldChanges["raw"] = string(fieldChanges)
	}

	return change, nil
}

// buildServiceQuery holds the SELECT column list, ORDER BY, and LIMIT shared across scopes.
// Both scopes previously hand-rolled this query separately and diverged on ORDER BY direction
// (vehicle: DESC, shop: ASC) with nothing to catch it — the one canonical ORDER BY here (newest
// service first, matching every other time-ordered list in this file) makes that divergence
// structurally impossible going forward.
func buildServiceQuery(whereClause, membershipJoin string, limitPlaceholder int) string {
	return fmt.Sprintf(`
SELECT
	es.id, es.shop_id, es.equipment_id, es.list_id, es.description, es.service_type,
	es.created_by, COALESCE(u.username, 'Unknown User') AS created_by_username,
	es.is_completed, es.created_at, es.updated_at, es.service_date, es.service_hours,
	es.completion_date
FROM equipment_services es
%s
LEFT JOIN users u ON u.uid = es.created_by
WHERE %s
ORDER BY es.service_date DESC NULLS LAST, es.created_at DESC, es.id ASC
LIMIT NULLIF($%d, 0)`, membershipJoin, whereClause, limitPlaceholder)
}

func scanEquipmentService(scanner rowScanner) (response.EquipmentServiceResponse, error) {
	var service response.EquipmentServiceResponse
	var serviceDate sql.NullTime
	var serviceHours sql.NullInt64
	var completionDate sql.NullTime
	err := scanner.Scan(
		&service.ID,
		&service.ShopID,
		&service.EquipmentID,
		&service.ListID,
		&service.Description,
		&service.ServiceType,
		&service.CreatedBy,
		&service.CreatedByUsername,
		&service.IsCompleted,
		&service.CreatedAt,
		&service.UpdatedAt,
		&serviceDate,
		&serviceHours,
		&completionDate,
	)
	if err != nil {
		return response.EquipmentServiceResponse{}, fmt.Errorf("failed to scan equipment service: %w", err)
	}
	service.ServiceDate = shared.NullTimePtr(serviceDate)
	service.ServiceHours = shared.NullInt32Ptr(serviceHours)
	service.CompletionDate = shared.NullTimePtr(completionDate)
	return service, nil
}

func (repo *RepositoryImpl) GetVehicleServices(ctx context.Context, vehicleID string, limit int) ([]response.EquipmentServiceResponse, error) {
	query := buildServiceQuery("es.equipment_id = $1", "", 2)

	rows, err := repo.db.QueryContext(ctx, query, vehicleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicle equipment services: %w", err)
	}
	defer rows.Close()

	services := []response.EquipmentServiceResponse{}
	for rows.Next() {
		service, err := scanEquipmentService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate vehicle equipment services: %w", err)
	}

	return services, nil
}

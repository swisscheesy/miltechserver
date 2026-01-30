package user_vehicles_test

import (
	"net/http"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserVehiclesLifecycle(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "user-vehicle-lifecycle"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-99",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-99",
		Uoc:         "UNK",
		Mileage:     99,
		Hours:       10,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	upsertVehicle := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", vehicle, userID)
	require.Equal(t, http.StatusOK, upsertVehicle.Code)

	notification := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      userID,
		VehicleID:   vehicle.ID,
		Title:       "Inspection",
		Description: "Desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	upsertNotification := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicle-notifications", notification, userID)
	require.Equal(t, http.StatusOK, upsertNotification.Code)

	item := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         userID,
		NotificationID: notification.ID,
		Niin:           "I-99",
		Nomenclature:   "Item",
		Quantity:       2,
		SaveTime:       now,
	}

	upsertItem := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/notification-items", item, userID)
	require.Equal(t, http.StatusOK, upsertItem.Code)

	deleteNotificationItems := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/notification-items/notification/"+notification.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteNotificationItems.Code)

	deleteVehicleNotifications := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/vehicle-notifications/vehicle/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteVehicleNotifications.Code)

	deleteVehicle := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/vehicles/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteVehicle.Code)
}

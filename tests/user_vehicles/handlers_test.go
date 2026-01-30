package user_vehicles_test

import (
	"net/http"
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVehicleHandlersLifecycle(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-vehicles"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-1",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-1",
		Uoc:         "UNK",
		Mileage:     5,
		Hours:       2,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	upsertResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", vehicle, userID)
	require.Equal(t, http.StatusOK, upsertResp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicles", nil, userID)
	require.Equal(t, http.StatusOK, getResp.Code)
	payload := decodeStandardResponse(t, getResp.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 1)

	getByID := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicles/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, getByID.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/vehicles/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteResp.Code)

	afterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicles", nil, userID)
	require.Equal(t, http.StatusOK, afterDelete.Code)
	payload = decodeStandardResponse(t, afterDelete.Body)
	items = decodeSlice(t, payload.Data)
	require.Len(t, items, 0)
}

func TestVehicleHandlersDeleteAll(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-vehicles-delete-all"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	first := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-1",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-1",
		Uoc:         "UNK",
		Mileage:     5,
		Hours:       2,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}
	second := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-2",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-2",
		Uoc:         "UNK",
		Mileage:     7,
		Hours:       3,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", first, userID).Code)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", second, userID).Code)

	deleteAll := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/vehicles", nil, userID)
	require.Equal(t, http.StatusOK, deleteAll.Code)

	afterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicles", nil, userID)
	require.Equal(t, http.StatusOK, afterDelete.Code)
	payload := decodeStandardResponse(t, afterDelete.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 0)
}

func TestVehicleHandlersErrors(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-vehicles-errors"
	ensureUser(t, testDB, userID)

	unauthResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicles", nil, "")
	require.Equal(t, http.StatusUnauthorized, unauthResp.Code)

	invalidJSON := doRawRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", "{", userID)
	require.Equal(t, http.StatusBadRequest, invalidJSON.Code)
}

func TestNotificationHandlersLifecycle(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-notifications"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-2",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-2",
		Uoc:         "UNK",
		Mileage:     5,
		Hours:       2,
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
		Title:       "Title",
		Description: "Desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	upsertResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicle-notifications", notification, userID)
	require.Equal(t, http.StatusOK, upsertResp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicle-notifications", nil, userID)
	require.Equal(t, http.StatusOK, getResp.Code)
	payload := decodeStandardResponse(t, getResp.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 1)

	getByVehicle := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicle-notifications/vehicle/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, getByVehicle.Code)

	getByID := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicle-notifications/"+notification.ID, nil, userID)
	require.Equal(t, http.StatusOK, getByID.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/vehicle-notifications/"+notification.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteResp.Code)

	afterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicle-notifications", nil, userID)
	require.Equal(t, http.StatusOK, afterDelete.Code)
	payload = decodeStandardResponse(t, afterDelete.Body)
	items = decodeSlice(t, payload.Data)
	require.Len(t, items, 0)
}

func TestNotificationHandlersDeleteAllByVehicle(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-notifications-delete-all"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-3",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-3",
		Uoc:         "UNK",
		Mileage:     5,
		Hours:       2,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", vehicle, userID).Code)

	first := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      userID,
		VehicleID:   vehicle.ID,
		Title:       "Title",
		Description: "Desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}
	second := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      userID,
		VehicleID:   vehicle.ID,
		Title:       "Title2",
		Description: "Desc2",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicle-notifications", first, userID).Code)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicle-notifications", second, userID).Code)

	deleteAll := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/vehicle-notifications/vehicle/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteAll.Code)

	afterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicle-notifications/vehicle/"+vehicle.ID, nil, userID)
	require.Equal(t, http.StatusOK, afterDelete.Code)
	payload := decodeStandardResponse(t, afterDelete.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 0)
}

func TestNotificationHandlersErrors(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-notifications-errors"
	ensureUser(t, testDB, userID)

	unauthResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/vehicle-notifications", nil, "")
	require.Equal(t, http.StatusUnauthorized, unauthResp.Code)

	invalidJSON := doRawRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicle-notifications", "{", userID)
	require.Equal(t, http.StatusBadRequest, invalidJSON.Code)
}

func TestNotificationItemsHandlersLifecycle(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-items"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-4",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-4",
		Uoc:         "UNK",
		Mileage:     5,
		Hours:       2,
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
		Title:       "Title",
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
		Niin:           "I-1",
		Nomenclature:   "Item",
		Quantity:       2,
		SaveTime:       now,
	}

	upsertResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/notification-items", item, userID)
	require.Equal(t, http.StatusOK, upsertResp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/notification-items", nil, userID)
	require.Equal(t, http.StatusOK, getResp.Code)
	payload := decodeStandardResponse(t, getResp.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 1)

	byNotification := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/notification-items/notification/"+notification.ID, nil, userID)
	require.Equal(t, http.StatusOK, byNotification.Code)

	byID := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/notification-items/"+item.ID, nil, userID)
	require.Equal(t, http.StatusOK, byID.Code)

	second := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         userID,
		NotificationID: notification.ID,
		Niin:           "I-2",
		Nomenclature:   "Item2",
		Quantity:       1,
		SaveTime:       now,
	}

	batchResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/notification-items/list", []model.UserNotificationItems{item, second}, userID)
	require.Equal(t, http.StatusOK, batchResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/notification-items/"+item.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteResp.Code)

	afterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/notification-items", nil, userID)
	require.Equal(t, http.StatusOK, afterDelete.Code)
	payload = decodeStandardResponse(t, afterDelete.Body)
	items = decodeSlice(t, payload.Data)
	require.Len(t, items, 1)
}

func TestNotificationItemsHandlersDeleteAllByNotification(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-items-delete-all"
	ensureUser(t, testDB, userID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      userID,
		Niin:        "N-5",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-5",
		Uoc:         "UNK",
		Mileage:     5,
		Hours:       2,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicles", vehicle, userID).Code)

	notification := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      userID,
		VehicleID:   vehicle.ID,
		Title:       "Title",
		Description: "Desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/vehicle-notifications", notification, userID).Code)

	first := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         userID,
		NotificationID: notification.ID,
		Niin:           "I-1",
		Nomenclature:   "Item",
		Quantity:       2,
		SaveTime:       now,
	}
	second := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         userID,
		NotificationID: notification.ID,
		Niin:           "I-2",
		Nomenclature:   "Item2",
		Quantity:       1,
		SaveTime:       now,
	}

	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/notification-items", first, userID).Code)
	require.Equal(t, http.StatusOK, doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/user/notification-items", second, userID).Code)

	deleteAll := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/user/notification-items/notification/"+notification.ID, nil, userID)
	require.Equal(t, http.StatusOK, deleteAll.Code)

	afterDelete := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/notification-items/notification/"+notification.ID, nil, userID)
	require.Equal(t, http.StatusOK, afterDelete.Code)
	payload := decodeStandardResponse(t, afterDelete.Body)
	items := decodeSlice(t, payload.Data)
	require.Len(t, items, 0)
}

func TestNotificationItemsHandlersErrors(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	router := newTestRouter(t)
	userID := "handler-items-errors"
	ensureUser(t, testDB, userID)

	unauthResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/user/notification-items", nil, "")
	require.Equal(t, http.StatusUnauthorized, unauthResp.Code)

	invalidJSON := doRawRequest(t, router, http.MethodPut, "/api/v1/auth/user/notification-items", "{", userID)
	require.Equal(t, http.StatusBadRequest, invalidJSON.Code)
}

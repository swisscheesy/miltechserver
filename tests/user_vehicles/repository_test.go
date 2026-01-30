package user_vehicles_test

import (
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/user_vehicles/notification_items"
	"miltechserver/api/user_vehicles/notifications"
	"miltechserver/api/user_vehicles/vehicles"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVehiclesRepositoryCRUD(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-vehicles"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
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

	repo := vehicles.NewRepository(testDB)
	require.NoError(t, repo.Upsert(user, vehicle))

	items, err := repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 1)

	found, err := repo.GetByID(user, vehicle.ID)
	require.NoError(t, err)
	require.Equal(t, vehicle.ID, found.ID)

	require.NoError(t, repo.Delete(user, vehicle.ID))
	items, err = repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestVehiclesRepositoryDeleteAll(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-vehicles-delete-all"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	repo := vehicles.NewRepository(testDB)

	first := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "N-11",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-11",
		Uoc:         "UNK",
		Mileage:     1,
		Hours:       1,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}
	second := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "N-12",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-12",
		Uoc:         "UNK",
		Mileage:     2,
		Hours:       2,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	require.NoError(t, repo.Upsert(user, first))
	require.NoError(t, repo.Upsert(user, second))

	items, err := repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.NoError(t, repo.DeleteAll(user))
	items, err = repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestNotificationsRepositoryCRUD(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-notifications"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "N-2",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-2",
		Uoc:         "UNK",
		Mileage:     1,
		Hours:       1,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	vehicleRepo := vehicles.NewRepository(testDB)
	require.NoError(t, vehicleRepo.Upsert(user, vehicle))

	notification := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		VehicleID:   vehicle.ID,
		Title:       "Check",
		Description: "desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	repo := notifications.NewRepository(testDB)
	require.NoError(t, repo.Upsert(user, notification))

	items, err := repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 1)

	byVehicle, err := repo.GetByVehicleID(user, vehicle.ID)
	require.NoError(t, err)
	require.Len(t, byVehicle, 1)

	found, err := repo.GetByID(user, notification.ID)
	require.NoError(t, err)
	require.Equal(t, notification.ID, found.ID)

	require.NoError(t, repo.Delete(user, notification.ID))
	items, err = repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestNotificationsRepositoryDeleteAllByVehicle(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-notifications-delete-all"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "N-21",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-21",
		Uoc:         "UNK",
		Mileage:     1,
		Hours:       1,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	vehicleRepo := vehicles.NewRepository(testDB)
	require.NoError(t, vehicleRepo.Upsert(user, vehicle))

	repo := notifications.NewRepository(testDB)
	first := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		VehicleID:   vehicle.ID,
		Title:       "Check",
		Description: "desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}
	second := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		VehicleID:   vehicle.ID,
		Title:       "Check2",
		Description: "desc2",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	require.NoError(t, repo.Upsert(user, first))
	require.NoError(t, repo.Upsert(user, second))

	items, err := repo.GetByVehicleID(user, vehicle.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.NoError(t, repo.DeleteAllByVehicle(user, vehicle.ID))
	items, err = repo.GetByVehicleID(user, vehicle.ID)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

func TestNotificationItemsRepositoryCRUD(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-items"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "N-4",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-4",
		Uoc:         "UNK",
		Mileage:     1,
		Hours:       1,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	vehicleRepo := vehicles.NewRepository(testDB)
	require.NoError(t, vehicleRepo.Upsert(user, vehicle))

	notification := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		VehicleID:   vehicle.ID,
		Title:       "Check",
		Description: "desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	notificationsRepo := notifications.NewRepository(testDB)
	require.NoError(t, notificationsRepo.Upsert(user, notification))

	item := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         user.UserID,
		NotificationID: notification.ID,
		Niin:           "I-1",
		Nomenclature:   "Item",
		Quantity:       2,
		SaveTime:       now,
	}

	repo := notification_items.NewRepository(testDB)
	require.NoError(t, repo.Upsert(user, item))

	items, err := repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 1)

	byNotification, err := repo.GetByNotificationID(user, notification.ID)
	require.NoError(t, err)
	require.Len(t, byNotification, 1)

	found, err := repo.GetByID(user, item.ID)
	require.NoError(t, err)
	require.Equal(t, item.ID, found.ID)

	second := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         user.UserID,
		NotificationID: notification.ID,
		Niin:           "I-2",
		Nomenclature:   "Item2",
		Quantity:       1,
		SaveTime:       now,
	}

	require.NoError(t, repo.UpsertBatch(user, []model.UserNotificationItems{item, second}))

	items, err = repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.NoError(t, repo.Delete(user, item.ID))
	items, err = repo.GetByUserID(user)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestNotificationItemsRepositoryDeleteAllByNotification(t *testing.T) {
	clearUserVehicleTables(t, testDB)
	user := &bootstrap.User{UserID: "repo-items-delete-all"}
	ensureUser(t, testDB, user.UserID)

	now := time.Now().UTC()
	vehicle := model.UserVehicle{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		Niin:        "N-31",
		Admin:       "admin",
		Model:       "Model",
		Serial:      "SER-31",
		Uoc:         "UNK",
		Mileage:     1,
		Hours:       1,
		Comment:     "comment",
		SaveTime:    now,
		LastUpdated: now,
	}

	vehicleRepo := vehicles.NewRepository(testDB)
	require.NoError(t, vehicleRepo.Upsert(user, vehicle))

	notification := model.UserVehicleNotifications{
		ID:          uuid.New().String(),
		UserID:      user.UserID,
		VehicleID:   vehicle.ID,
		Title:       "Check",
		Description: "desc",
		Type:        "PM",
		Completed:   false,
		SaveTime:    now,
		LastUpdated: now,
	}

	notificationsRepo := notifications.NewRepository(testDB)
	require.NoError(t, notificationsRepo.Upsert(user, notification))

	repo := notification_items.NewRepository(testDB)
	first := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         user.UserID,
		NotificationID: notification.ID,
		Niin:           "I-1",
		Nomenclature:   "Item",
		Quantity:       2,
		SaveTime:       now,
	}
	second := model.UserNotificationItems{
		ID:             uuid.New().String(),
		UserID:         user.UserID,
		NotificationID: notification.ID,
		Niin:           "I-2",
		Nomenclature:   "Item2",
		Quantity:       1,
		SaveTime:       now,
	}

	require.NoError(t, repo.Upsert(user, first))
	require.NoError(t, repo.Upsert(user, second))

	items, err := repo.GetByNotificationID(user, notification.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.NoError(t, repo.DeleteAllByNotification(user, notification.ID))
	items, err = repo.GetByNotificationID(user, notification.ID)
	require.NoError(t, err)
	require.Len(t, items, 0)
}

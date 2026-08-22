package vehicles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"strings"
	"time"

	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo: repo,
		auth: auth,
	}
}

func (service *ServiceImpl) WithAuthorization(auth shared.ShopAuthorization) shared.AuthorizationAware {
	return &ServiceImpl{
		repo: service.repo,
		auth: auth,
	}
}

func (service *ServiceImpl) CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	vehicle.ID = uuid.New().String()
	vehicle.CreatorID = user.UserID
	now := time.Now().UTC()
	vehicle.SaveTime = now
	vehicle.LastUpdated = now

	if vehicle.Uoc == "" {
		vehicle.Uoc = "UNK"
	}

	createdVehicle, err := service.repo.CreateShopVehicle(user, vehicle)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle created", "user_id", user.UserID, "shop_id", vehicle.ShopID, "vehicle_id", vehicle.ID)
	return createdVehicle, nil
}

func (service *ServiceImpl) GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	vehicles, err := service.repo.GetShopVehicles(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	if vehicles == nil {
		return []model.ShopVehicle{}, nil
	}

	return vehicles, nil
}

func (service *ServiceImpl) GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicle: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return vehicle, nil
}

func (service *ServiceImpl) UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	currentVehicle, err := service.repo.GetShopVehicleByID(user, vehicle.ID)
	if err != nil {
		return fmt.Errorf("failed to get current vehicle: %w", err)
	}

	vehicle.ShopID = currentVehicle.ShopID
	isMember, err := service.auth.IsUserMemberOfShop(user, currentVehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	isCreator := currentVehicle.CreatorID == user.UserID
	isAdmin, err := service.auth.IsUserShopAdmin(user, currentVehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isCreator && !isAdmin {
		if !isTrackedUsageUpdate(vehicle) {
			return errors.New("access denied: only vehicle creator or shop admin can update equipment details")
		}

		usageUpdate := ShopVehicleUsageUpdate{
			VehicleID:      vehicle.ID,
			TrackedMileage: vehicle.TrackedMileage,
			TrackedHours:   vehicle.TrackedHours,
			LastUpdated:    time.Now().UTC(),
		}
		if err := service.repo.UpdateShopVehicleUsage(user, usageUpdate); err != nil {
			return fmt.Errorf("failed to update shop vehicle usage: %w", err)
		}

		slog.Info("Shop vehicle usage updated", "user_id", user.UserID, "vehicle_id", vehicle.ID)
		return nil
	}

	if vehicle.Uoc == "" {
		vehicle.Uoc = "UNK"
	}

	vehicle.LastUpdated = time.Now().UTC()

	err = service.repo.UpdateShopVehicle(user, vehicle)
	if err != nil {
		return fmt.Errorf("failed to update shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle updated", "user_id", user.UserID, "vehicle_id", vehicle.ID)
	return nil
}

func (service *ServiceImpl) AdjustShopVehicleUsage(
	ctx context.Context,
	user *bootstrap.User,
	adjustment UsageAdjustment,
) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	adjustment, err := normalizeUsageAdjustment(adjustment)
	if err != nil {
		return nil, err
	}

	currentVehicle, err := service.repo.GetShopVehicleByID(user, adjustment.VehicleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, shared.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("failed to get current vehicle: %w", err)
	}

	if err := service.auth.RequireShopMember(user, currentVehicle.ShopID); err != nil {
		return nil, err
	}

	adjustment.LastUpdated = time.Now().UTC()
	updatedVehicle, err := service.repo.AdjustShopVehicleUsage(ctx, adjustment)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust shop vehicle usage: %w", err)
	}

	slog.Info("Shop vehicle usage adjusted", "user_id", user.UserID, "vehicle_id", adjustment.VehicleID)
	return updatedVehicle, nil
}

func normalizeUsageAdjustment(adjustment UsageAdjustment) (UsageAdjustment, error) {
	adjustment.Operation = UsageAdjustmentOperation(
		strings.TrimSpace(string(adjustment.Operation)),
	)
	if adjustment.Operation == "" {
		adjustment.Operation = UsageOperationAdd
	}
	if adjustment.Operation != UsageOperationAdd && adjustment.Operation != UsageOperationSubtract {
		return UsageAdjustment{}, fmt.Errorf("%w: operation must be add or subtract", ErrInvalidUsageAdjustment)
	}

	values := []*int32{adjustment.MileageAdjustment, adjustment.HoursAdjustment}
	hasPositive := false
	for _, value := range values {
		if value == nil {
			continue
		}
		if *value < 0 || *value > maximumUsageAdjustment {
			return UsageAdjustment{}, fmt.Errorf("%w: adjustments must be between 0 and 10000", ErrInvalidUsageAdjustment)
		}
		hasPositive = hasPositive || *value > 0
	}
	if !hasPositive {
		return UsageAdjustment{}, fmt.Errorf("%w: at least one adjustment must be greater than zero", ErrInvalidUsageAdjustment)
	}

	return adjustment, nil
}

func isTrackedUsageUpdate(vehicle model.ShopVehicle) bool {
	// The usage client carries admin and base readings from its snapshot. The
	// usage-only repository ignores them so stale values cannot overwrite details.
	hasUsageValue := vehicle.TrackedMileage != nil || vehicle.TrackedHours != nil
	return hasUsageValue &&
		vehicle.Niin == "" &&
		vehicle.Model == "" &&
		vehicle.Serial == "" &&
		vehicle.Uoc == "" &&
		vehicle.Comment == ""
}

func (service *ServiceImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(user, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	isCreator := vehicle.CreatorID == user.UserID
	isAdmin, err := service.auth.IsUserShopAdmin(user, vehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isCreator && !isAdmin {
		return errors.New("access denied: only vehicle creator or shop admin can delete vehicles")
	}

	vehicleDeletionChange := model.ShopVehicleNotificationChanges{
		NotificationID:    nil,
		ShopID:            vehicle.ShopID,
		VehicleID:         &vehicleID,
		ChangedBy:         &user.UserID,
		ChangeType:        "vehicle_deleted",
		FieldChanges:      buildVehicleDeletionFieldChanges(vehicle),
		NotificationTitle: nil,
		NotificationType:  nil,
		VehicleAdmin:      &vehicle.Admin,
	}

	err = service.repo.CreateNotificationChange(user, vehicleDeletionChange)
	if err != nil {
		slog.Warn("Failed to record vehicle deletion audit", "error", err, "vehicle_id", vehicleID)
	}

	err = service.repo.DeleteShopVehicle(user, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to delete shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle deleted", "user_id", user.UserID, "vehicle_id", vehicleID, "vehicle_admin", vehicle.Admin)
	return nil
}

// buildVehicleDeletionFieldChanges creates field changes JSON for vehicle deletion
func buildVehicleDeletionFieldChanges(vehicle *model.ShopVehicle) string {
	type VehicleData struct {
		Admin   string `json:"admin"`
		Niin    string `json:"niin"`
		Uoc     string `json:"uoc"`
		Mileage int32  `json:"mileage"`
		Hours   int32  `json:"hours"`
		Comment string `json:"comment"`
	}

	type FieldChangesData struct {
		Deleted     bool        `json:"deleted"`
		VehicleData VehicleData `json:"vehicle_data"`
	}

	data := FieldChangesData{
		Deleted: true,
		VehicleData: VehicleData{
			Admin:   vehicle.Admin,
			Niin:    vehicle.Niin,
			Uoc:     vehicle.Uoc,
			Mileage: vehicle.Mileage,
			Hours:   vehicle.Hours,
			Comment: vehicle.Comment,
		},
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.Warn("Failed to marshal vehicle deletion field changes", "error", err)
		return `{"deleted": true}`
	}

	return string(jsonBytes)
}

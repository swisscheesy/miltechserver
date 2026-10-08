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

func (service *ServiceImpl) CreateShopVehicle(ctx context.Context, user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if err := validateBaseUsage(&vehicle.Mileage, &vehicle.Hours); err != nil {
		return nil, err
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, vehicle.ShopID)
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

	createdVehicle, err := service.repo.CreateShopVehicle(ctx, user, vehicle)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop vehicle: %w", err)
	}

	slog.Info("Shop vehicle created", "user_id", user.UserID, "shop_id", vehicle.ShopID, "vehicle_id", vehicle.ID)
	return createdVehicle, nil
}

func (service *ServiceImpl) GetShopVehicles(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	vehicles, err := service.repo.GetShopVehicles(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	if vehicles == nil {
		return []model.ShopVehicle{}, nil
	}

	return vehicles, nil
}

func (service *ServiceImpl) GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicle: %w", err)
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, vehicle.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	return vehicle, nil
}

func (service *ServiceImpl) UpdateShopVehicle(ctx context.Context, user *bootstrap.User, input VehicleUpdateInput) error {
	if user == nil {
		return errors.New("unauthorized user")
	}
	if err := validateAbsoluteTrackedUsage(input); err != nil {
		return err
	}
	isUsageOnly := isTrackedUsageUpdate(input)
	if !isUsageOnly {
		if err := validateBaseUsage(input.Metadata.Mileage, input.Metadata.Hours); err != nil {
			return err
		}
		if input.Metadata.Admin != nil && *input.Metadata.Admin == "" {
			return fmt.Errorf("%w: admin cannot be empty", ErrInvalidUsageAdjustment)
		}
	}
	currentVehicle, err := service.repo.GetShopVehicleByID(ctx, user, input.Metadata.VehicleID)
	if err != nil {
		return fmt.Errorf("failed to get current vehicle: %w", err)
	}
	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, currentVehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}
	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}
	if isUsageOnly {
		update := ShopVehicleUsageUpdate{VehicleID: input.Metadata.VehicleID, TrackedMileage: input.TrackedMileage, TrackedHours: input.TrackedHours, LastUpdated: time.Now().UTC()}
		if err := service.repo.UpdateShopVehicleUsage(ctx, user, update); err != nil {
			return fmt.Errorf("failed to update shop vehicle usage: %w", err)
		}
		slog.Info("Shop vehicle usage updated", "user_id", user.UserID, "vehicle_id", input.Metadata.VehicleID)
		return nil
	}
	isCreator := currentVehicle.CreatorID == user.UserID
	isAdmin, err := service.auth.IsUserShopAdmin(ctx, user, currentVehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}
	if !isCreator && !isAdmin {
		return errors.New("access denied: only vehicle creator or shop admin can update equipment details")
	}
	if input.Metadata.Uoc != nil && *input.Metadata.Uoc == "" {
		value := "UNK"
		input.Metadata.Uoc = &value
	}
	if err := service.repo.UpdateShopVehicleMetadata(ctx, user, input); err != nil {
		return fmt.Errorf("failed to update shop vehicle: %w", err)
	}
	slog.Info("Shop vehicle updated", "user_id", user.UserID, "vehicle_id", input.Metadata.VehicleID)
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

	currentVehicle, err := service.repo.GetShopVehicleByID(ctx, user, adjustment.VehicleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, shared.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("failed to get current vehicle: %w", err)
	}

	if err := service.auth.RequireShopMember(ctx, user, currentVehicle.ShopID); err != nil {
		return nil, err
	}

	adjustment.LastUpdated = time.Now().UTC()
	updatedVehicle, err := service.repo.AdjustShopVehicleUsage(ctx, user, adjustment)
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

func validateAbsoluteTrackedUsage(vehicle VehicleUpdateInput) error {
	if vehicle.TrackedMileage != nil && *vehicle.TrackedMileage < 0 {
		return fmt.Errorf("%w: tracked_mileage cannot be negative", ErrInvalidUsageAdjustment)
	}
	if vehicle.TrackedHours != nil && *vehicle.TrackedHours < 0 {
		return fmt.Errorf("%w: tracked_hours cannot be negative", ErrInvalidUsageAdjustment)
	}
	return nil
}

func (service *ServiceImpl) DeleteShopVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	vehicle, err := service.repo.GetShopVehicleByID(ctx, user, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to get vehicle: %w", err)
	}

	if err := service.auth.RequireShopMember(ctx, user, vehicle.ShopID); err != nil {
		return err
	}

	isCreator := vehicle.CreatorID == user.UserID
	isAdmin, err := service.auth.IsUserShopAdmin(ctx, user, vehicle.ShopID)
	if err != nil {
		return fmt.Errorf("failed to verify admin status: %w", err)
	}

	if !isCreator && !isAdmin {
		return errors.New("access denied: only vehicle creator or shop admin can delete vehicles")
	}

	err = service.repo.DeleteShopVehicle(ctx, user, vehicleID)
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

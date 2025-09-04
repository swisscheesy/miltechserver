package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
)

type EquipmentServicesRepositoryImpl struct {
	Db *sql.DB
}

// Result structs for single-column queries
type ShopIDResult struct {
	ShopID string `alias:"shop_vehicle.shop_id"`
}

type ListShopIDResult struct {
	ShopID string `alias:"shop_lists.shop_id"`
}

type UsernameResult struct {
	Username string `alias:"users.username"`
}

func NewEquipmentServicesRepositoryImpl(db *sql.DB) *EquipmentServicesRepositoryImpl {
	return &EquipmentServicesRepositoryImpl{Db: db}
}

// CreateEquipmentService creates a new equipment service
func (repo *EquipmentServicesRepositoryImpl) CreateEquipmentService(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error) {
	// Validate user has access to shop through equipment
	shopID, err := repo.ValidateEquipmentAccess(user, service.EquipmentID)
	if err != nil {
		return nil, fmt.Errorf("equipment access validation failed: %w", err)
	}

	// Validate list belongs to same shop (only if ListID is provided)
	if service.ListID != "" {
		listShopID, err := repo.ValidateListAccess(user, service.ListID)
		if err != nil {
			return nil, fmt.Errorf("list access validation failed: %w", err)
		}

		if shopID != listShopID {
			return nil, errors.New("equipment and list must belong to the same shop")
		}
	}

	// Set the shop ID
	service.ShopID = shopID

	// Use go-jet for type-safe insert
	stmt := EquipmentServices.INSERT(
		EquipmentServices.ID,
		EquipmentServices.ShopID,
		EquipmentServices.EquipmentID,
		EquipmentServices.ListID,
		EquipmentServices.Description,
		EquipmentServices.ServiceType,
		EquipmentServices.CreatedBy,
		EquipmentServices.IsCompleted,
		EquipmentServices.CreatedAt,
		EquipmentServices.UpdatedAt,
		EquipmentServices.ServiceDate,
		EquipmentServices.ServiceHours,
		EquipmentServices.CompletionDate,
	).MODEL(service).RETURNING(EquipmentServices.AllColumns)

	var createdService model.EquipmentServices
	err = stmt.Query(repo.Db, &createdService)
	if err != nil {
		return nil, fmt.Errorf("failed to create equipment service: %w", err)
	}

	slog.Info("Equipment service created", "service_id", service.ID, "created_by", user.UserID)
	return &createdService, nil
}

// GetEquipmentServiceByID retrieves a specific equipment service by ID
func (repo *EquipmentServicesRepositoryImpl) GetEquipmentServiceByID(user *bootstrap.User, serviceID string) (*model.EquipmentServices, error) {
	stmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var service model.EquipmentServices
	err := stmt.Query(repo.Db, &service)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment service: %w", err)
	}

	return &service, nil
}

// UpdateEquipmentService updates an existing equipment service
func (repo *EquipmentServicesRepositoryImpl) UpdateEquipmentService(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error) {
	// Set updated timestamp
	now := time.Now()
	service.UpdatedAt = now

	stmt := EquipmentServices.UPDATE(
		EquipmentServices.Description,
		EquipmentServices.ServiceType,
		EquipmentServices.IsCompleted,
		EquipmentServices.ServiceDate,
		EquipmentServices.ServiceHours,
		EquipmentServices.CompletionDate,
		EquipmentServices.UpdatedAt,
	).MODEL(service).WHERE(
		EquipmentServices.ID.EQ(String(service.ID)).
			AND(EquipmentServices.ShopID.IN(
				SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
			)),
	).RETURNING(EquipmentServices.AllColumns)

	var updatedService model.EquipmentServices
	err := stmt.Query(repo.Db, &updatedService)
	if err != nil {
		return nil, fmt.Errorf("failed to update equipment service: %w", err)
	}

	slog.Info("Equipment service updated", "service_id", service.ID, "updated_by", user.UserID)
	return &updatedService, nil
}

// DeleteEquipmentService deletes an equipment service
func (repo *EquipmentServicesRepositoryImpl) DeleteEquipmentService(user *bootstrap.User, serviceID string) error {
	stmt := EquipmentServices.DELETE().WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.ShopID.IN(
				SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
			)),
	)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete equipment service: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("equipment service not found or access denied")
	}

	slog.Info("Equipment service deleted", "service_id", serviceID, "deleted_by", user.UserID)
	return nil
}

// GetEquipmentServices retrieves equipment services with filtering and pagination
func (repo *EquipmentServicesRepositoryImpl) GetEquipmentServices(user *bootstrap.User, shopID string, filters request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error) {
	// Build conditions based on filters
	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		ShopMembers.UserID.EQ(String(user.UserID)),
	}

	// Add optional filters
	if filters.EquipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*filters.EquipmentID)))
	}

	if filters.ServiceType != nil {
		conditions = append(conditions, EquipmentServices.ServiceType.LIKE(String("%"+*filters.ServiceType+"%")))
	}

	if filters.IsCompleted != nil {
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(*filters.IsCompleted)))
	}

	if filters.StartDate != nil {
		startTime, err := time.Parse(time.RFC3339, *filters.StartDate)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid start_date format: %w", err)
		}
		conditions = append(conditions, EquipmentServices.ServiceDate.GT_EQ(TimestampzT(startTime)))
	}

	if filters.EndDate != nil {
		endTime, err := time.Parse(time.RFC3339, *filters.EndDate)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid end_date format: %w", err)
		}
		conditions = append(conditions, EquipmentServices.ServiceDate.LT_EQ(TimestampzT(endTime)))
	}

	// Count query for pagination
	countStmt := SELECT(COUNT(STAR)).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...))

	var totalCount int64
	err := countStmt.Query(repo.Db, &totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}

	// Data query with pagination
	dataStmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.CreatedAt.DESC()).
		LIMIT(int64(filters.Limit)).
		OFFSET(int64(filters.Offset))

	var services []model.EquipmentServices
	err = dataStmt.Query(repo.Db, &services)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get services: %w", err)
	}

	return services, totalCount, nil
}

// GetServicesByEquipment retrieves services for a specific equipment
func (repo *EquipmentServicesRepositoryImpl) GetServicesByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) ([]model.EquipmentServices, int64, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.EquipmentID.EQ(String(equipmentID)),
		ShopMembers.UserID.EQ(String(user.UserID)),
	}

	if startDate != nil {
		conditions = append(conditions, EquipmentServices.ServiceDate.GT_EQ(TimestampzT(*startDate)))
	}
	if endDate != nil {
		conditions = append(conditions, EquipmentServices.ServiceDate.LT_EQ(TimestampzT(*endDate)))
	}

	// Count query
	countStmt := SELECT(COUNT(STAR)).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...))

	var totalCount int64
	err := countStmt.Query(repo.Db, &totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}

	// Data query
	dataStmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	var services []model.EquipmentServices
	err = dataStmt.Query(repo.Db, &services)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get services: %w", err)
	}

	return services, totalCount, nil
}

// GetServicesInDateRange retrieves services within a specific date range
func (repo *EquipmentServicesRepositoryImpl) GetServicesInDateRange(user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		EquipmentServices.ServiceDate.IS_NOT_NULL(),
		EquipmentServices.ServiceDate.GT_EQ(TimestampzT(startDate)),
		EquipmentServices.ServiceDate.LT_EQ(TimestampzT(endDate)),
		ShopMembers.UserID.EQ(String(user.UserID)),
	}

	if equipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*equipmentID)))
	}

	stmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC())

	var services []model.EquipmentServices
	err := stmt.Query(repo.Db, &services)
	if err != nil {
		return nil, fmt.Errorf("failed to get services in date range: %w", err)
	}

	return services, nil
}

// GetOverdueServices retrieves services that are overdue
func (repo *EquipmentServicesRepositoryImpl) GetOverdueServices(user *bootstrap.User, shopID string, equipmentID *string, limit int) ([]response.OverdueServiceResponse, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		EquipmentServices.ServiceDate.IS_NOT_NULL(),
		postgres.RawBool("service_date < NOW()"),
		EquipmentServices.IsCompleted.EQ(Bool(false)),
		ShopMembers.UserID.EQ(String(user.UserID)),
	}

	if equipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*equipmentID)))
	}

	stmt := SELECT(
		EquipmentServices.AllColumns,
		Raw("EXTRACT(DAY FROM NOW() - service_date)").AS("days_overdue"),
	).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC()).
		LIMIT(int64(limit))

	var results []struct {
		model.EquipmentServices
		DaysOverdue int `sql:"days_overdue"`
	}

	err := stmt.Query(repo.Db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue services: %w", err)
	}

	overdueServices := make([]response.OverdueServiceResponse, len(results))
	for i, result := range results {
		// Get current username dynamically for each service
		username, err := repo.GetUsernameByUserID(result.CreatedBy)
		if err != nil {
			username = "Unknown User"
		}

		overdueServices[i] = response.OverdueServiceResponse{
			EquipmentServiceResponse: response.EquipmentServiceResponse{
				ID:                result.ID,
				ShopID:            result.ShopID,
				EquipmentID:       result.EquipmentID,
				ListID:            result.ListID,
				Description:       result.Description,
				ServiceType:       result.ServiceType,
				CreatedBy:         result.CreatedBy,
				CreatedByUsername: username,
				IsCompleted:       result.IsCompleted,
				CreatedAt:         result.CreatedAt,
				UpdatedAt:         result.UpdatedAt,
				ServiceDate:       result.ServiceDate,
				ServiceHours:      result.ServiceHours,
				CompletionDate:    result.CompletionDate,
			},
			DaysOverdue: result.DaysOverdue,
		}
	}

	return overdueServices, nil
}

// GetServicesDueSoon retrieves services that are due soon
func (repo *EquipmentServicesRepositoryImpl) GetServicesDueSoon(user *bootstrap.User, shopID string, daysAhead int, equipmentID *string, limit int) ([]response.DueSoonServiceResponse, error) {
	futureDate := time.Now().AddDate(0, 0, daysAhead)

	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		EquipmentServices.ServiceDate.IS_NOT_NULL(),
		postgres.RawBool("service_date > NOW()"),
		EquipmentServices.ServiceDate.LT_EQ(TimestampzT(futureDate)),
		EquipmentServices.IsCompleted.EQ(Bool(false)),
		ShopMembers.UserID.EQ(String(user.UserID)),
	}

	if equipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*equipmentID)))
	}

	stmt := SELECT(
		EquipmentServices.AllColumns,
		Raw("EXTRACT(DAY FROM service_date - NOW())").AS("days_until_due"),
	).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC()).
		LIMIT(int64(limit))

	var results []struct {
		model.EquipmentServices
		DaysUntilDue int `sql:"days_until_due"`
	}

	err := stmt.Query(repo.Db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get due soon services: %w", err)
	}

	dueSoonServices := make([]response.DueSoonServiceResponse, len(results))
	for i, result := range results {
		// Get current username dynamically for each service
		username, err := repo.GetUsernameByUserID(result.CreatedBy)
		if err != nil {
			username = "Unknown User"
		}

		dueSoonServices[i] = response.DueSoonServiceResponse{
			EquipmentServiceResponse: response.EquipmentServiceResponse{
				ID:                result.ID,
				ShopID:            result.ShopID,
				EquipmentID:       result.EquipmentID,
				ListID:            result.ListID,
				Description:       result.Description,
				ServiceType:       result.ServiceType,
				CreatedBy:         result.CreatedBy,
				CreatedByUsername: username,
				IsCompleted:       result.IsCompleted,
				CreatedAt:         result.CreatedAt,
				UpdatedAt:         result.UpdatedAt,
				ServiceDate:       result.ServiceDate,
				ServiceHours:      result.ServiceHours,
				CompletionDate:    result.CompletionDate,
			},
			DaysUntilDue: result.DaysUntilDue,
		}
	}

	return dueSoonServices, nil
}

// CompleteEquipmentService marks a service as completed with optional completion date
func (repo *EquipmentServicesRepositoryImpl) CompleteEquipmentService(user *bootstrap.User, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error) {
	now := time.Now()

	// Use current time if no completion date provided
	if completionDate == nil {
		completionDate = &now
	}

	stmt := EquipmentServices.UPDATE(
		EquipmentServices.IsCompleted,
		EquipmentServices.CompletionDate,
		EquipmentServices.UpdatedAt,
	).SET(
		EquipmentServices.IsCompleted.SET(Bool(true)),
		EquipmentServices.CompletionDate.SET(TimestampzT(*completionDate)),
		EquipmentServices.UpdatedAt.SET(TimestampzT(now)),
	).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.ShopID.IN(
				SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
			)),
	).RETURNING(EquipmentServices.AllColumns)

	var completedService model.EquipmentServices
	err := stmt.Query(repo.Db, &completedService)
	if err != nil {
		return nil, fmt.Errorf("failed to complete equipment service: %w", err)
	}

	slog.Info("Equipment service completed", "service_id", serviceID, "completed_by", user.UserID)
	return &completedService, nil
}

// ValidateServiceOwnership checks if the user owns the service
func (repo *EquipmentServicesRepositoryImpl) ValidateServiceOwnership(user *bootstrap.User, serviceID string) (bool, error) {
	stmt := SELECT(COUNT(STAR)).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.CreatedBy.EQ(String(user.UserID))),
	)

	var count int64
	err := stmt.Query(repo.Db, &count)
	if err != nil {
		return false, fmt.Errorf("failed to validate service ownership: %w", err)
	}

	return count > 0, nil
}

// ValidateEquipmentAccess checks if the user has access to the equipment and returns shopID
func (repo *EquipmentServicesRepositoryImpl) ValidateEquipmentAccess(user *bootstrap.User, equipmentID string) (string, error) {
	stmt := SELECT(ShopVehicle.ShopID).FROM(
		ShopVehicle.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
	).WHERE(
		ShopVehicle.ID.EQ(String(equipmentID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var result ShopIDResult
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return "", fmt.Errorf("equipment not found or access denied: %w", err)
	}

	return result.ShopID, nil
}

// ValidateListAccess checks if the user has access to the list and returns shopID
func (repo *EquipmentServicesRepositoryImpl) ValidateListAccess(user *bootstrap.User, listID string) (string, error) {
	stmt := SELECT(ShopLists.ShopID).FROM(
		ShopLists.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopLists.ShopID)),
	).WHERE(
		ShopLists.ID.EQ(String(listID)).
			AND(ShopMembers.UserID.EQ(String(user.UserID))),
	)

	var result ListShopIDResult
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return "", fmt.Errorf("list not found or access denied: %w", err)
	}

	return result.ShopID, nil
}

// GetUsernameByUserID dynamically fetches username from users table
func (repo *EquipmentServicesRepositoryImpl) GetUsernameByUserID(userID string) (string, error) {
	stmt := SELECT(Users.Username).FROM(Users).WHERE(Users.UID.EQ(String(userID)))

	var result UsernameResult
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		slog.Warn("Failed to get username for user", "user_id", userID, "error", err)
		return "Unknown User", nil // Return fallback instead of error
	}

	if result.Username == "" {
		return "Unknown User", nil
	}

	return result.Username, nil
}

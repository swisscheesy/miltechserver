package shared

import (
	"context"
	"database/sql"
	"log/slog"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"

	. "github.com/go-jet/jet/v2/postgres"
)

type UsernameResolver interface {
	GetUsernameByUserID(ctx context.Context, userID string) (string, error)
}

type UsernameRepository struct {
	db *sql.DB
}

type usernameResult struct {
	Username string `alias:"users.username"`
}

func NewUsernameRepository(db *sql.DB) *UsernameRepository {
	return &UsernameRepository{db: db}
}

func (repo *UsernameRepository) GetUsernameByUserID(ctx context.Context, userID string) (string, error) {
	stmt := SELECT(Users.Username).FROM(Users).WHERE(Users.UID.EQ(String(userID)))

	var result usernameResult
	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		slog.Warn("Failed to get username for user", "user_id", userID, "error", err)
		return "Unknown User", nil
	}

	if result.Username == "" {
		return "Unknown User", nil
	}

	return result.Username, nil
}

type UsernameCache struct {
	resolver UsernameResolver
	cache    map[string]string
}

func NewUsernameCache(resolver UsernameResolver) *UsernameCache {
	return &UsernameCache{
		resolver: resolver,
		cache:    make(map[string]string),
	}
}

func (cache *UsernameCache) GetUsernameByUserID(ctx context.Context, userID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if userID == "" {
		return "Unknown User", nil
	}

	if username, ok := cache.cache[userID]; ok {
		return username, nil
	}

	username, err := cache.resolver.GetUsernameByUserID(ctx, userID)
	if err != nil {
		return username, err
	}
	if username == "" {
		username = "Unknown User"
	}

	cache.cache[userID] = username
	return username, err
}

func MapServiceToResponse(svc model.EquipmentServices, username string) response.EquipmentServiceResponse {
	return response.EquipmentServiceResponse{
		ID:                svc.ID,
		ShopID:            svc.ShopID,
		EquipmentID:       svc.EquipmentID,
		ListID:            svc.ListID,
		Description:       svc.Description,
		ServiceType:       svc.ServiceType,
		CreatedBy:         svc.CreatedBy,
		CreatedByUsername: username,
		IsCompleted:       svc.IsCompleted,
		CreatedAt:         svc.CreatedAt,
		UpdatedAt:         svc.UpdatedAt,
		ServiceDate:       svc.ServiceDate,
		ServiceHours:      svc.ServiceHours,
		CompletionDate:    svc.CompletionDate,
	}
}

func MapServicesToResponses(ctx context.Context, services []model.EquipmentServices, resolver UsernameResolver) ([]response.EquipmentServiceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	responses := make([]response.EquipmentServiceResponse, len(services))
	for i, svc := range services {
		username, err := resolver.GetUsernameByUserID(ctx, svc.CreatedBy)
		if err != nil {
			return nil, err
		}
		if username == "" {
			username = "Unknown User"
		}
		responses[i] = MapServiceToResponse(svc, username)
	}
	return responses, nil
}

package aggregates

import (
	"context"
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetListsWithItems(context.Context, *bootstrap.User, string) ([]response.ShopListWithItems, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleByIDForMember(context.Context, *bootstrap.User, string) (*model.ShopVehicle, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleNotificationsWithItems(context.Context, string) ([]response.VehicleNotificationWithItems, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleRecentChanges(context.Context, string, int) ([]response.NotificationChangeWithUsername, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetVehicleServices(context.Context, string, int) ([]response.EquipmentServiceResponse, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetShopSnapshot(context.Context, *bootstrap.User, string, ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	return nil, ErrAggregateUnavailable
}

func (repo *RepositoryImpl) GetBootstrap(context.Context, *bootstrap.User, BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
	return nil, ErrAggregateUnavailable
}

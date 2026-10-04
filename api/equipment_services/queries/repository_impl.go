package queries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/equipment_services/shared"
	"miltechserver/api/request"
	shareddb "miltechserver/api/shared/db"
	"miltechserver/bootstrap"
)

type RepositoryImpl struct{ db *sql.DB }

func NewRepository(db *sql.DB) *RepositoryImpl { return &RepositoryImpl{db: db} }

func (repo *RepositoryImpl) GetByShop(ctx context.Context, user *bootstrap.User, shopID string, req request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error) {
	return repo.getPage(ctx, user, EquipmentServices.ShopID.EQ(String(shopID)), req, EquipmentServices.CreatedAt.DESC().NULLS_FIRST())
}
func (repo *RepositoryImpl) GetByEquipment(ctx context.Context, user *bootstrap.User, equipmentID string, req request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error) {
	return repo.getPage(ctx, user, EquipmentServices.EquipmentID.EQ(String(equipmentID)), req, EquipmentServices.ServiceDate.DESC().NULLS_FIRST())
}

func (repo *RepositoryImpl) getPage(ctx context.Context, user *bootstrap.User, scope postgres.BoolExpression, req request.GetEquipmentServicesRequest, order postgres.OrderByClause) ([]model.EquipmentServices, int64, error) {
	if user == nil {
		return nil, 0, shared.ErrUnauthorizedUser
	}
	filters, err := shared.ServiceFiltersFromRequest(req)
	if err != nil {
		return nil, 0, err
	}
	var services []model.EquipmentServices
	var total int64
	// Both statements observe one committed version, including membership. Offset
	// pages from later requests can still shift after inserts or deletions.
	err = shareddb.WithTxOptions(ctx, repo.db, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(tx *sql.Tx) error {
		predicate, err := shared.ServiceFilterPredicate(filters, time.Now())
		if err != nil {
			return err
		}
		predicate = scope.AND(predicate)
		source := EquipmentServices.INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID).AND(ShopMembers.UserID.EQ(String(user.UserID))))
		countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(source).WHERE(predicate)
		countQuery, countArgs := countStmt.Sql()
		if err := tx.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("failed to count services: %w", err)
		}
		dataStmt := SELECT(EquipmentServices.AllColumns).FROM(source).WHERE(predicate).ORDER_BY(order, EquipmentServices.ID.ASC()).LIMIT(int64(req.Limit)).OFFSET(int64(req.Offset))
		if err := dataStmt.QueryContext(ctx, tx, &services); err != nil {
			return fmt.Errorf("failed to get services: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return services, total, nil
}

var _ Repository = (*RepositoryImpl)(nil)

package aggregates

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

func (repo *RepositoryImpl) GetListsWithItems(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopListWithItems, error) {
	const query = `
SELECT
	l.id, l.shop_id, l.created_by, creator.username, l.description, l.created_at, l.updated_at,
	i.id, i.list_id, i.niin, i.nomenclature, i.quantity, i.added_by, added.username,
	i.created_at, i.updated_at, i.nickname, i.unit_of_measure
FROM shop_lists l
INNER JOIN shop_members sm ON sm.shop_id = l.shop_id AND sm.user_id = $2
LEFT JOIN users creator ON creator.uid = l.created_by
LEFT JOIN shop_list_items i ON i.list_id = l.id
LEFT JOIN users added ON added.uid = i.added_by
WHERE l.shop_id = $1
ORDER BY l.created_at DESC, l.id ASC, i.created_at ASC, i.id ASC`

	rows, err := repo.db.QueryContext(ctx, query, shopID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shop lists with items: %w", err)
	}
	defer rows.Close()

	lists := []response.ShopListWithItems{}
	listIndexes := make(map[string]int)

	for rows.Next() {
		var (
			listID            string
			listShopID        string
			listCreatedBy     string
			listCreatorName   sql.NullString
			listDescription   string
			listCreatedAt     time.Time
			listUpdatedAt     time.Time
			itemID            sql.NullString
			itemListID        sql.NullString
			itemNiin          sql.NullString
			itemNomenclature  sql.NullString
			itemQuantity      sql.NullInt64
			itemAddedBy       sql.NullString
			itemAddedByName   sql.NullString
			itemCreatedAt     sql.NullTime
			itemUpdatedAt     sql.NullTime
			itemNickname      sql.NullString
			itemUnitOfMeasure sql.NullString
		)

		err := rows.Scan(
			&listID,
			&listShopID,
			&listCreatedBy,
			&listCreatorName,
			&listDescription,
			&listCreatedAt,
			&listUpdatedAt,
			&itemID,
			&itemListID,
			&itemNiin,
			&itemNomenclature,
			&itemQuantity,
			&itemAddedBy,
			&itemAddedByName,
			&itemCreatedAt,
			&itemUpdatedAt,
			&itemNickname,
			&itemUnitOfMeasure,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop list item aggregate row: %w", err)
		}

		listIndex, ok := listIndexes[listID]
		if !ok {
			lists = append(lists, response.ShopListWithItems{
				ShopListWithUsername: response.ShopListWithUsername{
					ID:                listID,
					ShopID:            listShopID,
					CreatedBy:         listCreatedBy,
					CreatedByUsername: nullStringPtr(listCreatorName),
					Description:       listDescription,
					CreatedAt:         timePtr(listCreatedAt),
					UpdatedAt:         timePtr(listUpdatedAt),
				},
				Items: []response.ShopListItemWithUsername{},
			})
			listIndex = len(lists) - 1
			listIndexes[listID] = listIndex
		}

		if itemID.Valid {
			lists[listIndex].Items = append(lists[listIndex].Items, response.ShopListItemWithUsername{
				ID:              itemID.String,
				ListID:          itemListID.String,
				Niin:            itemNiin.String,
				Nomenclature:    itemNomenclature.String,
				Quantity:        int32(itemQuantity.Int64),
				AddedBy:         itemAddedBy.String,
				AddedByUsername: nullStringPtr(itemAddedByName),
				CreatedAt:       nullTimePtr(itemCreatedAt),
				UpdatedAt:       nullTimePtr(itemUpdatedAt),
				Nickname:        nullStringPtr(itemNickname),
				UnitOfMeasure:   nullStringPtr(itemUnitOfMeasure),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shop lists with items: %w", err)
	}

	return lists, nil
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

func timePtr(value time.Time) *time.Time {
	return &value
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

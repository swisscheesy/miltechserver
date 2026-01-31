package quick_lists

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetQuickListClothing() (QuickListsClothingResponse, error) {
	var clothingData []model.QuickListClothing
	stmt := SELECT(
		table.QuickListClothing.AllColumns,
	).FROM(table.QuickListClothing)

	if err := stmt.Query(repo.db, &clothingData); err != nil {
		return QuickListsClothingResponse{}, err
	}

	return QuickListsClothingResponse{
		Clothing: clothingData,
		Count:    len(clothingData),
	}, nil
}

func (repo *RepositoryImpl) GetQuickListWheels() (QuickListsWheelsResponse, error) {
	var wheelsData []model.QuickListWheelTires
	stmt := SELECT(
		table.QuickListWheelTires.AllColumns,
	).FROM(table.QuickListWheelTires)

	if err := stmt.Query(repo.db, &wheelsData); err != nil {
		return QuickListsWheelsResponse{}, err
	}

	return QuickListsWheelsResponse{
		Wheels: wheelsData,
		Count:  len(wheelsData),
	}, nil
}

func (repo *RepositoryImpl) GetQuickListBatteries() (QuickListsBatteryResponse, error) {
	var batteriesData []model.QuickListBattery
	stmt := SELECT(
		table.QuickListBattery.AllColumns,
	).FROM(table.QuickListBattery)

	if err := stmt.Query(repo.db, &batteriesData); err != nil {
		return QuickListsBatteryResponse{}, err
	}

	return QuickListsBatteryResponse{
		Batteries: batteriesData,
		Count:     len(batteriesData),
	}, nil
}

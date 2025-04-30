package repository

import (
	"database/sql"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"

	. "github.com/go-jet/jet/v2/postgres"
)

type ItemQuickListsRepositoryImpl struct {
	db *sql.DB
}

func NewItemQuickListsRepositoryImpl(db *sql.DB) *ItemQuickListsRepositoryImpl {
	return &ItemQuickListsRepositoryImpl{db: db}
}

func (repo *ItemQuickListsRepositoryImpl) GetQuickListClothing() (response.QuickListsClothingResponse, error) {

	var clothingData []model.QuickListClothing
	stmt := SELECT(
		table.QuickListClothing.AllColumns,
	).FROM(table.QuickListClothing)

	err := stmt.Query(repo.db, &clothingData)
	if err != nil {
		return response.QuickListsClothingResponse{}, err
	}

	return response.QuickListsClothingResponse{
		Clothing: clothingData,
		Count:    len(clothingData),
	}, nil
}

func (repo *ItemQuickListsRepositoryImpl) GetQuickListWheels() (response.QuickListsWheelsResponse, error) {

	var wheelsData []model.QuickListWheelTires
	stmt := SELECT(
		table.QuickListWheelTires.AllColumns,
	).FROM(table.QuickListWheelTires)

	err := stmt.Query(repo.db, &wheelsData)
	if err != nil {
		return response.QuickListsWheelsResponse{}, err
	}

	return response.QuickListsWheelsResponse{
		Wheels: wheelsData,
		Count:  len(wheelsData),
	}, nil

}

func (repo *ItemQuickListsRepositoryImpl) GetQuickListBatteries() (response.QuickListsBatteryResponse, error) {

	var batteriesData []model.QuickListBattery
	stmt := SELECT(
		table.QuickListBattery.AllColumns,
	).FROM(table.QuickListBattery)

	err := stmt.Query(repo.db, &batteriesData)
	if err != nil {
		return response.QuickListsBatteryResponse{}, err
	}

	return response.QuickListsBatteryResponse{
		Batteries: batteriesData,
		Count:     len(batteriesData),
	}, nil
}

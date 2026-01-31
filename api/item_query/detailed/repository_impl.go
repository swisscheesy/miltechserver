package detailed

import (
	"database/sql"
	"log/slog"

	"miltechserver/api/item_query/detailed/queries"
	"miltechserver/api/response"
)

type RepositoryImpl struct {
	Db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{Db: db}
}

func (repo *RepositoryImpl) GetDetailedItemData(niin string) (response.DetailedResponse, error) {
	amdfData, err := queries.GetAmdfData(repo.Db, niin)
	repo.logQueryError("amdf", niin, err)

	armyPackData, err := queries.GetArmyPackagingAndFreight(repo.Db, niin)
	repo.logQueryError("army_packaging", niin, err)

	sarsscatData, err := queries.GetSarsscat(repo.Db, niin)
	repo.logQueryError("sarsscat", niin, err)

	identificationData, err := queries.GetIdentification(repo.Db, niin)
	repo.logQueryError("identification", niin, err)

	managementData, err := queries.GetManagement(repo.Db, niin)
	repo.logQueryError("management", niin, err)

	referenceData, err := queries.GetReference(repo.Db, niin)
	repo.logQueryError("reference", niin, err)

	freightData, err := queries.GetFreight(repo.Db, niin)
	repo.logQueryError("freight", niin, err)

	packagingData, err := queries.GetPackaging(repo.Db, niin)
	repo.logQueryError("packaging", niin, err)

	characteristicsData, err := queries.GetCharacteristics(repo.Db, niin)
	repo.logQueryError("characteristics", niin, err)

	dispositionData, err := queries.GetDisposition(repo.Db, niin)
	repo.logQueryError("disposition", niin, err)

	fullDetailedItem := response.DetailedResponse{
		Amdf:                    amdfData,
		ArmyPackagingAndFreight: armyPackData,
		Sarsscat:                sarsscatData,
		Identification:          identificationData,
		Management:              managementData,
		Reference:               referenceData,
		Freight:                 freightData,
		Packaging:               packagingData,
		Characteristics:         characteristicsData,
		Disposition:             dispositionData,
	}

	return fullDetailedItem, nil
}

func (repo *RepositoryImpl) logQueryError(source string, niin string, err error) {
	if err == nil {
		return
	}
	slog.Error("Detailed item query failed", "source", source, "niin", niin, "error", err)
}

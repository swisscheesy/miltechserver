package repository

import (
	"github.com/gin-gonic/gin"
	"math"
	"miltechserver/model"
	"miltechserver/prisma/db"
	"strconv"
	"strings"
)

var returnCount = 20

type ItemLokupRepositoryImpl struct {
	Db *db.PrismaClient
}

func NewItemLookupRepositoryImpl(db *db.PrismaClient) *ItemLokupRepositoryImpl {
	return &ItemLokupRepositoryImpl{Db: db}
}

// SearchLINByPage searches for LIN (Line Item Number) by page.
// \param ctx - the context for the request.
// \param page - the page number to retrieve.
// \return a LINPageResponse containing the LIN data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchLINByPage(ctx *gin.Context, page int) (model.LINPageResponse, error) {

	linData, _ := repo.Db.ArmyLineItemNumber.
		FindMany().
		Take(returnCount).
		Skip(returnCount * (page - 1)).
		Exec(ctx)

	var res []struct {
		Count db.RawString
	}
	// TODO: Cache this count to avoid querying the database every time
	// Also check error properly
	_ = repo.Db.Prisma.QueryRaw("SELECT COUNT(*) FROM public.army_line_item_number").Exec(ctx, &res)

	// TODO Handle this error properly as well
	count, _ := strconv.Atoi(string(res[0].Count))

	totalPages := math.Ceil(float64(count / 20))

	return model.LINPageResponse{
		Lins:       linData,
		Count:      count,
		Page:       page,
		TotalPages: int(totalPages),
		IsLastPage: page == int(totalPages),
	}, nil

}

func (repo *ItemLokupRepositoryImpl) SearchLINByNIIN(ctx *gin.Context, niin string) ([]db.LookupLinNiinModel, error) {
	linData, _ := repo.Db.LookupLinNiin.FindMany(db.LookupLinNiin.Niin.Contains(niin)).Exec(ctx)

	return linData, nil
}

func (repo *ItemLokupRepositoryImpl) SearchNIINByLIN(ctx *gin.Context, lin string) ([]db.LookupLinNiinModel, error) {
	linData, _ := repo.Db.LookupLinNiin.FindMany(db.LookupLinNiin.Lin.Contains(lin)).Exec(ctx)

	return linData, nil
}

func (repo *ItemLokupRepositoryImpl) SearchUOCByPage(ctx *gin.Context, page int) (model.UOCPageResponse, error) {
	uocData, _ := repo.Db.LookupUoc.
		FindMany().
		Take(returnCount).
		Skip(returnCount * (page - 1)).
		Exec(ctx)

	var res []struct {
		Count db.RawString
	}
	// TODO: Cache this count to avoid querying the database every time
	// Also check error properly
	_ = repo.Db.Prisma.QueryRaw("SELECT COUNT(*) FROM public.lookup_uoc").Exec(ctx, &res)

	// TODO Handle this error properly as well
	count, _ := strconv.Atoi(string(res[0].Count))

	totalPages := math.Ceil(float64(count / 20))

	return model.UOCPageResponse{
		UOCs:       uocData,
		Count:      count,
		Page:       page,
		TotalPages: int(totalPages),
		IsLastPage: page == int(totalPages),
	}, nil
}

func (repo *ItemLokupRepositoryImpl) SearchSpecificUOC(ctx *gin.Context, uoc string) ([]db.LookupUocModel, error) {
	uocData, _ := repo.Db.LookupUoc.FindMany(db.LookupUoc.Uoc.Contains(strings.ToUpper(uoc))).Exec(ctx)

	return uocData, nil
}

func (repo *ItemLokupRepositoryImpl) SearchUOCByModel(ctx *gin.Context, model string) ([]db.LookupUocModel, error) {
	uocData, _ := repo.Db.LookupUoc.FindMany(db.LookupUoc.Model.Contains(strings.ToUpper(model))).Exec(ctx)

	return uocData, nil
}

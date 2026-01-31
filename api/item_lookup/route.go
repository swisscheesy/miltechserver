package item_lookup

import (
	"database/sql"

	"miltechserver/api/item_lookup/cage"
	"miltechserver/api/item_lookup/lin"
	"miltechserver/api/item_lookup/substitute"
	"miltechserver/api/item_lookup/uoc"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB *sql.DB
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	linRepo := lin.NewRepository(deps.DB)
	linService := lin.NewService(linRepo)
	lin.RegisterRoutes(router, linService)

	uocRepo := uoc.NewRepository(deps.DB)
	uocService := uoc.NewService(uocRepo)
	uoc.RegisterRoutes(router, uocService)

	cageRepo := cage.NewRepository(deps.DB)
	cageService := cage.NewService(cageRepo)
	cage.RegisterRoutes(router, cageService)

	substituteRepo := substitute.NewRepository(deps.DB)
	substituteService := substitute.NewService(substituteRepo)
	substitute.RegisterRoutes(router, substituteService)
}

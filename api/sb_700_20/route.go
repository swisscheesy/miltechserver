package sb_700_20

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	DB *sql.DB
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo)
	RegisterHandlers(router, svc)
}

func RegisterHandlers(router *gin.RouterGroup, svc Service) {
	h := Handler{service: svc}
	router.GET("/sb700-20/app-b/list", h.listAppB)
	router.GET("/sb700-20/app-b/search/:lin", h.searchAppB)
	router.GET("/sb700-20/app-c/list", h.listAppC)
	router.GET("/sb700-20/app-c/search/:lin", h.searchAppC)
	router.GET("/sb700-20/app-d/list", h.listAppD)
	router.GET("/sb700-20/app-d/search/:lin", h.searchAppD)
	router.GET("/sb700-20/app-e/list", h.listAppE)
	router.GET("/sb700-20/app-e/search/:lin", h.searchAppE)
	router.GET("/sb700-20/app-f/list", h.listAppF)
	router.GET("/sb700-20/app-f/search/:lin", h.searchAppF)
	router.GET("/sb700-20/app-g/list", h.listAppG)
	router.GET("/sb700-20/app-g/search/:lin", h.searchAppG)
	router.GET("/sb700-20/app-h1/list", h.listAppH1)
	router.GET("/sb700-20/app-h1/search/:lin", h.searchAppH1)
	router.GET("/sb700-20/app-h2/list", h.listAppH2)
	router.GET("/sb700-20/app-h2/search/:lin", h.searchAppH2)
	router.GET("/sb700-20/app-i/list", h.listAppI)
	router.GET("/sb700-20/app-i/search/:lin", h.searchAppI)
	router.GET("/sb700-20/app-j/list", h.listAppJ)
	router.GET("/sb700-20/app-j/search/:lin", h.searchAppJ)
	router.GET("/sb700-20/chp-4/list", h.listChp4)
	router.GET("/sb700-20/chp-4/search/:lin", h.searchChp4)
	router.GET("/sb700-20/chp-6/list", h.listChp6)
	router.GET("/sb700-20/chp-6/search/:lin", h.searchChp6)
	router.GET("/sb700-20/chp-8/list", h.listChp8)
	router.GET("/sb700-20/chp-8/search/:lin", h.searchChp8)
	router.GET("/sb700-20/app-e/search-new-lin/:new_lin", h.searchAppEByNewLIN)
	router.GET("/sb700-20/app-g/search-new-lin/:new_lin", h.searchAppGByNewLIN)
	router.GET("/sb700-20/app-h1/search-sublin/:sublin", h.searchAppH1BySubLIN)
	router.GET("/sb700-20/app-h2/search-sublin/:sublin", h.searchAppH2BySubLIN)
	router.GET("/sb700-20/chp-4/search-ric/:ric", h.searchChp4ByRIC)
	router.GET("/sb700-20/chp-6/search-ric/:ric", h.searchChp6ByRIC)
	router.GET("/sb700-20/chp-8/search-ric/:ric", h.searchChp8ByRIC)
}

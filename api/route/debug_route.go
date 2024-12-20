package route

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"miltechserver/bootstrap"
	"net/http"
	"time"
)

func NewDebugRouter(env *bootstrap.Env, timeout time.Duration, db *sql.DB, group *gin.RouterGroup) {
	// All Public Routes

	group.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "user")
		ctx.JSON(200, gin.H{
			"message": "pong",
		})
	})

	//group.GET("/ping", func(ctx *gin.Context) {
	//	test, err := db.Nsn.FindFirst(db.Nsn.Niin.Equals("013469317")).Exec()
	//	if err != nil {
	//		ctx.JSON(200, gin.H{
	//			"message": "error",
	//			"data":    err,
	//		})
	//		return
	//	}
	//	ctx.JSON(200, gin.H{
	//		"message": "pong",
	//		"data":    test,
	//	})
	//})
}

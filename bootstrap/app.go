package bootstrap

import (
	"miltechserver/prisma/db"
)

type Application struct {
	Env        *Env
	PostgresDB *db.PrismaClient
}

func App() Application {
	app := &Application{}
	app.Env = NewEnv()
	app.PostgresDB = NewPrismaClient(app.Env)
	//defer func() {
	//	if err := app.PostgresDB.Disconnect(); err != nil {
	//		log.Fatalf("Unable to disconnect from database: %s", err)
	//	}
	//}()
	return *app
}

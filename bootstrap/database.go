package bootstrap

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log/slog"
	"miltechserver/helper"
)

func NewSqlClient(env *Env) *sql.DB {
	dsnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", env.Host, env.Port, env.Username, env.Password, env.DBName)
	slog.Info("Connecting to Database")
	db, err := sql.Open("postgres", dsnStr)
	helper.PanicOnError(err)

	err = db.Ping()

	if err != nil {
		slog.Error("Unable to connect to database: %s", err)
		panic(nil)
	}

	slog.Info("Connected to Database!")

	return db
}

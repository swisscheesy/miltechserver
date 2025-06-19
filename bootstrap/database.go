package bootstrap

import (
	"database/sql"
	"fmt"
	"log/slog"
	"miltechserver/helper"

	_ "github.com/lib/pq"
)

func NewSqlClient(env *Env) *sql.DB {
	dsnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", env.Host, env.Port, env.Username, env.Password, env.DBName, env.SslMode)
	slog.Info("Connecting to Database")
	db, err := sql.Open("postgres", dsnStr)
	helper.PanicOnError(err)

	err = db.Ping()

	if err != nil {
		slog.Error("Unable to connect to database", "error", err)
		panic(err)
	}

	slog.Info("Connected to Database!")

	return db
}

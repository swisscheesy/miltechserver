package bootstrap

import (
	"database/sql"
	"fmt"
	"log/slog"
	"miltechserver/helper"
	"time"

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

	// Configure connection pool for parallel query workloads
	db.SetMaxOpenConns(env.DBMaxOpenConns)
	db.SetMaxIdleConns(env.DBMaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	slog.Info("Connection pool configured",
		"maxOpenConns", env.DBMaxOpenConns,
		"maxIdleConns", env.DBMaxIdleConns,
		"connMaxLifetime", "5m",
		"connMaxIdleTime", "1m")

	return db
}

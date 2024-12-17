package bootstrap

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
)

func NewGormClient(env *Env) *gorm.DB {
	dsnStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Phoenix", env.Host, env.Username, env.Password, env.DBName, env.Port)
	db, err := gorm.Open(postgres.Open(dsnStr), &gorm.Config{})
	slog.Info("Connecting to Database")

	if err != nil {
		slog.Error("Unable to connect to database: %s", err)
		panic(nil)
	}

	slog.Info("Connected to Database!")

	return db
}

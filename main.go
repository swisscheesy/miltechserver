package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"miltechserver/api/route"
	"miltechserver/api/shops"
	"miltechserver/bootstrap"
	"miltechserver/helper"
	"miltechserver/internal/jetgen"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	// Start the engine
	engine := SetupEngine()
	err := engine.Run(":8080")
	helper.PanicOnError(err)

}

func SetupEngine() *gin.Engine {
	ctx := context.Background()
	env := bootstrap.NewEnv()
	var app bootstrap.Application
	server, err := setupEngine(ctx, env, jetgen.Generate, func(ctx context.Context, env *bootstrap.Env) (bootstrap.Application, error) {
		var err error
		app = bootstrap.App(ctx, env)
		return app, err
	})
	if err != nil {
		log.Fatal(err)
	}

	worker, err := shops.NewCleanupWorker(shops.Dependencies{DB: app.Db, BlobClient: app.BlobClient, Env: env})
	if err != nil {
		log.Fatal("Unable to initialize image cleanup worker")
	}
	workerCtx, cancelWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		if err := worker.Run(workerCtx); err != nil {
			log.Println("Image cleanup worker stopped")
		}
	}()

	// Cleanup server on crash or interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		if err := stopCleanupBeforeDatabase(cancelWorker, workerDone, app.Db.Close); err != nil {
			log.Fatal("Unable to disconnect from database")
		}
		log.Println("Disconnected from database")
		os.Exit(1)
	}()

	return server
}

// setupEngine prevents route registration until generation has completed against the application pool.
func setupEngine(ctx context.Context, env *bootstrap.Env, generate func(*sql.DB, jetgen.Config) error, buildApplication func(context.Context, *bootstrap.Env) (bootstrap.Application, error)) (*gin.Engine, error) {
	if env == nil || env.DBSchema != "public" || generate == nil || buildApplication == nil {
		return nil, errors.New("invalid startup configuration")
	}
	app, err := buildApplication(ctx, env)
	if err != nil {
		return nil, errors.New("unable to initialize application")
	}
	if app.Db == nil {
		return nil, errors.New("application database unavailable")
	}
	if err := jetgen.ValidateStartupSchema(ctx, app.Db); err != nil {
		if closeErr := app.Db.Close(); closeErr != nil {
			return nil, errors.New("required startup schema unavailable; database cleanup failed")
		}
		return nil, errors.New("required startup schema unavailable")
	}
	if err := generate(app.Db, jetgen.Config{Schema: env.DBSchema, OutputDirectory: ".gen"}); err != nil {
		if err := app.Db.Close(); err != nil {
			return nil, errors.New("startup Jet generation failed; database cleanup failed")
		}
		return nil, errors.New("required startup Jet generation failed")
	}
	server := route.NewEngine()
	route.Setup(app.Db, server, app.FireAuth, env, app.BlobClient)
	return server, nil
}

func stopCleanupBeforeDatabase(cancel context.CancelFunc, done <-chan struct{}, closeDatabase func() error) error {
	cancel()
	<-done
	return closeDatabase()
}

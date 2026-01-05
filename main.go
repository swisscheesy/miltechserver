package main

import (
	"context"
	"fmt"
	"log"
	"miltechserver/api/route"
	"miltechserver/bootstrap"
	"miltechserver/helper"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/go-jet/jet/v2/generator/metadata"
	"github.com/go-jet/jet/v2/generator/postgres"
	"github.com/go-jet/jet/v2/generator/template"
	postgres2 "github.com/go-jet/jet/v2/postgres"
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
	generateSchema(env)
	app := bootstrap.App(ctx, env)
	db := app.Db

	server := gin.Default()

	route.Setup(db, server, app.FireAuth, env, app.BlobClient, app.BlobCredential, app.Hub)

	// Cleanup server on crash or interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		// Shutdown WebSocket Hub if it exists
		if app.Hub != nil {
			app.Hub.Shutdown()
			log.Println("WebSocket Hub shut down")
		}

		if err := db.Close(); err != nil {
			log.Fatalf("Unable to disconnect from database: %s", err)
		}
		log.Println("Disconnected from database")
		os.Exit(1)
	}()

	return server
}

func generateSchema(env *bootstrap.Env) {
	err1 := postgres.Generate(
		"./.gen",
		postgres.DBConnection{
			Host:       env.Host,
			Port:       5432,
			User:       env.Username,
			Password:   env.Password,
			SslMode:    env.SslMode,
			DBName:     env.DBName,
			SchemaName: env.DBSchema,
		},
		template.Default(postgres2.Dialect).
			UseSchema(func(schema metadata.Schema) template.Schema {
				return template.DefaultSchema(schema).
					UseModel(template.DefaultModel().
						UseTable(func(table metadata.Table) template.TableModel {
							return template.DefaultTableModel(table).
								UseField(func(columnMetaData metadata.Column) template.TableModelField {
									defaultTableModelField := template.DefaultTableModelField(columnMetaData)
									return defaultTableModelField.UseTags(
										fmt.Sprintf(`json:"%s"`, columnMetaData.Name),
									)
								})
						}).UseView(func(table metadata.Table) template.TableModel {
						return template.DefaultTableModel(table).
							UseField(func(columnMetaData metadata.Column) template.TableModelField {
								defaultTableModelField := template.DefaultTableModelField(columnMetaData)
								return defaultTableModelField.UseTags(
									fmt.Sprintf(`json:"%s"`, columnMetaData.Name),
								)
							})
					}),
					)
			}),
	)

	if err1 != nil {
		log.Fatalf("Error generating code: %s", err1)
	}
}

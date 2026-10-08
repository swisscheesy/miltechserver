// Command jetregen generates canonical models with snake_case JSON tags.
package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	"miltechserver/internal/jetgen"
	"os"
)

func main() {
	dsn := os.Getenv("JET_DSN")
	if dsn == "" {
		log.Fatal("JET_DSN is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("invalid Jet database configuration")
	}
	defer db.Close()
	schema := os.Getenv("JET_SCHEMA")
	if schema == "" {
		schema = "public"
	}
	output := os.Getenv("JET_OUTPUT_DIR")
	if output == "" {
		output = ".gen"
	}
	if err := jetgen.Generate(db, jetgen.Config{Schema: schema, OutputDirectory: output}); err != nil {
		log.Fatal(err)
	}
}

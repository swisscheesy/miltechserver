// Command jetregen regenerates .gen/ with snake_case json tags on every model
// field. The plain `jet` CLI omits json tags, and API responses marshal these
// models directly, so running the CLI would silently rename response keys
// (shop_id -> ShopID) for every client.
//
// Usage, from the repository root:
//
//	JET_DSN="postgresql://postgres:<password>@<host>:5432/miltech_ng?sslmode=disable" go run ./tools/jetregen
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-jet/jet/v2/generator/metadata"
	"github.com/go-jet/jet/v2/generator/postgres"
	"github.com/go-jet/jet/v2/generator/template"
	postgresdialect "github.com/go-jet/jet/v2/postgres"
	_ "github.com/lib/pq"
)

const (
	schemaName = "public"
	outputDir  = "./.gen"
)

func main() {
	dsn := os.Getenv("JET_DSN")
	if dsn == "" {
		log.Fatal("JET_DSN is required")
	}

	err := postgres.GenerateDSN(dsn, schemaName, outputDir, jsonTaggedTemplate())
	if err != nil {
		log.Fatalf("jet generation failed: %v", err)
	}
}

func jsonTaggedTemplate() template.Template {
	return template.Default(postgresdialect.Dialect).
		UseSchema(func(schema metadata.Schema) template.Schema {
			return template.DefaultSchema(schema).
				UseModel(template.DefaultModel().
					UseTable(jsonTaggedModel).
					UseView(jsonTaggedModel))
		})
}

func jsonTaggedModel(table metadata.Table) template.TableModel {
	return template.DefaultTableModel(table).
		UseField(func(column metadata.Column) template.TableModelField {
			return template.DefaultTableModelField(column).
				UseTags(fmt.Sprintf(`json:"%s"`, column.Name))
		})
}

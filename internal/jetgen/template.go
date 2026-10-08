package jetgen

import (
	"fmt"
	"github.com/go-jet/jet/v2/generator/metadata"
	"github.com/go-jet/jet/v2/generator/template"
	postgresdialect "github.com/go-jet/jet/v2/postgres"
)

func JSONTaggedTemplate() template.Template {
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

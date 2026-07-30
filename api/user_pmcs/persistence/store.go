package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const maxBindParameters = 1000

type Queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct {
	DB               *sql.DB
	MaxWriteAttempts int
}

func NewStore(database *sql.DB, maxWriteAttempts int) Store {
	return Store{
		DB:               database,
		MaxWriteAttempts: maxWriteAttempts,
	}
}

func insertRows(
	ctx context.Context,
	tx *sql.Tx,
	table string,
	columns []string,
	rows [][]any,
) error {
	batchSize := maxRowsPerBatch(len(columns))
	for start := 0; start < len(rows); start += batchSize {
		end := min(start+batchSize, len(rows))
		var query strings.Builder
		fmt.Fprintf(
			&query,
			"INSERT INTO %s (%s) VALUES ",
			table,
			strings.Join(columns, ", "),
		)
		arguments := make([]any, 0, (end-start)*len(columns))
		for rowIndex, row := range rows[start:end] {
			if len(row) != len(columns) {
				return fmt.Errorf(
					"insert %s row has %d values for %d columns",
					table,
					len(row),
					len(columns),
				)
			}
			if rowIndex > 0 {
				query.WriteString(", ")
			}
			query.WriteByte('(')
			for columnIndex, value := range row {
				if columnIndex > 0 {
					query.WriteString(", ")
				}
				arguments = append(arguments, value)
				fmt.Fprintf(&query, "$%d", len(arguments))
			}
			query.WriteByte(')')
		}
		if _, err := tx.ExecContext(ctx, query.String(), arguments...); err != nil {
			return fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return nil
}

func maxRowsPerBatch(columnsPerRow int) int {
	if columnsPerRow <= 0 {
		return 0
	}
	return max(1, maxBindParameters/columnsPerRow)
}

func inStatement(prefix string, ids []uuid.UUID) (string, []any) {
	placeholders, arguments := uuidPlaceholders(ids)
	return prefix + placeholders + ")", arguments
}

func uuidPlaceholders(ids []uuid.UUID) (string, []any) {
	placeholders := make([]string, len(ids))
	arguments := make([]any, len(ids))
	for index, id := range ids {
		placeholders[index] = fmt.Sprintf("$%d", index+1)
		arguments[index] = id
	}
	return strings.Join(placeholders, ", "), arguments
}

func closeRowsWithError(
	rows *sql.Rows,
	primaryError error,
	resource string,
) error {
	if closeError := rows.Close(); closeError != nil {
		return errors.Join(
			primaryError,
			fmt.Errorf("close %s rows: %w", resource, closeError),
		)
	}
	return primaryError
}

func queryRows(
	ctx context.Context,
	queryer Queryer,
	resource string,
	query string,
	argument any,
	scan func(*sql.Rows) error,
) error {
	rows, err := queryer.QueryContext(ctx, query, argument)
	if err != nil {
		return fmt.Errorf("query %s: %w", resource, err)
	}
	for rows.Next() {
		if err := scan(rows); err != nil {
			return closeRowsWithError(
				rows,
				fmt.Errorf("scan %s: %w", resource, err),
				resource,
			)
		}
	}
	if err := rows.Err(); err != nil {
		return closeRowsWithError(
			rows,
			fmt.Errorf("iterate %s: %w", resource, err),
			resource,
		)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close %s rows: %w", resource, err)
	}
	return nil
}

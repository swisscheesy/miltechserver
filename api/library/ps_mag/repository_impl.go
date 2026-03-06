package ps_mag

import (
	"database/sql"
	"fmt"
)

// RepositoryImpl implements Repository using a PostgreSQL database.
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository constructs a RepositoryImpl backed by the given *sql.DB.
func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

// SearchSummaries returns rows from ps_mag_summaries whose summary contains
// phrase (ILIKE, case-insensitive), paginated by page/pageSize.
// Also returns the total count of matching rows for pagination metadata.
func (r *RepositoryImpl) SearchSummaries(phrase string, page, pageSize int) ([]summaryRow, int, error) {
	pattern := "%" + phrase + "%"
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM ps_mag_summaries WHERE summary ILIKE $1`,
		pattern,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count summaries: %w", err)
	}

	rows, err := r.db.Query(
		`SELECT file_name, summary
		 FROM ps_mag_summaries
		 WHERE summary ILIKE $1
		 ORDER BY file_name ASC
		 LIMIT $2 OFFSET $3`,
		pattern, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search summaries query: %w", err)
	}
	defer rows.Close()

	var results []summaryRow
	for rows.Next() {
		var row summaryRow
		var summary sql.NullString
		if err := rows.Scan(&row.FileName, &summary); err != nil {
			return nil, 0, fmt.Errorf("scan summary row: %w", err)
		}
		if summary.Valid {
			row.Summary = summary.String
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate summary rows: %w", err)
	}

	return results, total, nil
}

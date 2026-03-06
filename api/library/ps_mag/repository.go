package ps_mag

// summaryRow holds a single row from ps_mag_summaries.
// It is an internal type used between the repository and service layers.
type summaryRow struct {
	FileName string
	Summary  string
}

// Repository provides data access for the ps_mag_summaries table.
type Repository interface {
	// SearchSummaries returns paginated rows from ps_mag_summaries whose summary
	// contains phrase (case-insensitive), plus the total count of matching rows.
	// page is 1-indexed. pageSize controls how many rows are returned.
	SearchSummaries(phrase string, page, pageSize int) ([]summaryRow, int, error)
}

package ps_mag

import "context"

// Service provides methods for accessing PS Magazine issues from Azure Blob Storage.
type Service interface {
	// ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	// page is 1-indexed. order must be "asc" or "desc".
	// year and issueNumber are optional — pass nil to skip the filter.
	ListIssues(ctx context.Context, page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error)

	// GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	// blobPath must start with "ps-mag/" and end with ".pdf".
	GenerateDownloadURL(ctx context.Context, blobPath string) (*DownloadURLResponse, error)

	// SearchSummaries returns a paginated list of PS Magazine issues whose summaries
	// contain query. Only the lines matching query are returned per file.
	// query must be at least 3 characters. page is 1-indexed.
	SearchSummaries(query string, page int) (*PSMagSearchResponse, error)
}

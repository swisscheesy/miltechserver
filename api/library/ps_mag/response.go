package ps_mag

// PSMagIssueResponse represents a single parsed PS Magazine issue.
type PSMagIssueResponse struct {
	Name         string `json:"name"`          // e.g. "PS_Magazine_Issue_495_February_1994.pdf"
	BlobPath     string `json:"blob_path"`     // e.g. "ps-mag/PS_Magazine_Issue_495_February_1994.pdf"
	IssueNumber  int    `json:"issue_number"`  // 495
	Month        string `json:"month"`         // "February"
	Year         int    `json:"year"`          // 1994
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"` // RFC3339
}

// PSMagIssuesResponse is the paginated listing response.
type PSMagIssuesResponse struct {
	Issues     []PSMagIssueResponse `json:"issues"`
	Count      int                  `json:"count"`       // items on this page
	TotalCount int                  `json:"total_count"` // total matching issues across all pages
	Page       int                  `json:"page"`
	TotalPages int                  `json:"total_pages"`
	Order      string               `json:"order"`
}

// DownloadURLResponse contains a time-limited SAS URL for a PS Magazine issue.
type DownloadURLResponse struct {
	BlobPath    string `json:"blob_path"`
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"` // RFC3339
}

// PSMagSearchResult is a single file with only its matching summary lines.
type PSMagSearchResult struct {
	FileName      string   `json:"file_name"`
	MatchingLines []string `json:"matching_lines"`
}

// PSMagSearchResponse is the paginated summary search response.
type PSMagSearchResponse struct {
	Results    []PSMagSearchResult `json:"results"`
	Count      int                 `json:"count"`
	TotalCount int                 `json:"total_count"`
	Page       int                 `json:"page"`
	TotalPages int                 `json:"total_pages"`
	Query      string              `json:"query"`
}

package sb_700_20

type PageResponse[T any] struct {
	Items      []T  `json:"items"`
	Count      int  `json:"count"`
	Page       int  `json:"page"`
	TotalPages int  `json:"total_pages"`
	IsLastPage bool `json:"is_last_page"`
}

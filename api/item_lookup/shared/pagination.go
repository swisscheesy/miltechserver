package shared

import "math"

const DefaultPageSize = int64(20)

type PagedResponse struct {
	Count      int  `json:"count"`
	Page       int  `json:"page"`
	TotalPages int  `json:"total_pages"`
	IsLastPage bool `json:"is_last_page"`
}

func CalculateTotalPages(totalCount int, pageSize int64) int {
	if totalCount == 0 {
		return 1
	}
	return int(math.Ceil(float64(totalCount) / float64(pageSize)))
}

func CalculateOffset(page int, pageSize int64) int64 {
	return pageSize * int64(page-1)
}

package pagination

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePage reads the "page" query parameter, defaulting to 1 if absent,
// and validates it is a positive integer. On invalid input it writes a
// 400 response and returns (0, false); callers must return immediately
// when ok is false.
func ParsePage(c *gin.Context) (page int, ok bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return 0, false
	}
	return page, true
}

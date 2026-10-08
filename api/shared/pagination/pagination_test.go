package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParsePage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryString string
		wantPage    int
		wantOK      bool
	}{
		{"valid page", "?page=3", 3, true},
		{"missing page defaults to 1", "", 1, true},
		{"zero page is invalid", "?page=0", 0, false},
		{"negative page is invalid", "?page=-1", 0, false},
		{"non-numeric page is invalid", "?page=abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.queryString, nil)
			c.Request = req

			page, ok := ParsePage(c)

			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.Equal(t, tt.wantPage, page)
			} else {
				require.Equal(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}

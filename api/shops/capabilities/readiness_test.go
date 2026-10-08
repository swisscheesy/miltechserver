package capabilities

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"miltechserver/internal/testsql"
	"net/http/httptest"
	"testing"
)

func TestCapabilitiesFlagAndReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, enabled := range []bool{false, true} {
		for _, ready := range []bool{false, true} {
			calls := 0
			router := gin.New()
			group := router.Group("", shared.ContractMiddleware, func(c *gin.Context) { c.Set("user", &bootstrap.User{UserID: "member"}) })
			RegisterRoutes(group, Flags{AtomicNotificationSave: enabled, AtomicNotificationReady: func(context.Context) bool { calls++; return ready }})
			req := httptest.NewRequest("GET", "/shops/capabilities", nil)
			req.Header.Set(shared.ContractHeader, "2")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, 200, rec.Code)
			var body struct{ Data map[string]any }
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, enabled && ready, body.Data["atomic_notification_save"])
			require.Equal(t, false, body.Data["service_dates"])
			require.Equal(t, false, body.Data["service_reads"])
			if enabled {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		}
	}
}

func TestAtomicReadinessReadOnlyAndFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name     string
		result   bool
		queryErr error
		want     bool
	}{
		{"ready", true, nil, true}, {"missing", false, nil, false}, {"error", false, errors.New("secret connection details"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			behavior := &testsql.Behavior{Columns: []string{"ready"}, Rows: [][]driver.Value{{tc.result}}, QueryErr: tc.queryErr}
			db := testsql.Open(t, behavior)
			require.Equal(t, tc.want, AtomicReady(context.Background(), db))
			require.True(t, behavior.BeginReadOnly.Load())
			require.Zero(t, behavior.Execs.Load())
		})
	}
	require.False(t, AtomicReady(context.Background(), nil))
}

package shops_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"miltechserver/api/shops/shared"
	"net/http/httptest"
	"testing"
)

func TestContractCapabilitiesProductionRouter(t *testing.T) {
	router := newTestRouter(t)
	for _, selector := range []string{"", "99", "2"} {
		req := httptest.NewRequest("GET", "/api/v1/auth/shops/capabilities", nil)
		req.Header.Set("X-User-ID", "contract-user")
		req.Header.Set(shared.ContractHeader, selector)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var body struct {
			Status int
			Code   string
			Data   map[string]interface{}
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		if selector != "2" {
			require.Equal(t, 400, w.Code)
			require.Equal(t, "unsupported_contract", body.Code)
			continue
		}
		require.Equal(t, 200, w.Code)
		require.Equal(t, float64(2), body.Data["contract_version"])
		for _, key := range []string{"typed_errors", "atomic_notification_save", "service_dates", "service_reads", "message_sync"} {
			require.Equal(t, false, body.Data[key])
		}
	}
}

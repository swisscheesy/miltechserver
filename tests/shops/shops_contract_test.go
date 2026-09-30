package shops_test

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
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

func capabilityValue(t *testing.T, router *gin.Engine, key string) any {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/auth/shops/capabilities", nil)
	req.Header.Set("X-User-ID", "contract-user")
	req.Header.Set(shared.ContractHeader, "2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var body struct{ Data map[string]interface{} }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Data[key]
}

// The trigger is disabled for the duration of this test only. Tests in this
// package run serially (no t.Parallel), and the cleanup re-enables it even when
// an assertion fails, so no other test can observe the disabled trigger.
func TestMessageSyncCapabilityRequiresFlagAndReadySchema(t *testing.T) {
	require.Equal(t, false, capabilityValue(t, newTestRouterWithFlags(t, false, false), "message_sync"))
	require.Equal(t, true, capabilityValue(t, newTestRouterWithFlags(t, false, true), "message_sync"))

	_, err := testDB.Exec(`ALTER TABLE public.shop_messages DISABLE TRIGGER shop_messages_assign_insertion_number`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := testDB.Exec(`ALTER TABLE public.shop_messages ENABLE TRIGGER shop_messages_assign_insertion_number`)
		require.NoError(t, err)
	})
	require.Equal(t, false, capabilityValue(t, newTestRouterWithFlags(t, false, true), "message_sync"))
}

func TestMessageSyncCapabilityLeavesOtherCapabilitiesUnchanged(t *testing.T) {
	router := newTestRouterWithFlags(t, true, true)
	require.Equal(t, true, capabilityValue(t, router, "message_sync"))
	require.Equal(t, true, capabilityValue(t, router, "atomic_notification_save"))
	for _, key := range []string{"typed_errors", "service_dates", "service_reads"} {
		require.Equal(t, false, capabilityValue(t, router, key))
	}
}

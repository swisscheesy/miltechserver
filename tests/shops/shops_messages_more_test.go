package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessagesPaginatedUpdateDelete(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Paged Shop")
	messageID := createMessage(t, router, "user-1", shopID, "First message")
	_ = createMessage(t, router, "user-1", shopID, "Second message")

	getPagedResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages/paginated?page=1&limit=1", nil, "user-1")
	require.Equal(t, http.StatusOK, getPagedResp.Code)

	updateBody := map[string]interface{}{
		"message_id": messageID,
		"message":    "Updated message",
	}

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/messages", updateBody, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/messages/"+messageID, nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)
}

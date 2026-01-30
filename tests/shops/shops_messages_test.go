package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessagesCreateAndGet(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	createShopBody := map[string]interface{}{
		"name":    "Message Shop",
		"details": "Message flow",
	}

	createShopResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops", createShopBody, "user-1")
	require.Equal(t, http.StatusCreated, createShopResp.Code)

	created := decodeStandardResponse(t, createShopResp.Body)
	shopData := decodeMap(t, created.Data)
	shopID, ok := shopData["id"].(string)
	require.True(t, ok)

	messageBody := map[string]interface{}{
		"shop_id": shopID,
		"message": "Hello shop",
	}

	createMessageResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", messageBody, "user-1")
	require.Equal(t, http.StatusCreated, createMessageResp.Code)

	getMessagesResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages", nil, "user-1")
	require.Equal(t, http.StatusOK, getMessagesResp.Code)

	messages := decodeStandardResponse(t, getMessagesResp.Body)
	messagesList := decodeSlice(t, messages.Data)
	require.Len(t, messagesList, 1)
}

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

func TestCreateMessageReply(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Reply Shop")

	// Create a top-level message to reply to
	parentID := createMessage(t, router, "user-1", shopID, "Parent message")

	// Create a reply with parent_id set
	replyBody := map[string]interface{}{
		"shop_id":   shopID,
		"message":   "Reply to parent",
		"parent_id": parentID,
	}

	replyResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", replyBody, "user-1")
	require.Equal(t, http.StatusCreated, replyResp.Code)

	resp := decodeStandardResponse(t, replyResp.Body)
	replyData := decodeMap(t, resp.Data)

	require.Equal(t, parentID, replyData["parent_id"], "reply should have parent_id set to the parent message's id")
}

func TestMessageParentIDInGetMessages(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Get Reply Shop")

	parentID := createMessage(t, router, "user-1", shopID, "Parent message")

	replyBody := map[string]interface{}{
		"shop_id":   shopID,
		"message":   "Reply message",
		"parent_id": parentID,
	}
	replyResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", replyBody, "user-1")
	require.Equal(t, http.StatusCreated, replyResp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages", nil, "user-1")
	require.Equal(t, http.StatusOK, getResp.Code)

	messages := decodeStandardResponse(t, getResp.Body)
	messagesList := decodeSlice(t, messages.Data)
	require.Len(t, messagesList, 2, "shop should have 2 messages: parent and reply")

	// Find the reply message and assert its parent_id
	foundReply := false
	for _, m := range messagesList {
		msg := m.(map[string]interface{})
		if msg["message"] == "Reply message" {
			require.Equal(t, parentID, msg["parent_id"], "reply message should have parent_id set")
			foundReply = true
		}
	}
	require.True(t, foundReply, "reply message should be present in get response")
}

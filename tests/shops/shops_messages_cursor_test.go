package shops_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMessagesPaginatedCursorBefore(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Cursor Shop")
	_ = createMessage(t, router, "user-1", shopID, "First message")
	time.Sleep(1 * time.Millisecond)
	_ = createMessage(t, router, "user-1", shopID, "Second message")
	time.Sleep(1 * time.Millisecond)
	cursorMessageID := createMessage(t, router, "user-1", shopID, "Third message")

	getPagedResp := doJSONRequest(
		t,
		router,
		http.MethodGet,
		"/api/v1/auth/shops/"+shopID+"/messages/paginated?before_id="+cursorMessageID+"&limit=1",
		nil,
		"user-1",
	)
	require.Equal(t, http.StatusOK, getPagedResp.Code)

	decoded := decodeStandardResponse(t, getPagedResp.Body)
	data := decodeMap(t, decoded.Data)

	messages, ok := data["messages"].([]interface{})
	require.True(t, ok)
	require.Len(t, messages, 1)

	message, ok := messages[0].(map[string]interface{})
	require.True(t, ok)
	require.NotEqual(t, cursorMessageID, message["id"])

	nextCursor, ok := data["next_cursor"].(string)
	require.True(t, ok)
	require.NotEmpty(t, nextCursor)

	_, hasPagination := data["pagination"]
	require.False(t, hasPagination)
}

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
	first := createMessage(t, router, "user-1", shopID, "First message")
	second := createMessage(t, router, "user-1", shopID, "Second message")
	cursorMessageID := createMessage(t, router, "user-1", shopID, "Third message")
	setLegacyMessageFixtureTimes(t, []string{first, second, cursorMessageID})

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
	requireAuthorUsername(t, message, "test-user")

	nextCursor, ok := data["next_cursor"].(string)
	require.True(t, ok)
	require.NotEmpty(t, nextCursor)

	_, hasPagination := data["pagination"]
	require.False(t, hasPagination)
}

func TestMessagesPaginatedCursorAfterIncludesAuthorUsername(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "After Cursor Shop")
	cursorMessageID := createMessage(t, router, "user-1", shopID, "First message")
	second := createMessage(t, router, "user-1", shopID, "Second message")
	setLegacyMessageFixtureTimes(t, []string{cursorMessageID, second})

	resp := doJSONRequest(
		t,
		router,
		http.MethodGet,
		"/api/v1/auth/shops/"+shopID+"/messages/paginated?after_id="+cursorMessageID+"&limit=1",
		nil,
		"user-1",
	)
	require.Equal(t, http.StatusOK, resp.Code)

	data := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	messages := data["messages"].([]interface{})
	require.Len(t, messages, 1)
	requireAuthorUsername(t, messages[0].(map[string]interface{}), "test-user")
}

func setLegacyMessageFixtureTimes(t *testing.T, ids []string) {
	t.Helper()
	base := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for i, id := range ids {
		_, err := testDB.Exec(`UPDATE shop_messages SET created_at=$1 WHERE id=$2`, base.Add(time.Duration(i)*time.Second), id)
		require.NoError(t, err)
	}
}

package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListsWithItemsAggregate(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Aggregate Lists")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Lists")
	listID := createList(t, router, "user-1", shopID)
	hiddenListID := createList(t, router, "other-user", hiddenShopID)
	createListItem(t, router, "user-1", listID, "111111111", "Visible Item")
	createListItem(t, router, "other-user", hiddenListID, "222222222", "Hidden Item")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/lists-with-items", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	lists := payload["lists"].([]interface{})
	require.Len(t, lists, 1)
	list := lists[0].(map[string]interface{})
	require.Equal(t, listID, list["id"])
	items := list["items"].([]interface{})
	require.Len(t, items, 1)
	require.Equal(t, "111111111", items[0].(map[string]interface{})["niin"])
}

func TestListsWithItemsAggregateRejectsNonMember(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Private Lists")
	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/lists-with-items", nil, "user-2")
	require.Equal(t, http.StatusForbidden, resp.Code)
	require.Equal(t, "access denied", decodeStandardResponse(t, resp.Body).Message)
}

func TestListsWithItemsAggregatePreservesEmptyLists(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Empty Lists")
	listID := createList(t, router, "user-1", shopID)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/lists-with-items", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	lists := payload["lists"].([]interface{})
	require.Len(t, lists, 1)
	list := lists[0].(map[string]interface{})
	require.Equal(t, listID, list["id"])
	require.Empty(t, list["items"].([]interface{}))
}

package shops_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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

func TestListsWithItemsAggregateAppliesDefaultCaps(t *testing.T) {
	const (
		expectedListLimit = 50
		expectedItemLimit = 50
	)

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Default Capped Lists")
	insertAggregateLists(t, shopID, "user-1", expectedListLimit+1, 1)
	cappedItemListID := insertAggregateList(t, shopID, "user-1", "default-capped-item-list", time.Now().Add(time.Hour))
	insertAggregateListItems(t, cappedItemListID, "user-1", expectedItemLimit+1)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/lists-with-items", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	limits := payload["limits"].(map[string]interface{})
	require.Equal(t, float64(expectedListLimit), limits["lists"])
	require.Equal(t, float64(expectedItemLimit), limits["items_per_list"])
	lists := payload["lists"].([]interface{})
	require.Len(t, lists, expectedListLimit)
	firstList := lists[0].(map[string]interface{})
	require.Equal(t, cappedItemListID, firstList["id"])
	require.Len(t, firstList["items"].([]interface{}), expectedItemLimit)
}

func TestListsWithItemsAggregateClampsMaxCaps(t *testing.T) {
	const (
		expectedMaxListLimit = 200
		expectedMaxItemLimit = 200
	)

	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Max Capped Lists")
	insertAggregateLists(t, shopID, "user-1", expectedMaxListLimit+1, 1)
	cappedItemListID := insertAggregateList(t, shopID, "user-1", "max-capped-item-list", time.Now().Add(time.Hour))
	insertAggregateListItems(t, cappedItemListID, "user-1", expectedMaxItemLimit+1)

	path := "/api/v1/auth/shops/" + shopID + "/lists-with-items?lists_limit=999&items_limit=999"
	resp := doJSONRequest(t, router, http.MethodGet, path, nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeMap(t, decodeStandardResponse(t, resp.Body).Data)
	limits := payload["limits"].(map[string]interface{})
	require.Equal(t, float64(expectedMaxListLimit), limits["lists"])
	require.Equal(t, float64(expectedMaxItemLimit), limits["items_per_list"])
	lists := payload["lists"].([]interface{})
	require.Len(t, lists, expectedMaxListLimit)
	firstList := lists[0].(map[string]interface{})
	require.Equal(t, cappedItemListID, firstList["id"])
	require.Len(t, firstList["items"].([]interface{}), expectedMaxItemLimit)
}

func insertAggregateLists(t *testing.T, shopID string, userID string, count int, itemCount int) {
	t.Helper()

	for i := 0; i < count; i++ {
		listID := fmt.Sprintf("aggregate-list-%03d", i)
		createdAt := time.Now().Add(-time.Duration(i) * time.Second)
		insertAggregateList(t, shopID, userID, listID, createdAt)
		insertAggregateListItems(t, listID, userID, itemCount)
	}
}

func insertAggregateList(t *testing.T, shopID string, userID string, listID string, createdAt time.Time) string {
	t.Helper()

	_, err := testDB.Exec(
		`INSERT INTO shop_lists (id, shop_id, created_by, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		listID,
		shopID,
		userID,
		"Aggregate list "+listID,
		createdAt.UTC(),
		createdAt.UTC(),
	)
	require.NoError(t, err)

	return listID
}

func insertAggregateListItems(t *testing.T, listID string, userID string, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		createdAt := time.Now().Add(time.Duration(i) * time.Second).UTC()
		_, err := testDB.Exec(
			`INSERT INTO shop_list_items (
				id, list_id, niin, nomenclature, quantity, added_by, created_at, updated_at, unit_of_measure
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			fmt.Sprintf("%s-item-%03d", listID, i),
			listID,
			fmt.Sprintf("%09d", i),
			fmt.Sprintf("Aggregate Item %03d", i),
			1,
			userID,
			createdAt,
			createdAt,
			"ea",
		)
		require.NoError(t, err)
	}
}

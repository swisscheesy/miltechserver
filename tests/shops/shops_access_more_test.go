package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAdminEndpoint(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Admin Shop")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/is-admin", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeStandardResponse(t, resp.Body)
	data := decodeMap(t, payload.Data)
	require.Equal(t, true, data["is_admin"])
}

func TestAdminOnlyListsEnforcement(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "AdminOnly Lists")
	_, inviteCode := createInviteCode(t, router, "user-1", shopID)

	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]interface{}{"invite_code": inviteCode}, "user-2")
	require.Equal(t, http.StatusOK, joinResp.Code)

	setAdminOnly := map[string]interface{}{"admin_only_lists": true}
	setResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/"+shopID+"/settings/admin-only-lists", setAdminOnly, "user-1")
	require.Equal(t, http.StatusOK, setResp.Code)

	createListBody := map[string]interface{}{
		"shop_id":     shopID,
		"description": "Should fail",
	}

	createListResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", createListBody, "user-2")
	require.Equal(t, http.StatusInternalServerError, createListResp.Code)

	promoteBody := map[string]interface{}{"shop_id": shopID, "target_user_id": "user-2"}
	promoteResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/members/promote", promoteBody, "user-1")
	require.Equal(t, http.StatusOK, promoteResp.Code)

	createListResp = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/lists", createListBody, "user-2")
	require.Equal(t, http.StatusCreated, createListResp.Code)
}

func TestLeaveShopDeletesWhenLastMember(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Leave Shop")

	leaveResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/"+shopID+"/leave", nil, "user-1")
	require.Equal(t, http.StatusOK, leaveResp.Code)

	shopsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops", nil, "user-1")
	require.Equal(t, http.StatusOK, shopsResp.Code)

	shops := decodeStandardResponse(t, shopsResp.Body)
	shopList := decodeSlice(t, shops.Data)
	require.Len(t, shopList, 0)
}

func TestDeleteShopByAdmin(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Delete Shop")

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/"+shopID, nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)

	shopsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops", nil, "user-1")
	require.Equal(t, http.StatusOK, shopsResp.Code)

	shops := decodeStandardResponse(t, shopsResp.Body)
	shopList := decodeSlice(t, shops.Data)
	require.Len(t, shopList, 0)
}

func TestMemberAdminActionsRestricted(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Member Actions")
	_, inviteCode := createInviteCode(t, router, "user-1", shopID)

	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]interface{}{"invite_code": inviteCode}, "user-2")
	require.Equal(t, http.StatusOK, joinResp.Code)

	removeBody := map[string]interface{}{"shop_id": shopID, "target_user_id": "user-1"}
	removeResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/members/remove", removeBody, "user-2")
	require.Equal(t, http.StatusInternalServerError, removeResp.Code)

	promoteBody := map[string]interface{}{"shop_id": shopID, "target_user_id": "user-2"}
	promoteResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/shops/members/promote", promoteBody, "user-2")
	require.Equal(t, http.StatusInternalServerError, promoteResp.Code)
}

func TestJoinWithInvalidOrInactiveCode(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Invite Guard")
	codeID, inviteCode := createInviteCode(t, router, "user-1", shopID)

	invalidResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]interface{}{"invite_code": "bad-code"}, "user-2")
	require.Equal(t, http.StatusInternalServerError, invalidResp.Code)

	deactivateResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/invite-codes/"+codeID, nil, "user-1")
	require.Equal(t, http.StatusOK, deactivateResp.Code)

	inactiveResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", map[string]interface{}{"invite_code": inviteCode}, "user-2")
	require.Equal(t, http.StatusInternalServerError, inactiveResp.Code)
}

func TestUserDataWithShops(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	_ = createShop(t, router, "user-1", "User Data Shop")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/user-data", nil, "user-1")
	require.Equal(t, http.StatusOK, resp.Code)

	payload := decodeStandardResponse(t, resp.Body)
	data := decodeMap(t, payload.Data)
	_, hasUser := data["user"]
	_, hasShops := data["shops"]
	require.True(t, hasUser)
	require.True(t, hasShops)
}

func TestNonMemberNotificationAccessDenied(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Notify Access")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	_ = createNotification(t, router, "user-1", shopID, vehicleID, "PM")

	resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/vehicles/"+vehicleID+"/notifications", nil, "user-2")
	require.Equal(t, http.StatusInternalServerError, resp.Code)
}

package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInviteAndJoinShop(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	router := newTestRouter(t)

	createBody := map[string]interface{}{
		"name":    "Invite Shop",
		"details": "Invite flow",
	}

	createResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops", createBody, "user-1")
	require.Equal(t, http.StatusCreated, createResp.Code)

	created := decodeStandardResponse(t, createResp.Body)
	shopData := decodeMap(t, created.Data)
	shopID, ok := shopData["id"].(string)
	require.True(t, ok)

	inviteBody := map[string]interface{}{
		"shop_id": shopID,
	}

	inviteResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/invite-codes", inviteBody, "user-1")
	require.Equal(t, http.StatusCreated, inviteResp.Code)

	invite := decodeStandardResponse(t, inviteResp.Body)
	inviteData := decodeMap(t, invite.Data)
	code, ok := inviteData["code"].(string)
	require.True(t, ok)
	require.NotEmpty(t, code)

	joinBody := map[string]interface{}{
		"invite_code": code,
	}

	joinResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/join", joinBody, "user-2")
	require.Equal(t, http.StatusOK, joinResp.Code)

	membersResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/members", nil, "user-1")
	require.Equal(t, http.StatusOK, membersResp.Code)

	members := decodeStandardResponse(t, membersResp.Body)
	membersList := decodeSlice(t, members.Data)
	require.GreaterOrEqual(t, len(membersList), 2)
}

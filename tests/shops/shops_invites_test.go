package shops_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInviteLifecycle(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)

	shopID := createShop(t, router, "user-1", "Invite Shop")
	codeID, _ := createInviteCode(t, router, "user-1", shopID)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/invite-codes", nil, "user-1")
	require.Equal(t, http.StatusOK, listResp.Code)

	list := decodeStandardResponse(t, listResp.Body)
	codes := decodeSlice(t, list.Data)
	require.Len(t, codes, 1)

	deactivateResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/invite-codes/"+codeID, nil, "user-1")
	require.Equal(t, http.StatusOK, deactivateResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/shops/invite-codes/"+codeID+"/delete", nil, "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)
}

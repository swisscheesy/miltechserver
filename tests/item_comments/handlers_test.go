package item_comments_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type commentResponse struct {
	ID          string  `json:"id"`
	CommentNiin string  `json:"comment_niin"`
	AuthorID    string  `json:"author_id"`
	Text        string  `json:"text"`
	ParentID    *string `json:"parent_id"`
}

func TestItemCommentsRoutes(t *testing.T) {
	if !hasRelation(t, testDB, "item_comments") {
		t.Skip("item_comments table missing in test DB")
	}
	if !hasRelation(t, testDB, "item_comment_flags") {
		t.Skip("item_comment_flags table missing in test DB")
	}

	clearItemCommentsTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")
	ensureNsn(t, testDB, "123456789")
	ensureNsn(t, testDB, "987654321")

	router := newTestRouter(t)

	invalidNiinResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/items/ABC/comments", "", "")
	require.Equal(t, http.StatusBadRequest, invalidNiinResp.Code)

	unauthCreateResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments", `{"text":"hi"}`, "")
	require.Equal(t, http.StatusUnauthorized, unauthCreateResp.Code)

	created := createComment(t, router, "123456789", "hello", "user-1", nil)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "123456789", created.CommentNiin)
	require.Equal(t, "hello", created.Text)

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/items/123456789/comments", "", "")
	require.Equal(t, http.StatusOK, listResp.Code)

	var listPayload standardResponse
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listPayload))
	var list []commentResponse
	require.NoError(t, json.Unmarshal(listPayload.Data, &list))
	require.NotEmpty(t, list)

	updateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/items/123456789/comments/"+created.ID, `{"text":"updated"}`, "user-1")
	require.Equal(t, http.StatusOK, updateResp.Code)

	var updatePayload standardResponse
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &updatePayload))
	var updated commentResponse
	require.NoError(t, json.Unmarshal(updatePayload.Data, &updated))
	require.Equal(t, "updated", updated.Text)

	unauthUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/items/123456789/comments/"+created.ID, `{"text":"nope"}`, "")
	require.Equal(t, http.StatusUnauthorized, unauthUpdateResp.Code)

	forbiddenUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/items/123456789/comments/"+created.ID, `{"text":"nope"}`, "user-2")
	require.Equal(t, http.StatusForbidden, forbiddenUpdateResp.Code)

	invalidUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/items/123456789/comments/not-a-uuid", `{"text":"nope"}`, "user-1")
	require.Equal(t, http.StatusNotFound, invalidUpdateResp.Code)

	emptyUpdateResp := doJSONRequest(t, router, http.MethodPut, "/api/v1/auth/items/123456789/comments/"+created.ID, `{"text":""}`, "user-1")
	require.Equal(t, http.StatusBadRequest, emptyUpdateResp.Code)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/items/123456789/comments/"+created.ID, "", "user-1")
	require.Equal(t, http.StatusOK, deleteResp.Code)

	var deletePayload standardResponse
	require.NoError(t, json.Unmarshal(deleteResp.Body.Bytes(), &deletePayload))
	var deleted commentResponse
	require.NoError(t, json.Unmarshal(deletePayload.Data, &deleted))
	require.Equal(t, "Deleted by user", deleted.Text)

	unauthDeleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/items/123456789/comments/"+created.ID, "", "")
	require.Equal(t, http.StatusUnauthorized, unauthDeleteResp.Code)

	forbiddenDeleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/items/123456789/comments/"+created.ID, "", "user-2")
	require.Equal(t, http.StatusForbidden, forbiddenDeleteResp.Code)

	missingDeleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/items/123456789/comments/00000000-0000-0000-0000-000000000000", "", "user-1")
	require.Equal(t, http.StatusNotFound, missingDeleteResp.Code)

	flagResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments/"+created.ID+"/flags", "", "user-1")
	require.Equal(t, http.StatusOK, flagResp.Code)

	repeatFlagResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments/"+created.ID+"/flags", "", "user-1")
	require.Equal(t, http.StatusOK, repeatFlagResp.Code)

	unauthFlagResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments/"+created.ID+"/flags", "", "")
	require.Equal(t, http.StatusUnauthorized, unauthFlagResp.Code)

	missingFlagResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments/00000000-0000-0000-0000-000000000000/flags", "", "user-1")
	require.Equal(t, http.StatusNotFound, missingFlagResp.Code)

	parentComment := createComment(t, router, "987654321", "parent", "user-1", nil)
	parentID := parentComment.ID
	invalidParentResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments", `{"text":"child","parent_id":"not-a-uuid"}`, "user-1")
	require.Equal(t, http.StatusBadRequest, invalidParentResp.Code)

	crossParentResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/items/123456789/comments", `{"text":"child","parent_id":"`+parentID+`"}`, "user-1")
	require.Equal(t, http.StatusBadRequest, crossParentResp.Code)
}

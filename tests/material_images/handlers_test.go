package material_images_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"miltechserver/api/response"

	"github.com/stretchr/testify/require"
)

func TestMaterialImagesLifecycle(t *testing.T) {
	clearMaterialImagesTables(t, testDB)
	userID := "user-images"
	ensureUser(t, testDB, userID)

	router := newTestRouter(t)

	uploadResp := doMultipartRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/auth/material-images/upload",
		map[string]string{"niin": "123456789"},
		"image",
		"test.jpg",
		[]byte("image-bytes"),
		userID,
	)
	require.Equal(t, http.StatusCreated, uploadResp.Code)

	var uploadPayload response.ImageUploadResponse
	require.NoError(t, json.Unmarshal(uploadResp.Body.Bytes(), &uploadPayload))
	require.NotNil(t, uploadPayload.Image)
	require.NotEmpty(t, uploadPayload.Image.ID)
	imageID := uploadPayload.Image.ID

	listResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/material-images/niin/123456789", nil, "")
	require.Equal(t, http.StatusOK, listResp.Code)

	var listPayload response.PaginatedImagesResponse
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listPayload))
	require.Len(t, listPayload.Images, 1)

	listWithUserResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/material-images/niin/123456789", nil, userID)
	require.Equal(t, http.StatusOK, listWithUserResp.Code)

	var listWithUserPayload response.PaginatedImagesResponse
	require.NoError(t, json.Unmarshal(listWithUserResp.Body.Bytes(), &listWithUserPayload))
	require.Len(t, listWithUserPayload.Images, 1)
	require.Nil(t, listWithUserPayload.Images[0].UserVote)

	byUserResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/material-images/user/"+userID, nil, userID)
	require.Equal(t, http.StatusOK, byUserResp.Code)

	var byUserPayload response.PaginatedImagesResponse
	require.NoError(t, json.Unmarshal(byUserResp.Body.Bytes(), &byUserPayload))
	require.Len(t, byUserPayload.Images, 1)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/material-images/"+imageID, nil, "")
	require.Equal(t, http.StatusOK, getResp.Code)

	voteResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/material-images/"+imageID+"/vote", map[string]string{"vote_type": "upvote"}, userID)
	require.Equal(t, http.StatusOK, voteResp.Code)

	var votePayload response.ImageVoteResponse
	require.NoError(t, json.Unmarshal(voteResp.Body.Bytes(), &votePayload))
	require.Equal(t, 1, votePayload.UpvoteCount)

	listWithUserAfterVote := doJSONRequest(t, router, http.MethodGet, "/api/v1/material-images/niin/123456789", nil, userID)
	require.Equal(t, http.StatusOK, listWithUserAfterVote.Code)

	var listWithUserAfterVotePayload response.PaginatedImagesResponse
	require.NoError(t, json.Unmarshal(listWithUserAfterVote.Body.Bytes(), &listWithUserAfterVotePayload))
	require.Len(t, listWithUserAfterVotePayload.Images, 1)
	require.NotNil(t, listWithUserAfterVotePayload.Images[0].UserVote)
	require.Equal(t, "upvote", *listWithUserAfterVotePayload.Images[0].UserVote)

	userGetResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/material-images/"+imageID, nil, userID)
	require.Equal(t, http.StatusOK, userGetResp.Code)

	var userGetPayload response.MaterialImageResponse
	require.NoError(t, json.Unmarshal(userGetResp.Body.Bytes(), &userGetPayload))
	require.NotNil(t, userGetPayload.UserVote)
	require.Equal(t, "upvote", *userGetPayload.UserVote)

	removeVoteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/material-images/"+imageID+"/vote", nil, userID)
	require.Equal(t, http.StatusOK, removeVoteResp.Code)

	var removeVotePayload response.ImageVoteResponse
	require.NoError(t, json.Unmarshal(removeVoteResp.Body.Bytes(), &removeVotePayload))
	require.Equal(t, 0, removeVotePayload.UpvoteCount)

	flagResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/material-images/"+imageID+"/flag", map[string]string{"reason": "Incorrect Item", "description": "bad"}, userID)
	require.Equal(t, http.StatusOK, flagResp.Code)

	flagsResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/material-images/"+imageID+"/flags", nil, userID)
	require.Equal(t, http.StatusOK, flagsResp.Code)

	var flagsPayload map[string]interface{}
	require.NoError(t, json.Unmarshal(flagsResp.Body.Bytes(), &flagsPayload))
	flagsRaw, ok := flagsPayload["flags"].([]interface{})
	require.True(t, ok)
	require.Len(t, flagsRaw, 1)

	deleteResp := doJSONRequest(t, router, http.MethodDelete, "/api/v1/auth/material-images/"+imageID, nil, userID)
	require.Equal(t, http.StatusOK, deleteResp.Code)

	afterDeleteList := doJSONRequest(t, router, http.MethodGet, "/api/v1/material-images/niin/123456789", nil, "")
	require.Equal(t, http.StatusOK, afterDeleteList.Code)

	var afterDeletePayload response.PaginatedImagesResponse
	require.NoError(t, json.Unmarshal(afterDeleteList.Body.Bytes(), &afterDeletePayload))
	require.Len(t, afterDeletePayload.Images, 0)
}

func TestMaterialImagesRateLimit(t *testing.T) {
	clearMaterialImagesTables(t, testDB)
	userID := "user-rate"
	ensureUser(t, testDB, userID)

	router := newTestRouter(t)

	for i := 0; i < 3; i++ {
		resp := doMultipartRequest(
			t,
			router,
			http.MethodPost,
			"/api/v1/auth/material-images/upload",
			map[string]string{"niin": "987654321"},
			"image",
			"test.jpg",
			[]byte("image-bytes"),
			userID,
		)
		require.Equal(t, http.StatusCreated, resp.Code)
	}

	rateResp := doMultipartRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/auth/material-images/upload",
		map[string]string{"niin": "987654321"},
		"image",
		"test.jpg",
		[]byte("image-bytes"),
		userID,
	)
	require.Equal(t, http.StatusBadRequest, rateResp.Code)
	require.Contains(t, rateResp.Body.String(), "rate limit exceeded")
}

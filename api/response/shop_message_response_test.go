package response

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"testing"
	"time"
)

func TestShopMessageLegacyNineKeys(t *testing.T) {
	now := time.Date(2026, 10, 3, 1, 2, 3, 123000000, time.FixedZone("legacy", -7*3600))
	edited := true
	message := model.ShopMessages{ID: "message-id", ShopID: "shop-id", UserID: "author-id", Message: "persisted text", CreatedAt: &now, IsEdited: &edited}
	dto := NewShopMessageResponse(message, nil)
	data, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	keys := []string{}
	for key := range got {
		keys = append(keys, key)
	}
	require.ElementsMatch(t, []string{"id", "shop_id", "user_id", "message", "created_at", "updated_at", "is_edited", "parent_id", "author_username"}, keys)
	require.Equal(t, "message-id", got["id"])
	require.Equal(t, "persisted text", got["message"])
	require.Equal(t, "2026-10-03T01:02:03.123-07:00", got["created_at"])
	require.Equal(t, true, got["is_edited"])
	require.Nil(t, got["parent_id"])
	require.Nil(t, got["updated_at"])
	require.Nil(t, got["author_username"])
}

func TestShopMessagePreservesPopulatedAndNullValues(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	parent := "parent"
	author := "author name"
	edited := false
	dto := NewShopMessageResponse(model.ShopMessages{ID: "child", ShopID: "shop", UserID: "user", Message: "reply", UpdatedAt: &now, ParentID: &parent, IsEdited: &edited}, &author)
	data, err := json.Marshal(dto)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"child","shop_id":"shop","user_id":"user","message":"reply","created_at":null,"updated_at":"2026-10-03T12:00:00Z","is_edited":false,"parent_id":"parent","author_username":"author name"}`, string(data))
	empty := NewShopMessageResponse(model.ShopMessages{}, nil)
	data, err = json.Marshal(empty)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"","shop_id":"","user_id":"","message":"","created_at":null,"updated_at":null,"is_edited":null,"parent_id":null,"author_username":null}`, string(data))
}

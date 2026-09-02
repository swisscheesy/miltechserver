package shops_test

import (
	"testing"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

// TestCreateShopMessageRollsBackOnDuplicateID forces CreateShopMessage's
// INSERT to fail on a primary key violation (reusing an existing message
// ID) and confirms the transaction rolled back cleanly: the original
// message content is untouched, and no second row related to the failed
// call was created.
func TestCreateShopMessageRollsBackOnDuplicateID(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Rollback Shop")

	repo := messages.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "test-account"})
	user := &bootstrap.User{UserID: "user-1", Username: "test-user"}

	now := time.Now().UTC()
	isEdited := false
	original := model.ShopMessages{
		ID:        "duplicate-message-id",
		ShopID:    shopID,
		UserID:    user.UserID,
		Message:   "original message",
		CreatedAt: &now,
		UpdatedAt: &now,
		IsEdited:  &isEdited,
	}

	created, err := repo.CreateShopMessage(user, original)
	require.NoError(t, err)
	require.Equal(t, "original message", created.Message)

	conflicting := original
	conflicting.Message = "must not overwrite or duplicate"

	_, err = repo.CreateShopMessage(user, conflicting)
	require.Error(t, err, "duplicate primary key insert must fail")

	fetched, err := repo.GetShopMessageByID(user, original.ID)
	require.NoError(t, err)
	require.Equal(t, "original message", fetched.Message, "rollback must leave the original row untouched")

	count, err := repo.GetShopMessagesCount(user, shopID)
	require.NoError(t, err)
	require.Equal(t, int64(1), count, "the failed transaction must not have left a partial write")
}

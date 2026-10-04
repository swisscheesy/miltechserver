package messages

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/stretchr/testify/require"
	"io"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"net/http"
	"strings"
	"testing"
)

type deletionAuthorityRepository struct {
	Repository
	cloudDeletes int
}

func (r *deletionAuthorityRepository) GetShopMessageByID(context.Context, *bootstrap.User, string) (*response.ShopMessageResponse, error) {
	return &response.ShopMessageResponse{Message: "[IMAGE:https://foreign.blob.core.windows.net/shop-message-images/victim/file.jpg]"}, nil
}
func (r *deletionAuthorityRepository) DeleteShopMessage(context.Context, *bootstrap.User, string) error {
	return nil
}
func (r *deletionAuthorityRepository) DeleteBlobByURL(string) error { r.cloudDeletes++; return nil }
func TestMessageTextNeverAuthorizesCloudDeletion(t *testing.T) {
	r := &deletionAuthorityRepository{}
	require.NoError(t, NewService(r, nil).DeleteShopMessage(context.Background(), &bootstrap.User{UserID: "actor"}, "message"))
	require.Zero(t, r.cloudDeletes, "client text must never select a cloud deletion target")
}

func TestManagedMarkerCanonicalIdentity(t *testing.T) {
	storage := AssetStorage{Account: "owned", Container: "shop-message-images"}
	keys := managedMarkerKeys("[IMAGE:https://owned.blob.core.windows.net/shop-message-images/shop/a.jpg?sig=x] [IMAGE:https://owned.blob.core.windows.net/shop-message-images/shop/%62.jpg] [IMAGE:https://foreign.blob.core.windows.net/shop-message-images/shop/c.jpg] [IMAGE:https://owned.blob.core.windows.net.evil/shop-message-images/shop/d.jpg] [IMAGE:https://owned.blob.core.windows.net/shop-message-images/shop/../e.jpg]", storage)
	require.Equal(t, map[string]bool{"shop/a.jpg": true, "shop/b.jpg": true, "shop/../e.jpg": true}, keys)
}

func TestAzureBlobStoreVerifiesActualEndpoint(t *testing.T) {
	storage := AssetStorage{Account: "owned", Container: "shop-message-images"}
	a := Asset{ID: "id", ShopID: "shop", Account: "owned", Container: storage.Container, BlobKey: "shop/id.jpg", Extension: ".jpg"}
	for _, endpoint := range []string{"https://foreign.blob.core.windows.net", "https://owned.blob.core.windows.net.evil", "https://owned.blob.core.windows.net/another-account", "http://owned.blob.core.windows.net"} {
		t.Run(endpoint, func(t *testing.T) {
			client, err := azblob.NewClientWithNoCredential(endpoint, nil)
			require.NoError(t, err)
			store := NewAzureBlobStore(client, storage)
			require.Error(t, store.Delete(context.Background(), a))
			require.Error(t, store.Upload(context.Background(), a, []byte("test"), "image/jpeg"))
		})
	}
	calls := 0
	transport := assetTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/shop-message-images/shop/id.jpg", req.URL.Path)
		return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})
	client, err := azblob.NewClientWithNoCredential("https://owned.blob.core.windows.net", &azblob.ClientOptions{ClientOptions: azcore.ClientOptions{Transport: transport}})
	require.NoError(t, err)
	store := NewAzureBlobStore(client, storage)
	require.NoError(t, store.Delete(context.Background(), a))
	require.Equal(t, 1, calls)
	foreign := a
	foreign.Account = "foreign"
	require.Error(t, store.Delete(context.Background(), foreign))
	require.Equal(t, 1, calls)
}

type assetTransport func(*http.Request) (*http.Response, error)

func (f assetTransport) Do(r *http.Request) (*http.Response, error) { return f(r) }

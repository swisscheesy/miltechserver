package messages

import (
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/google/uuid"
	"net/url"
	"strings"
)

type BlobStore interface {
	Upload(context.Context, Asset, []byte, string) error
	Delete(context.Context, Asset) error
	ListPrefix(context.Context, string, string, string, string) ([]string, string, error)
}

type azureBlobStore struct {
	client  *azblob.Client
	storage AssetStorage
}

func NewAzureBlobStore(client *azblob.Client, storage AssetStorage) BlobStore {
	return &azureBlobStore{client: client, storage: storage}
}

func (b *azureBlobStore) validate(a Asset) error {
	if b.client == nil || !b.storage.valid() {
		return errCleanupTarget
	}
	endpoint, err := url.Parse(b.client.URL())
	if err != nil || endpoint.Scheme != "https" || endpoint.User != nil || !strings.EqualFold(endpoint.Host, b.storage.Account+".blob.core.windows.net") || (endpoint.Path != "" && endpoint.Path != "/") {
		return errCleanupTarget
	}
	if a.State == "shop_prefix" {
		for _, part := range strings.Split(a.BlobKey, "/") {
			if part == "." || part == ".." {
				return errCleanupTarget
			}
		}
		shop, err := uuid.Parse(a.ShopID)
		if err != nil || shop.String() != a.ShopID || a.Account != b.storage.Account || a.Container != b.storage.Container || !strings.HasPrefix(a.BlobKey, a.ShopID+"/") {
			return errCleanupTarget
		}
		return nil
	}
	if a.Account != b.storage.Account || a.Container != b.storage.Container || a.ID == "" || a.ShopID == "" || a.BlobKey != a.ShopID+"/"+a.ID+a.Extension {
		return errCleanupTarget
	}
	return nil
}

func (b *azureBlobStore) Upload(ctx context.Context, a Asset, data []byte, contentType string) error {
	if a.State == "shop_prefix" {
		return errCleanupTarget
	}
	if err := b.validate(a); err != nil {
		return err
	}
	_, err := b.client.UploadBuffer(ctx, a.Container, a.BlobKey, data, &azblob.UploadBufferOptions{HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &contentType}})
	return err
}
func (b *azureBlobStore) Delete(ctx context.Context, a Asset) error {
	if err := b.validate(a); err != nil {
		return err
	}
	_, err := b.client.DeleteBlob(ctx, a.Container, a.BlobKey, nil)
	var response *azcore.ResponseError
	if errors.As(err, &response) && response.StatusCode == 404 {
		return nil
	}
	return err
}

func (b *azureBlobStore) ListPrefix(ctx context.Context, account, container, prefix, cursor string) ([]string, string, error) {
	shop := strings.TrimSuffix(prefix, "/")
	if prefix != shop+"/" {
		return nil, "", errCleanupTarget
	}
	if err := b.validate(Asset{Account: account, Container: container, ShopID: shop, BlobKey: prefix, State: "shop_prefix"}); err != nil {
		return nil, "", err
	}
	max := int32(100)
	options := &azblob.ListBlobsFlatOptions{Prefix: &prefix, MaxResults: &max}
	if cursor != "" {
		options.Marker = &cursor
	}
	pager := b.client.NewListBlobsFlatPager(container, options)
	page, err := pager.NextPage(ctx)
	if err != nil {
		return nil, "", err
	}
	keys := []string{}
	if page.Segment != nil {
		for _, item := range page.Segment.BlobItems {
			if item.Name != nil {
				keys = append(keys, *item.Name)
			}
		}
	}
	next := ""
	if page.NextMarker != nil {
		next = *page.NextMarker
	}
	return keys, next, nil
}

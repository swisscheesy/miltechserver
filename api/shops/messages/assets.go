package messages

import (
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	UploadOperationTimeout = 30 * time.Second
	UploadLeaseDuration    = 2 * time.Minute
)

type ImageUpload struct {
	MessageID     string `json:"message_id"`
	ShopID        string `json:"shop_id"`
	ImageURL      string `json:"image_url"`
	FileExtension string `json:"file_extension"`
}

type Asset struct {
	ID, OperationID, ShopID, UploaderID, Account, Container, BlobKey, URL, Extension, State string
	LeaseUntil                                                                              time.Time
}

type AssetStorage struct{ Account, Container string }

var accountNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (s AssetStorage) valid() bool {
	return accountNamePattern.MatchString(s.Account) && s.Container == shopMessageImagesContainer
}

var imageMarkers = regexp.MustCompile(`\[IMAGE:([^\]]+)\]`)

type assetTarget struct{ Account, Container, BlobKey string }

// URL text selects registry candidates, never cloud targets. The registry match
// establishes ownership; configuration drift must not erase published refs.
func managedMarkerTargets(text string) map[assetTarget]bool {
	targets := map[assetTarget]bool{}
	for _, match := range imageMarkers.FindAllStringSubmatch(text, -1) {
		u, err := url.Parse(match[1])
		if err != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
			continue
		}
		host := strings.ToLower(u.Hostname())
		account := strings.TrimSuffix(host, ".blob.core.windows.net")
		storage := AssetStorage{Account: account, Container: shopMessageImagesContainer}
		if !storage.valid() || host != account+".blob.core.windows.net" {
			continue
		}
		prefix := "/" + storage.Container + "/"
		if !strings.HasPrefix(u.Path, prefix) {
			continue
		}
		// Parse decodes once; never clean paths or decode again.
		key := strings.TrimPrefix(u.Path, prefix)
		if key != "" {
			targets[assetTarget{account, storage.Container, key}] = true
		}
	}
	return targets
}

func managedMarkerKeys(text string, storage AssetStorage) map[string]bool {
	keys := map[string]bool{}
	for target := range managedMarkerTargets(text) {
		if target.Account == storage.Account && target.Container == storage.Container {
			keys[target.BlobKey] = true
		}
	}
	return keys
}

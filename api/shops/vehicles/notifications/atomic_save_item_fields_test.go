package notifications

import (
	"crypto/sha256"
	"encoding/json"
	"miltechserver/api/request"
	"sort"
	"strings"
	"testing"
)

func stringPtr(value string) *string { return &value }

// releasedSaveItem is the item shape released clients send, frozen from the
// contract before nickname/unit_of_measure existed.
type releasedSaveItem struct {
	ID           string `json:"id"`
	Niin         string `json:"niin"`
	Nomenclature string `json:"nomenclature"`
	Quantity     int32  `json:"quantity"`
}

func releasedClientFingerprint(t *testing.T, r request.NotificationSaveRequest) [32]byte {
	t.Helper()
	items := make([]releasedSaveItem, len(r.Items))
	for i, item := range r.Items {
		items[i] = releasedSaveItem{ID: item.ID, Niin: item.Niin, Nomenclature: item.Nomenclature, Quantity: item.Quantity}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	payload := struct {
		ShopID         string                             `json:"shop_id"`
		VehicleID      string                             `json:"vehicle_id"`
		NotificationID *string                            `json:"notification_id"`
		Details        request.NotificationSaveDetails    `json:"details"`
		Attachment     request.NotificationSaveAttachment `json:"attachment"`
		Items          []releasedSaveItem                 `json:"items"`
	}{r.ShopID, r.VehicleID, r.NotificationID, r.Details, r.Attachment, items}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}

// Stored operation fingerprints from before the deploy must still match a
// released client's retry, or its replay would be rejected as a conflict.
func TestFingerprintOfReleasedClientPayloadIsUnchanged(t *testing.T) {
	r := atomicRequest()
	r.Items = []request.NotificationSaveItem{
		{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Niin: "456", Nomenclature: "other", Quantity: 2},
		{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Niin: "123", Nomenclature: "part", Quantity: 1},
	}
	got, err := FingerprintNotificationSave(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != releasedClientFingerprint(t, r) {
		t.Fatal("omitted nickname/unit changed the released-client fingerprint")
	}

	r.Items[0].UnitOfMeasure = stringPtr("KT")
	withUnit, err := FingerprintNotificationSave(r)
	if err != nil {
		t.Fatal(err)
	}
	if withUnit == got {
		t.Fatal("unit of measure must be part of the fingerprint")
	}
}

func TestReleasedClientJSONDecodesWithNilItemFields(t *testing.T) {
	var item request.NotificationSaveItem
	if err := json.Unmarshal([]byte(`{"id":"a","niin":"123","nomenclature":"part","quantity":1}`), &item); err != nil {
		t.Fatal(err)
	}
	if item.Nickname != nil || item.UnitOfMeasure != nil {
		t.Fatalf("omitted fields must decode as nil, got %+v", item)
	}
}

func TestAtomicValidationOfNicknameAndUnit(t *testing.T) {
	validItem := func() request.NotificationSaveItem {
		return request.NotificationSaveItem{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Niin: "123", Nomenclature: "part", Quantity: 1}
	}
	cases := []struct {
		name          string
		nickname      *string
		unitOfMeasure *string
		valid         bool
	}{
		{"omitted fields", nil, nil, true},
		{"empty nickname clears", stringPtr(""), stringPtr("EA"), true},
		{"nickname at limit", stringPtr(strings.Repeat("é", 50)), stringPtr("DZ"), true},
		{"nickname over limit", stringPtr(strings.Repeat("a", 51)), nil, false},
		{"empty unit", nil, stringPtr(""), false},
		{"blank unit", nil, stringPtr("   "), false},
		{"unit over limit", nil, stringPtr(strings.Repeat("A", 51)), false},
		{"unknown unit code is allowed", nil, stringPtr("ZZ"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := atomicRequest()
			item := validItem()
			item.Nickname = tc.nickname
			item.UnitOfMeasure = tc.unitOfMeasure
			r.Items = []request.NotificationSaveItem{item}
			err := ValidateNotificationSave(r)
			if tc.valid && err != nil {
				t.Fatalf("rejected valid item: %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatal("accepted invalid item")
			}
		})
	}
}

func TestDirectItemChanged(t *testing.T) {
	stored := request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 2, Nickname: stringPtr("Hub"), UnitOfMeasure: stringPtr("KT")}
	legacyStored := request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 2}
	cases := []struct {
		name    string
		stored  request.NotificationSaveItem
		desired request.NotificationSaveItem
		changed bool
	}{
		{"released client omits fields", stored, legacyStored, false},
		{"released client changes quantity", stored, request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 3}, true},
		{"same values sent", stored, stored, false},
		{"unit changed", stored, request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 2, Nickname: stringPtr("Hub"), UnitOfMeasure: stringPtr("EA")}, true},
		{"nickname cleared", stored, request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 2, Nickname: stringPtr(""), UnitOfMeasure: stringPtr("KT")}, true},
		{"legacy NULLs equal client defaults", legacyStored, request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 2, Nickname: stringPtr(""), UnitOfMeasure: stringPtr("EA")}, false},
		{"legacy NULL unit replaced", legacyStored, request.NotificationSaveItem{ID: "a", Niin: "123", Nomenclature: "part", Quantity: 2, Nickname: stringPtr(""), UnitOfMeasure: stringPtr("DZ")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := directItemChanged(tc.stored, tc.desired); got != tc.changed {
				t.Fatalf("directItemChanged = %v, want %v", got, tc.changed)
			}
		})
	}
}

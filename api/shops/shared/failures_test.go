package shared

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-jet/jet/v2/qrm"
	"testing"
)

func TestNormalizeWrappedNoRows(t *testing.T) {
	for _, source := range []error{sql.ErrNoRows, qrm.ErrNoRows} {
		if !errors.Is(NormalizeNoRows(fmt.Errorf("query: %w", source)), ErrNotFound) {
			t.Fatal("absence lost")
		}
	}
	err := errors.New("driver private details")
	if NormalizeNoRows(err) != err {
		t.Fatal("cause lost")
	}
}
func TestContractFailureMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		code   string
		status int
	}{
		{ErrListNotFound, "list_not_found", 404}, {ErrNotificationNotFound, "notification_not_found", 404},
		{ErrShopAccessDenied, "denied", 403}, {errors.New("driver password"), "internal_error", 500},
	} {
		f := ClassifyFailure(tc.err)
		if f.Code != tc.code || f.Status != tc.status {
			t.Fatalf("%+v", f)
		}
		if f.PublicMessage == "driver password" {
			t.Fatal("leak")
		}
	}
}

func TestContractKnownPublicErrorsAndSanitization(t *testing.T) {
	for _, tc := range []struct {
		message, code string
		status        int
	}{
		{"unauthorized user", "unauthorized", 401}, {"access denied: user is not a member of this shop", "denied", 403},
		{"shop list not found", "list_not_found", 404}, {"service not found", "service_not_found", 404},
		{"invalid attached_shop_list", "invalid", 400}, {"list is in use", "list_in_use", 409},
		{"operation payload conflicts with previous request", "operation_payload_conflict", 409},
	} {
		f := ClassifyFailure(fmt.Errorf("query: %w", errors.New(tc.message)))
		if f.Code != tc.code || f.Status != tc.status {
			t.Fatalf("%s: %+v", tc.message, f)
		}
	}
}

package vehicles

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"miltechserver/api/request"
	"testing"
)

func TestVehicleMutationIntentMatrix(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		usage      bool
	}{
		{"mileage only", `{"tracked_mileage":0}`, true},
		{"hours only", `{"tracked_hours":20}`, true},
		{"no tracked values", `{}`, false},
		{"null tracked values", `{"tracked_mileage":null,"tracked_hours":null}`, false},
		{"ignored snapshot labels", `{"admin":"SNAPSHOT","mileage":-100,"hours":-20,"tracked_mileage":120}`, true},
		{"empty details", `{"niin":"","model":"","serial":"","uoc":"","comment":"","tracked_hours":1}`, true},
		{"null details", `{"niin":null,"model":null,"serial":null,"uoc":null,"comment":null,"tracked_hours":1}`, true},
		{"niin edit", `{"niin":"NEW","tracked_hours":1}`, false},
		{"model edit", `{"model":"NEW","tracked_hours":1}`, false},
		{"serial edit", `{"serial":"NEW","tracked_hours":1}`, false},
		{"uoc edit", `{"uoc":"NEW","tracked_hours":1}`, false},
		{"comment edit", `{"comment":"NEW","tracked_hours":1}`, false},
		{"raw whitespace still metadata", `{"comment":" ","tracked_hours":1}`, false},
		{"admin only", `{"admin":"NEW"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r request.UpdateShopVehicleRequest
			require.NoError(t, json.Unmarshal([]byte(tc.body), &r))
			input := VehicleUpdateInput{Metadata: VehicleMetadataUpdate{Admin: r.Admin, Niin: r.Niin, Model: r.Model, Serial: r.Serial, Uoc: r.Uoc, Comment: r.Comment, Mileage: r.Mileage, Hours: r.Hours}, TrackedMileage: r.TrackedMileage, TrackedHours: r.TrackedHours}
			require.Equal(t, tc.usage, isTrackedUsageUpdate(input))
		})
	}
}

func TestBaseUsageValidation(t *testing.T) {
	zero, positive, negative := int32(0), int32(5), int32(-1)
	for _, tc := range []struct {
		name           string
		mileage, hours *int32
		invalid        bool
	}{
		{"omitted", nil, nil, false}, {"zero", &zero, &zero, false}, {"positive", &positive, &positive, false},
		{"negative mileage", &negative, nil, true}, {"negative hours", nil, &negative, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBaseUsage(tc.mileage, tc.hours)
			if tc.invalid {
				require.ErrorIs(t, err, ErrInvalidUsageAdjustment)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

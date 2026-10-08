package shared

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestValidateItemFields(t *testing.T) {
	for _, tc := range []struct {
		niin, nomenclature string
		quantity           int32
		valid              bool
	}{
		{" \t", "Part", 1, false}, {"NSN", " \n", 1, false}, {"NSN", "Part", 0, false}, {"NSN", "Part", -1, false},
		{"  NSN-raw  ", strings.Repeat("界", 10000), 1, true},
	} {
		err := ValidateItemFields(tc.niin, tc.nomenclature, tc.quantity)
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.True(t, IsValidationError(err))
		}
	}
	for _, value := range []string{"M1", "PM", "MW"} {
		require.NoError(t, ValidateNotificationType(value))
	}
	for _, value := range []string{"", " PM ", "OTHER"} {
		require.True(t, IsValidationError(ValidateNotificationType(value)))
	}
	empty, raw, long, blank := "", "  NEW-CODE  ", strings.Repeat("界", 51), " \t"
	require.NoError(t, ValidateItemEnrichment(&empty, &raw))
	require.Error(t, ValidateItemEnrichment(&long, nil))
	require.Error(t, ValidateItemEnrichment(nil, &long))
	require.Error(t, ValidateItemEnrichment(nil, &blank))
	require.NoError(t, ValidateNotificationFields(strings.Repeat("界", 10000), "PM"))
}

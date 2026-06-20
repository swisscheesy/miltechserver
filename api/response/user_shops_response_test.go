package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewResponseJSON(t *testing.T) {
	details := "Maintenance section"
	result := ShopEquipmentOverviewResponse{
		Shops: []ShopEquipmentOverview{{
			ID: "shop-1", Name: "Alpha Shop", Details: &details, Role: "member",
			EquipmentCount: 1,
			Equipment: []ShopEquipmentSummary{{
				ID: "equipment-1", Admin: "A123", Model: "M1097",
				Serial: "SER-1", Niin: "012345678",
			}},
		}},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"shops":[{
			"id":"shop-1","name":"Alpha Shop","details":"Maintenance section",
			"role":"member","equipment_count":1,
			"equipment":[{"id":"equipment-1","admin":"A123","model":"M1097","serial":"SER-1","niin":"012345678"}]
		}]
	}`, string(payload))
}

func TestShopEquipmentOverviewResponseEmptyArrays(t *testing.T) {
	result := ShopEquipmentOverviewResponse{Shops: []ShopEquipmentOverview{{
		ID: "shop-1", Name: "Empty Shop", Role: "admin",
		Equipment: []ShopEquipmentSummary{},
	}}}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"equipment":[]`)
	require.NotContains(t, string(payload), `"equipment":null`)
}

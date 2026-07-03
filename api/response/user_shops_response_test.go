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

func TestShopAggregateResponseDTOsEncodeEmptyArrays(t *testing.T) {
	result := ShopListsWithItemsResponse{
		Lists: []ShopListWithItems{{
			ShopListWithUsername: ShopListWithUsername{
				ID: "list-1", ShopID: "shop-1", CreatedBy: "user-1", Description: "Parts",
			},
			Items: []ShopListItemWithUsername{},
		}},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"lists":[`)
	require.Contains(t, string(payload), `"items":[]`)
}

func TestShopBootstrapResponseDTOsEncodeCounts(t *testing.T) {
	result := ShopsBootstrapResponse{
		Shops: []ShopBootstrapSummary{{
			ID: "shop-1", Name: "Alpha", Role: "admin", IsAdmin: true,
			Settings:  ShopAggregateSettings{AdminOnlyLists: true},
			Counts:    ShopAggregateCounts{Members: 2, Vehicles: 3},
			Equipment: []ShopEquipmentSummary{},
		}},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"admin_only_lists":true`)
	require.Contains(t, string(payload), `"members":2`)
	require.Contains(t, string(payload), `"equipment":[]`)
}

func TestVehicleMaintenanceSnapshotResponseCountsOnlyIncludesMaintenanceFields(t *testing.T) {
	result := VehicleMaintenanceSnapshotResponse{
		Counts: VehicleMaintenanceSnapshotCounts{
			Notifications:     1,
			NotificationItems: 2,
			RecentChanges:     3,
			Services:          4,
		},
	}

	payload, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded map[string]any
	err = json.Unmarshal(payload, &decoded)
	require.NoError(t, err)

	counts := decoded["counts"].(map[string]any)
	require.Equal(t, map[string]any{
		"notifications":      float64(1),
		"notification_items": float64(2),
		"recent_changes":     float64(3),
		"services":           float64(4),
	}, counts)
}

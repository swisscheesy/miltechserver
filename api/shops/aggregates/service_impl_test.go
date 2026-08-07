package aggregates

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type authStubForService struct {
	requireShopMemberErr error
}

func (a authStubForService) IsUserMemberOfShop(*bootstrap.User, string) (bool, error) {
	return false, errors.New("unexpected IsUserMemberOfShop call")
}

func (a authStubForService) IsUserShopAdmin(*bootstrap.User, string) (bool, error) {
	return false, errors.New("unexpected IsUserShopAdmin call")
}

func (a authStubForService) GetUserRoleInShop(*bootstrap.User, string) (string, error) {
	return "", errors.New("unexpected GetUserRoleInShop call")
}

func (a authStubForService) CanUserModifyVehicle(*bootstrap.User, string) (bool, error) {
	return false, errors.New("unexpected CanUserModifyVehicle call")
}

func (a authStubForService) CanUserModifyList(*bootstrap.User, string) (bool, error) {
	return false, errors.New("unexpected CanUserModifyList call")
}

func (a authStubForService) CanUserModifyNotification(*bootstrap.User, string) (bool, error) {
	return false, errors.New("unexpected CanUserModifyNotification call")
}

func (a authStubForService) RequireShopMember(*bootstrap.User, string) error {
	return a.requireShopMemberErr
}

func (a authStubForService) RequireShopAdmin(*bootstrap.User, string) error {
	return errors.New("unexpected RequireShopAdmin call")
}

type repositoryStubForService struct {
	listsResp            []response.ShopListWithItems
	listsErr             error
	vehicleResp          *model.ShopVehicle
	vehicleErr           error
	notifications        []response.VehicleNotificationWithItems
	changes              []response.NotificationChangeWithUsername
	services             []response.EquipmentServiceResponse
	equipmentHistoryResp []response.EquipmentWithPmcsHistory
	equipmentHistoryErr  error
}

func (r repositoryStubForService) GetListsWithItems(context.Context, *bootstrap.User, string, ListTreeLimits) ([]response.ShopListWithItems, error) {
	return r.listsResp, r.listsErr
}

func (r repositoryStubForService) GetVehicleByIDForMember(context.Context, *bootstrap.User, string) (*model.ShopVehicle, error) {
	return r.vehicleResp, r.vehicleErr
}

func (r repositoryStubForService) GetVehicleNotificationsWithItems(context.Context, string, SnapshotLimits) ([]response.VehicleNotificationWithItems, error) {
	return r.notifications, nil
}

func (r repositoryStubForService) GetVehicleRecentChanges(context.Context, string, int) ([]response.NotificationChangeWithUsername, error) {
	return r.changes, nil
}

func (r repositoryStubForService) GetVehicleServices(context.Context, string, int) ([]response.EquipmentServiceResponse, error) {
	return r.services, nil
}

func (r repositoryStubForService) GetShopSnapshot(context.Context, *bootstrap.User, string, ShopSnapshotOptions) (*response.ShopSnapshotResponse, error) {
	return nil, errors.New("unexpected GetShopSnapshot call")
}

func (r repositoryStubForService) GetBootstrap(context.Context, *bootstrap.User, BootstrapOptions) ([]response.ShopBootstrapSummary, error) {
	return nil, errors.New("unexpected GetBootstrap call")
}

func (r repositoryStubForService) GetEquipmentPmcsHistory(context.Context, *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error) {
	return r.equipmentHistoryResp, r.equipmentHistoryErr
}

func TestGetListsWithItemsMapsOnlyAccessDeniedAuthErrorsToAccessDenied(t *testing.T) {
	service := NewService(
		repositoryStubForService{},
		authStubForService{requireShopMemberErr: shared.ErrShopAccessDenied},
	)

	_, err := service.GetListsWithItems(context.Background(), &bootstrap.User{UserID: "user-1"}, "shop-1", ListTreeLimits{})

	require.ErrorIs(t, err, ErrAccessDenied)
	require.NotErrorIs(t, err, ErrAggregateUnavailable)
}

func TestGetListsWithItemsMapsUnexpectedAuthErrorsToAggregateUnavailable(t *testing.T) {
	service := NewService(
		repositoryStubForService{},
		authStubForService{requireShopMemberErr: errors.New("membership lookup failed")},
	)

	_, err := service.GetListsWithItems(context.Background(), &bootstrap.User{UserID: "user-1"}, "shop-1", ListTreeLimits{})

	require.ErrorIs(t, err, ErrAggregateUnavailable)
	require.NotErrorIs(t, err, ErrAccessDenied)
}

func TestGetShopSnapshotMapsUnexpectedAuthErrorsToAggregateUnavailable(t *testing.T) {
	service := NewService(
		repositoryStubForService{},
		authStubForService{requireShopMemberErr: errors.New("membership lookup failed")},
	)

	_, err := service.GetShopSnapshot(context.Background(), &bootstrap.User{UserID: "user-1"}, "shop-1", ShopSnapshotOptions{})

	require.ErrorIs(t, err, ErrAggregateUnavailable)
	require.NotErrorIs(t, err, ErrAccessDenied)
}

func TestGetVehicleMaintenanceSnapshotMapsVehicleAccessDenied(t *testing.T) {
	service := NewService(
		repositoryStubForService{vehicleErr: shared.ErrVehicleAccessDenied},
		authStubForService{},
	)

	_, err := service.GetVehicleMaintenanceSnapshot(context.Background(), &bootstrap.User{UserID: "user-1"}, "vehicle-1", SnapshotLimits{})

	require.ErrorIs(t, err, ErrAccessDenied)
	require.NotErrorIs(t, err, ErrAggregateUnavailable)
}

func TestGetVehicleMaintenanceSnapshotMapsVehicleRepositoryFailureToAggregateUnavailable(t *testing.T) {
	service := NewService(
		repositoryStubForService{vehicleErr: errors.New("vehicle lookup failed")},
		authStubForService{},
	)

	_, err := service.GetVehicleMaintenanceSnapshot(context.Background(), &bootstrap.User{UserID: "user-1"}, "vehicle-1", SnapshotLimits{})

	require.ErrorIs(t, err, ErrAggregateUnavailable)
	require.NotErrorIs(t, err, ErrAccessDenied)
}

func TestGetVehicleMaintenanceSnapshotHandlesNilVehicle(t *testing.T) {
	service := NewService(repositoryStubForService{}, authStubForService{})

	var err error
	require.NotPanics(t, func() {
		_, err = service.GetVehicleMaintenanceSnapshot(context.Background(), &bootstrap.User{UserID: "user-1"}, "vehicle-1", SnapshotLimits{})
	})
	require.ErrorIs(t, err, ErrAggregateUnavailable)
}

func TestGetEquipmentPmcsHistoryRequiresAuth(t *testing.T) {
	service := NewService(repositoryStubForService{}, authStubForService{})

	_, err := service.GetEquipmentPmcsHistory(context.Background(), nil)

	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestGetEquipmentPmcsHistoryReturnsRepositoryResult(t *testing.T) {
	guideManual := "pmcs_sbs/hmmwv/file.json"
	expected := []response.EquipmentWithPmcsHistory{
		{
			ShopEquipmentSummary: response.ShopEquipmentSummary{ID: "vehicle-1", Admin: "A1", Model: "M1", Serial: "S1", Niin: "N1"},
			ShopID:               "shop-1",
			HistoricalPmcs:       []response.PmcsHistorySummary{{SourceType: "guide", GuideManual: &guideManual}},
		},
	}
	service := NewService(repositoryStubForService{equipmentHistoryResp: expected}, authStubForService{})

	result, err := service.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Count)
	require.Equal(t, expected, result.Equipment)
}

func TestPmcsHistorySummaryJSONOmitsSourceInapplicableFields(t *testing.T) {
	guideManual := "pmcs_sbs/hmmwv/file.json"
	guidePayload := marshalPmcsHistorySummary(t, response.PmcsHistorySummary{
		SourceType:  "guide",
		GuideManual: &guideManual,
	})

	require.Equal(t, "guide", guidePayload["source_type"])
	require.Equal(t, guideManual, guidePayload["guide_manual"])
	require.NotContains(t, guidePayload, "custom_checklist_id")
	require.NotContains(t, guidePayload, "custom_revision_id")
	require.NotContains(t, guidePayload, "custom_revision_number")
	require.NotContains(t, guidePayload, "custom_checklist_name")

	checklistID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	revisionID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	revisionNumber := int32(0)
	checklistName := "Device Checklist"
	customPayload := marshalPmcsHistorySummary(t, response.PmcsHistorySummary{
		SourceType:           "custom",
		CustomChecklistID:    &checklistID,
		CustomRevisionID:     &revisionID,
		CustomRevisionNumber: &revisionNumber,
		CustomChecklistName:  &checklistName,
	})

	require.Equal(t, "custom", customPayload["source_type"])
	require.NotContains(t, customPayload, "guide_manual")
	require.Equal(t, checklistID.String(), customPayload["custom_checklist_id"])
	require.Equal(t, revisionID.String(), customPayload["custom_revision_id"])
	require.Equal(t, float64(0), customPayload["custom_revision_number"])
	require.Equal(t, checklistName, customPayload["custom_checklist_name"])
}

func marshalPmcsHistorySummary(t *testing.T, summary response.PmcsHistorySummary) map[string]interface{} {
	t.Helper()

	payload, err := json.Marshal(summary)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &decoded))
	return decoded
}

func TestGetEquipmentPmcsHistoryNormalizesNilSlices(t *testing.T) {
	service := NewService(repositoryStubForService{equipmentHistoryResp: []response.EquipmentWithPmcsHistory{
		{ShopEquipmentSummary: response.ShopEquipmentSummary{ID: "vehicle-1"}, ShopID: "shop-1", HistoricalPmcs: nil},
	}}, authStubForService{})

	result, err := service.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})

	require.NoError(t, err)
	require.NotNil(t, result.Equipment[0].HistoricalPmcs)
	require.Empty(t, result.Equipment[0].HistoricalPmcs)
}

func TestGetEquipmentPmcsHistoryWrapsRepositoryError(t *testing.T) {
	service := NewService(repositoryStubForService{equipmentHistoryErr: errors.New("db exploded")}, authStubForService{})

	_, err := service.GetEquipmentPmcsHistory(context.Background(), &bootstrap.User{UserID: "user-1"})

	require.ErrorIs(t, err, ErrAggregateUnavailable)
}

package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"
	"sort"
)

type AtomicSaver interface {
	SaveAtomic(context.Context, string, request.NotificationSaveRequest) (response.NotificationSaveReceipt, error)
}

func invalidSave() error {
	return &shared.Failure{Code: "invalid", PublicMessage: "Invalid notification save", Status: 400}
}

func ValidateNotificationSave(r request.NotificationSaveRequest) error {
	validID := func(s string) bool { id, err := uuid.Parse(s); return err == nil && id != uuid.Nil && id.String() == s }
	if !validID(r.OperationID) || !validID(r.ShopID) || !validID(r.VehicleID) || (r.NotificationID != nil && !validID(*r.NotificationID)) {
		return invalidSave()
	}
	if shared.ValidateNotificationFields(r.Details.Title, r.Details.Type) != nil {
		return invalidSave()
	}
	switch r.Attachment.Intent {
	case "keep", "remove":
		if r.Attachment.ListID != nil {
			return invalidSave()
		}
	case "attach":
		if r.Attachment.ListID == nil || !validID(*r.Attachment.ListID) {
			return invalidSave()
		}
	default:
		return invalidSave()
	}
	if r.Items == nil {
		return invalidSave()
	}
	seen := map[string]bool{}
	for _, item := range r.Items {
		if !validID(item.ID) || seen[item.ID] || shared.ValidateItemFields(item.Niin, item.Nomenclature, item.Quantity) != nil || !shared.ValidNotificationItemFields(item.Nickname, item.UnitOfMeasure) {
			return invalidSave()
		}
		seen[item.ID] = true
	}
	return nil
}

// FingerprintNotificationSave hashes semantic values, not JSON formatting or
// operation identity. Validation is separate so this function remains pure.
func FingerprintNotificationSave(r request.NotificationSaveRequest) ([32]byte, error) {
	payload := struct {
		ShopID         string                             `json:"shop_id"`
		VehicleID      string                             `json:"vehicle_id"`
		NotificationID *string                            `json:"notification_id"`
		Details        request.NotificationSaveDetails    `json:"details"`
		Attachment     request.NotificationSaveAttachment `json:"attachment"`
		Items          []request.NotificationSaveItem     `json:"items"`
	}{r.ShopID, r.VehicleID, r.NotificationID, r.Details, r.Attachment, append([]request.NotificationSaveItem{}, r.Items...)}
	sort.Slice(payload.Items, func(i, j int) bool { return payload.Items[i].ID < payload.Items[j].ID })
	data, err := json.Marshal(payload)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}

func (service *ServiceImpl) SaveAtomic(ctx context.Context, userID string, r request.NotificationSaveRequest) (response.NotificationSaveReceipt, error) {
	saver, ok := service.repo.(AtomicSaver)
	if !ok {
		return response.NotificationSaveReceipt{}, &shared.Failure{
			Code: "unsupported_contract", PublicMessage: "Notification save unavailable", Status: 503,
		}
	}
	return saver.SaveAtomic(ctx, userID, r)
}

func (handler *Handler) SaveAtomic(c *gin.Context) {
	value, _ := c.Get("user")
	user, ok := value.(*bootstrap.User)
	if !ok || user == nil || user.UserID == "" {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !shared.RequireContract2(c) {
		return
	}
	var r request.NotificationSaveRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&r); err != nil {
		shared.WriteValidationError(c, "Invalid notification save")
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		shared.WriteValidationError(c, "Invalid notification save")
		return
	}
	if err := ValidateNotificationSave(r); err != nil {
		shared.WriteValidationError(c, "Invalid notification save")
		return
	}
	saver, ok := handler.service.(AtomicSaver)
	if !ok || handler.atomicReady == nil || !handler.atomicReady(c.Request.Context()) {
		shared.WriteFailure(c, &shared.Failure{Code: "unsupported_contract", PublicMessage: "Notification save unavailable", Status: 503}, 503, "Notification save unavailable")
		return
	}
	receipt, err := saver.SaveAtomic(c.Request.Context(), user.UserID, r)
	if err != nil {
		shared.WriteFailure(c, shared.ClassifyFailure(err), 500, "Unable to save notification")
		return
	}
	response.OK(c, receipt)
}

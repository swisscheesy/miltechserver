package items

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

// Shop List Item Operations

// AddListItem adds an item to a shop list
func (handler *Handler) AddListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	item := model.ShopListItems{
		ListID:        req.ListID,
		Niin:          req.Niin,
		Nomenclature:  req.Nomenclature,
		Quantity:      req.Quantity,
		Nickname:      req.Nickname,
		UnitOfMeasure: req.UnitOfMeasure,
	}

	service := handler.service
	createdItem, err := service.AddListItem(c.Request.Context(), user, item)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Item added successfully",
		Data:    *createdItem,
	})
}

// GetListItems returns all items for a list with added by usernames
func (handler *Handler) GetListItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		response.Error(c, 400, "list_id is required")
		return
	}

	service := handler.service
	items, err := service.GetListItems(c.Request.Context(), user, listID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, items)
}

// UpdateListItem updates an existing list item
func (handler *Handler) UpdateListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	item := model.ShopListItems{
		ID:            req.ItemID,
		Niin:          req.Niin,
		Nomenclature:  req.Nomenclature,
		Quantity:      req.Quantity,
		Nickname:      req.Nickname,
		UnitOfMeasure: req.UnitOfMeasure,
	}

	service := handler.service
	err := service.UpdateListItem(c.Request.Context(), user, item)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Item updated successfully"})
}

// RemoveListItem removes an item from a list
func (handler *Handler) RemoveListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	service := handler.service
	err := service.RemoveListItem(c.Request.Context(), user, req.ItemID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, gin.H{"message": "Item removed successfully"})
}

// AddListItemBatch adds multiple items to a list
func (handler *Handler) AddListItemBatch(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddListItemBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	var items []model.ShopListItems
	for _, reqItem := range req.Items {
		item := model.ShopListItems{
			ListID:        reqItem.ListID,
			Niin:          reqItem.Niin,
			Nomenclature:  reqItem.Nomenclature,
			Quantity:      reqItem.Quantity,
			Nickname:      reqItem.Nickname,
			UnitOfMeasure: reqItem.UnitOfMeasure,
		}
		items = append(items, item)
	}

	service := handler.service
	createdItems, err := service.AddListItemBatch(c.Request.Context(), user, items)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw StandardResponse literal: response.OK hardcodes an empty
	// Message and has no parameter to carry this success text.
	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Items added successfully",
		Data:    createdItems,
	})
}

// RemoveListItemBatch removes multiple items from lists
func (handler *Handler) RemoveListItemBatch(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		response.Error(c, 401, "unauthorized")
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveListItemBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		response.Error(c, 400, "invalid request")
		return
	}

	service := handler.service
	err := service.RemoveListItemBatch(c.Request.Context(), user, req.ItemIDs)
	if err != nil {
		c.Error(err)
		return
	}

	// Kept as a raw gin.H{} response: flat multi-field body ("message" +
	// "count") that response.OK()'s single data field cannot represent
	// without nesting it under "data", changing this response's shape.
	c.JSON(200, gin.H{"message": "Items removed successfully", "count": len(req.ItemIDs)})
}

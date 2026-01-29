package controller

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// Shop List Item Operations

// AddListItem adds an item to a shop list
func (controller *ShopsController) AddListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddListItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
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

	createdItem, err := controller.ShopsService.AddListItem(user, item)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Item added successfully",
		Data:    *createdItem,
	})
}

// GetListItems returns all items for a list with added by usernames
func (controller *ShopsController) GetListItems(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	listID := c.Param("list_id")
	if listID == "" {
		c.JSON(400, gin.H{"message": "list_id is required"})
		return
	}

	items, err := controller.ShopsService.GetListItems(user, listID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, response.StandardResponse{
		Status:  200,
		Message: "",
		Data:    items,
	})
}

// UpdateListItem updates an existing list item
func (controller *ShopsController) UpdateListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.UpdateListItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
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

	err := controller.ShopsService.UpdateListItem(user, item)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item updated successfully"})
}

// RemoveListItem removes an item from a list
func (controller *ShopsController) RemoveListItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveListItemRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveListItem(user, req.ItemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Item removed successfully"})
}

// AddListItemBatch adds multiple items to a list
func (controller *ShopsController) AddListItemBatch(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.AddListItemBatchRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
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

	createdItems, err := controller.ShopsService.AddListItemBatch(user, items)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(201, response.StandardResponse{
		Status:  201,
		Message: "Items added successfully",
		Data:    createdItems,
	})
}

// RemoveListItemBatch removes multiple items from lists
func (controller *ShopsController) RemoveListItemBatch(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var req request.RemoveListItemBatchRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.ShopsService.RemoveListItemBatch(user, req.ItemIDs)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(200, gin.H{"message": "Items removed successfully", "count": len(req.ItemIDs)})
}

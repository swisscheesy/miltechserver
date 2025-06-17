package controller

import (
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/api/service"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

type UserVehicleController struct {
	UserVehicleService service.UserVehicleService
}

func NewUserVehicleController(userVehicleService service.UserVehicleService) *UserVehicleController {
	return &UserVehicleController{UserVehicleService: userVehicleService}
}

// User Vehicle Operations

func (controller *UserVehicleController) GetUserVehiclesByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserVehicleService.GetUserVehiclesByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetUserVehicleById(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleId := c.Param("vehicleId")
	if vehicleId == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetUserVehicleById(user, vehicleId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) UpsertUserVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var vehicle model.UserVehicle

	if err := c.BindJSON(&vehicle); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserVehicleService.UpsertUserVehicle(user, vehicle)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteUserVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleId := c.Param("vehicleId")
	if vehicleId == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteUserVehicle(user, vehicleId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteAllUserVehicles(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	err := controller.UserVehicleService.DeleteAllUserVehicles(user)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

// User Vehicle Notifications Operations

func (controller *UserVehicleController) GetVehicleNotificationsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserVehicleService.GetVehicleNotificationsByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetVehicleNotificationsByVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleId := c.Param("vehicleId")
	if vehicleId == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetVehicleNotificationsByVehicle(user, vehicleId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetVehicleNotificationById(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationId := c.Param("notificationId")
	if notificationId == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetVehicleNotificationById(user, notificationId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) UpsertVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var notification model.UserVehicleNotifications

	if err := c.BindJSON(&notification); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserVehicleService.UpsertVehicleNotification(user, notification)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteVehicleNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationId := c.Param("notificationId")
	if notificationId == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteVehicleNotification(user, notificationId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteAllVehicleNotificationsByVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleId := c.Param("vehicleId")
	if vehicleId == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteAllVehicleNotificationsByVehicle(user, vehicleId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

// User Vehicle Comments Operations

func (controller *UserVehicleController) GetVehicleCommentsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserVehicleService.GetVehicleCommentsByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetVehicleCommentsByVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleId := c.Param("vehicleId")
	if vehicleId == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetVehicleCommentsByVehicle(user, vehicleId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetVehicleCommentsByNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationId := c.Param("notificationId")
	if notificationId == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetVehicleCommentsByNotification(user, notificationId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetVehicleCommentById(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	commentId := c.Param("commentId")
	if commentId == "" {
		c.JSON(400, gin.H{"message": "comment ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetVehicleCommentById(user, commentId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) UpsertVehicleComment(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var comment model.UserVehicleComments

	if err := c.BindJSON(&comment); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserVehicleService.UpsertVehicleComment(user, comment)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteVehicleComment(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	commentId := c.Param("commentId")
	if commentId == "" {
		c.JSON(400, gin.H{"message": "comment ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteVehicleComment(user, commentId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteAllVehicleCommentsByVehicle(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	vehicleId := c.Param("vehicleId")
	if vehicleId == "" {
		c.JSON(400, gin.H{"message": "vehicle ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteAllVehicleCommentsByVehicle(user, vehicleId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

// User Notification Items Operations

func (controller *UserVehicleController) GetNotificationItemsByUser(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	result, err := controller.UserVehicleService.GetNotificationItemsByUser(user)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetNotificationItemsByNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationId := c.Param("notificationId")
	if notificationId == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetNotificationItemsByNotification(user, notificationId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) GetNotificationItemById(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	itemId := c.Param("itemId")
	if itemId == "" {
		c.JSON(400, gin.H{"message": "item ID is required"})
		return
	}

	result, err := controller.UserVehicleService.GetNotificationItemById(user, itemId)

	if err != nil {
		c.JSON(404, response.EmptyResponseMessage())
	} else {
		c.JSON(200, response.StandardResponse{
			Status:  200,
			Message: "",
			Data:    result,
		})
	}
}

func (controller *UserVehicleController) UpsertNotificationItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var item model.UserNotificationItems

	if err := c.BindJSON(&item); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserVehicleService.UpsertNotificationItem(user, item)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) UpsertNotificationItemList(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	var items []model.UserNotificationItems

	if err := c.BindJSON(&items); err != nil {
		slog.Info("invalid request", "error", err)
		c.JSON(400, gin.H{"message": "invalid request"})
		return
	}

	err := controller.UserVehicleService.UpsertNotificationItemList(user, items)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteNotificationItem(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	itemId := c.Param("itemId")
	if itemId == "" {
		c.JSON(400, gin.H{"message": "item ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteNotificationItem(user, itemId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

func (controller *UserVehicleController) DeleteAllNotificationItemsByNotification(c *gin.Context) {
	ctxUser, ok := c.Get("user")
	user, _ := ctxUser.(*bootstrap.User)

	if !ok {
		c.JSON(401, gin.H{"message": "unauthorized"})
		slog.Info("Unauthorized request")
		return
	}

	notificationId := c.Param("notificationId")
	if notificationId == "" {
		c.JSON(400, gin.H{"message": "notification ID is required"})
		return
	}

	err := controller.UserVehicleService.DeleteAllNotificationItemsByNotification(user, notificationId)

	if err != nil {
		c.Error(err)
	} else {
		c.Status(200)
	}
}

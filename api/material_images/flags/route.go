package flags

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"miltechserver/api/material_images/images"
	"miltechserver/api/material_images/shared"
	"miltechserver/api/request"
	"miltechserver/api/response"
)

type Handler struct {
	service       Service
	imagesService images.Service
}

func RegisterRoutes(authRouter *gin.RouterGroup, service Service, imagesService images.Service) {
	handler := Handler{service: service, imagesService: imagesService}

	authRouter.POST("/material-images/:image_id/flag", handler.flag)
	authRouter.GET("/material-images/:image_id/flags", handler.getFlags)
}

func (h *Handler) flag(c *gin.Context) {
	user, err := shared.GetUserFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	imageID := c.Param("image_id")

	var req request.FlagImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	err = h.service.Flag(user, imageID, req.Reason, req.Description)
	if err != nil {
		if err.Error() == "you have already flagged this image" {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	updatedImage, err := h.imagesService.GetByID(imageID, user)
	var flagCount int
	var isFlagged bool
	if err == nil {
		flagCount = 1
		isFlagged = updatedImage.IsFlagged
	}

	c.JSON(http.StatusOK, response.ImageFlagResponse{
		Success:   true,
		Message:   "Image flagged successfully",
		FlagCount: flagCount,
		IsFlagged: isFlagged,
	})
}

func (h *Handler) getFlags(c *gin.Context) {
	imageID := c.Param("image_id")

	flags, err := h.service.GetByImage(imageID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve flags")
		return
	}

	// NOTE: intentionally NOT migrated to response.OK() here. Wrapping this
	// payload in the standard envelope would move "flags" from the response
	// body's top level to a nested "data.flags", which is a response-shape
	// change: tests/material_images/handlers_test.go:101-104 unmarshals the
	// raw body into map[string]interface{} and asserts flagsPayload["flags"]
	// directly at the top level. Confirmed by running the test with the
	// migration applied (fails at handlers_test.go:104). Left as a raw
	// c.JSON call, matching the ps_mag precedent from Task 6 (a single call
	// site deliberately excluded from a mechanical helper substitution to
	// preserve its existing response body) — flagged for explicit approval
	// per this task's brief rather than silently updated.
	c.JSON(http.StatusOK, gin.H{"flags": flags})
}

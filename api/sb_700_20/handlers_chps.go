package sb_700_20

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listChp4(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetChp4Paginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchChp4(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	item, err := h.service.GetChp4ByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}

func (h *Handler) listChp6(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetChp6Paginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchChp6(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetChp6ByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) listChp8(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetChp8Paginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchChp8(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetChp8ByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

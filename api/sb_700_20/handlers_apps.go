package sb_700_20

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"miltechserver/api/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listAppB(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppBPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppB(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetAppBByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) listAppC(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppCPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppC(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	item, err := h.service.GetAppCByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}

func (h *Handler) listAppD(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppDPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppD(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetAppDByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) listAppE(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppEPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppE(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetAppEByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) listAppF(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppFPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppF(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	item, err := h.service.GetAppFByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}

func (h *Handler) listAppG(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppGPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppG(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	item, err := h.service.GetAppGByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}

func (h *Handler) listAppH1(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppH1Paginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppH1(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetAppH1ByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) listAppH2(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppH2Paginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppH2(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	items, err := h.service.GetAppH2ByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) listAppI(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppIPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppI(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	item, err := h.service.GetAppIByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}

func (h *Handler) listAppJ(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAppJPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: data})
}

func (h *Handler) searchAppJ(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		return
	}
	item, err := h.service.GetAppJByLIN(lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: item})
}

func (h *Handler) searchAppEByNewLIN(c *gin.Context) {
	newLin := c.Param("new_lin")
	if strings.TrimSpace(newLin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		return
	}
	items, err := h.service.GetAppEByNewLIN(newLin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchAppGByNewLIN(c *gin.Context) {
	newLin := c.Param("new_lin")
	if strings.TrimSpace(newLin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		return
	}
	items, err := h.service.GetAppGByNewLIN(newLin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchAppH1BySubLIN(c *gin.Context) {
	sublin := c.Param("sublin")
	if strings.TrimSpace(sublin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		return
	}
	items, err := h.service.GetAppH1BySubLIN(sublin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchAppH2BySubLIN(c *gin.Context) {
	sublin := c.Param("sublin")
	if strings.TrimSpace(sublin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		return
	}
	items, err := h.service.GetAppH2BySubLIN(sublin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

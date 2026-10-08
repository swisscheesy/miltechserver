package sb_700_20

import (
	"errors"
	"net/http"
	"strings"

	"miltechserver/api/response"
	"miltechserver/api/shared/pagination"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listAppB(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppBPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppB(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetAppBByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) listAppC(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppCPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppC(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	item, err := h.service.GetAppCByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, item)
}

func (h *Handler) listAppD(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppDPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppD(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetAppDByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) listAppE(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppEPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppE(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetAppEByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) listAppF(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppFPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppF(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	item, err := h.service.GetAppFByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, item)
}

func (h *Handler) listAppG(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppGPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppG(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	item, err := h.service.GetAppGByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, item)
}

func (h *Handler) listAppH1(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppH1Paginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppH1(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetAppH1ByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) listAppH2(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppH2Paginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppH2(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetAppH2ByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) listAppI(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppIPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppI(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	item, err := h.service.GetAppIByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, item)
}

func (h *Handler) listAppJ(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetAppJPaginated(c.Request.Context(), page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrInvalidPage) {
			response.Error(c, http.StatusBadRequest, "Invalid page number")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) searchAppJ(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	item, err := h.service.GetAppJByLIN(c.Request.Context(), lin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, item)
}

func (h *Handler) searchAppEByNewLIN(c *gin.Context) {
	newLin := c.Param("new_lin")
	if strings.TrimSpace(newLin) == "" {
		response.Error(c, http.StatusBadRequest, "new_lin parameter is required")
		return
	}
	items, err := h.service.GetAppEByNewLIN(c.Request.Context(), newLin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "new_lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) searchAppGByNewLIN(c *gin.Context) {
	newLin := c.Param("new_lin")
	if strings.TrimSpace(newLin) == "" {
		response.Error(c, http.StatusBadRequest, "new_lin parameter is required")
		return
	}
	items, err := h.service.GetAppGByNewLIN(c.Request.Context(), newLin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "new_lin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) searchAppH1BySubLIN(c *gin.Context) {
	sublin := c.Param("sublin")
	if strings.TrimSpace(sublin) == "" {
		response.Error(c, http.StatusBadRequest, "sublin parameter is required")
		return
	}
	items, err := h.service.GetAppH1BySubLIN(c.Request.Context(), sublin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "sublin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) searchAppH2BySubLIN(c *gin.Context) {
	sublin := c.Param("sublin")
	if strings.TrimSpace(sublin) == "" {
		response.Error(c, http.StatusBadRequest, "sublin parameter is required")
		return
	}
	items, err := h.service.GetAppH2BySubLIN(c.Request.Context(), sublin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "sublin parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

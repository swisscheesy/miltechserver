package sb_700_20

import (
	"errors"
	"net/http"
	"strings"

	"miltechserver/api/response"
	"miltechserver/api/shared/pagination"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listChp4(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetChp4Paginated(c.Request.Context(), page)
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

func (h *Handler) searchChp4(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	item, err := h.service.GetChp4ByLIN(c.Request.Context(), lin)
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

func (h *Handler) listChp6(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetChp6Paginated(c.Request.Context(), page)
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

func (h *Handler) searchChp6(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetChp6ByLIN(c.Request.Context(), lin)
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

func (h *Handler) listChp8(c *gin.Context) {
	page, ok := pagination.ParsePage(c)
	if !ok {
		return
	}
	data, err := h.service.GetChp8Paginated(c.Request.Context(), page)
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

func (h *Handler) searchChp8(c *gin.Context) {
	lin := c.Param("lin")
	if strings.TrimSpace(lin) == "" {
		response.Error(c, http.StatusBadRequest, "lin parameter is required")
		return
	}
	items, err := h.service.GetChp8ByLIN(c.Request.Context(), lin)
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

func (h *Handler) searchChp4ByRIC(c *gin.Context) {
	ric := c.Param("ric")
	if strings.TrimSpace(ric) == "" {
		response.Error(c, http.StatusBadRequest, "ric parameter is required")
		return
	}
	items, err := h.service.GetChp4ByRIC(c.Request.Context(), ric)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "ric parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) searchChp6ByRIC(c *gin.Context) {
	ric := c.Param("ric")
	if strings.TrimSpace(ric) == "" {
		response.Error(c, http.StatusBadRequest, "ric parameter is required")
		return
	}
	items, err := h.service.GetChp6ByRIC(c.Request.Context(), ric)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "ric parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

func (h *Handler) searchChp8ByRIC(c *gin.Context) {
	ric := c.Param("ric")
	if strings.TrimSpace(ric) == "" {
		response.Error(c, http.StatusBadRequest, "ric parameter is required")
		return
	}
	items, err := h.service.GetChp8ByRIC(c.Request.Context(), ric)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			response.Error(c, http.StatusBadRequest, "ric parameter is required")
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	response.OK(c, items)
}

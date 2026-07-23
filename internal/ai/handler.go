package ai

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/ruhulamin1398/ai-backend/internal/common"
)

// Handler handles HTTP requests for the AI proxy.
type Handler struct {
	svc *Service
	val *validator.Validate
}

// NewHandler creates a new AI handler.
func NewHandler(svc *Service, val *validator.Validate) *Handler {
	return &Handler{svc: svc, val: val}
}

// Proxy handles POST /api/v1/ai/proxy.
func (h *Handler) Proxy(c *gin.Context) {
	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.val.Struct(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.svc.Proxy(c.Request.Context(), &req)
	if err != nil {
		var ssrfErr *SSRFError
		if errors.As(err, &ssrfErr) {
			common.Fail(c, http.StatusBadRequest, "SSRF_BLOCKED", err.Error())
			return
		}
		common.Fail(c, http.StatusBadGateway, "PROXY_ERROR", err.Error())
		return
	}

	common.Success(c, http.StatusOK, result)
}

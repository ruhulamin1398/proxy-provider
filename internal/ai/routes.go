package ai

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the proxy route at /proxy (no auth needed).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/proxy", h.Proxy)
}
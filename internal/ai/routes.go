package ai

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all routes on the given engine.
func RegisterRoutes(r *gin.Engine, h *Handler) {
	// Existing proxy endpoint
	r.POST("/proxy", h.Proxy)

	// OpenAI-compatible endpoints
	v1 := r.Group("/v1")
	{
		v1.POST("/chat/completions", h.ChatCompletions)
		v1.GET("/models", h.ListModels)
	}
}
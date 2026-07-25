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
		v1.GET("/models", h.ListFreeModels)
		v1.GET("/props", h.DiscoveryProxy)
	}

	// Also register at the bare path
	r.POST("/chat/completions", h.ChatCompletions)
	r.GET("/models", h.ListFreeModels)

	// ── Hermes / Ollama discovery probes ──
	api := r.Group("/api")
	{
		api.GET("/v1/models", h.ListFreeModels)
		api.GET("/tags", h.DiscoveryProxy)
	}

	r.GET("/props", h.DiscoveryProxy)
	r.GET("/version", h.DiscoveryProxy)
}
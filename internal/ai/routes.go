package ai

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all routes on the given engine.
func RegisterRoutes(r *gin.Engine, h *Handler) {
	// Existing proxy endpoint
	r.POST("/proxy", h.Proxy)

	// OpenAI-compatible endpoints — both /v1/ and / (bare) prefixes
	// for compatibility with different Hermes auxiliary tasks
	v1 := r.Group("/v1")
	{
		v1.POST("/chat/completions", h.ChatCompletions)
		v1.GET("/models", h.DiscoveryProxy)
		v1.GET("/props", h.DiscoveryProxy)
	}

	// Also register at the bare path (Hermes calls without /v1 prefix)
	r.POST("/chat/completions", h.ChatCompletions)
	r.GET("/models", h.DiscoveryProxy)

	// ── Hermes / Ollama discovery probes (proxied to upstream) ──

	api := r.Group("/api")
	{
		api.GET("/v1/models", h.DiscoveryProxy)
		api.GET("/tags", h.DiscoveryProxy)
	}

	r.GET("/props", h.DiscoveryProxy)
	r.GET("/version", h.DiscoveryProxy)
}
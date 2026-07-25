package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes on the given engine.
func RegisterRoutes(r *gin.Engine, h *Handler) {
	// Existing proxy endpoint
	r.POST("/proxy", h.Proxy)

	// OpenAI-compatible endpoints — both /v1/ and / (bare) prefixes
	// for compatibility with different Hermes auxiliary tasks
	v1 := r.Group("/v1")
	{
		v1.POST("/chat/completions", h.ChatCompletions)
		v1.GET("/models", h.ListModels)
		v1.GET("/props", h.Props)
	}

	// Also register at the bare path (Hermes calls without /v1 prefix)
	r.POST("/chat/completions", h.ChatCompletions)
	r.GET("/models", h.ListModels)

	// ── Hermes / Ollama discovery probes ──

	// Ollama-style API
	api := r.Group("/api")
	{
		api.GET("/v1/models", h.ListModels)
		api.GET("/tags", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"models": []interface{}{}})
		})
	}

	// Hermes proprietary probe
	r.GET("/props", h.Props)

	// Version info
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":   "1.0.0",
			"name":      "proxy-provider",
			"platform":  "opencode",
		})
	})
}
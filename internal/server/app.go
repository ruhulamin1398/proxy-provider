package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/ruhulamin1398/ai-backend/internal/ai"
	"github.com/ruhulamin1398/ai-backend/internal/common"
)

// Run starts the HTTP server and blocks until a shutdown signal is received.
func Run() {
	cfg := common.LoadConfig()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	val := validator.New()

	aiSvc := ai.NewService()
	aiHdl := ai.NewHandler(aiSvc, val)

	// Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(common.RequestLogger())
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "ok"}})
	})

	// Log viewer
	r.GET("/logs", func(c *gin.Context) {
		data, err := os.ReadFile(common.LogFile)
		if err != nil {
			c.String(http.StatusOK, "No logs recorded yet.\n")
			return
		}
		// Display logs in reverse chronological order (newest first)
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		var out strings.Builder
		for i := 0; i < len(lines); i++ {
			fmt.Fprintf(&out, "%3d. %s\n", len(lines)-i, lines[i])
		}
		c.String(http.StatusOK, out.String())
	})
	r.GET("/logs/clear", func(c *gin.Context) {
		if err := os.Truncate(common.LogFile, 0); err != nil {
			c.String(http.StatusInternalServerError, "Failed to clear: %v\n", err)
			return
		}
		c.String(http.StatusOK, "Log file cleared.\n")
	})

	// Register all routes (proxy + OpenAI-compatible endpoints)
	ai.RegisterRoutes(r, aiHdl)

	// Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

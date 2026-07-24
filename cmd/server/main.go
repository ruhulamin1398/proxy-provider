package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/ruhulamin1398/ai-backend/internal/ai"
	"github.com/ruhulamin1398/ai-backend/internal/common"
)

func main() {
	cfg := common.LoadConfig()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	val := validator.New()

	aiSvc := ai.NewService()
	aiHdl := ai.NewHandler(aiSvc, val)

	// Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		println("[REQ] " + c.Request.Method + " " + c.Request.URL.Path)
		c.Next()
	})

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "ok"}})
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
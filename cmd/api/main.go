package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"idempotent-webhook-processor/internal/config"
	"idempotent-webhook-processor/internal/domain"
	"idempotent-webhook-processor/internal/handler"
	"idempotent-webhook-processor/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// 1. Initialize Structured Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 2. Load Configs
	cfg := config.Load()

	// 3. Connect to Redis Client Pool
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Quick Ping check to verify connection viability before boot
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("could not establish strong connection link to Redis cache pool", zap.Error(err))
	}

	// 4. Component Wiring (Dependency Injection)
	// Set lock TTL to 5 minutes to prevent race states
	redisRepo := repository.NewRedisRepository(redisClient, 5*time.Minute)

	// Create a worker pool with 3 dedicated workers and a queue limit of 100 jobs
	webhookUC := domain.NewWebhookUseCase(redisRepo, logger, 3, 100)
	webhookHandler := handler.NewWebhookHandler(webhookUC)

	// 5. Initialize Server Router Layout
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/v1/webhooks", webhookHandler.HandleWebhook)

	// 6. Graceful Shutdown Framework Setup
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("booting up microservice container engine", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal("microservice encountered fatal crash crash scenario", zap.Error(err))
		}
	}()

	// Listen for system termination flags
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("initiating graceful exit procedures...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("forced system crash encountered while cutting engine connections", zap.Error(err))
	}

	logger.Info("microservice container closed cleanly")
}

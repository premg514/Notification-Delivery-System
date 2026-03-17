package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-system/internal/api"
	"notification-system/internal/api/handlers"
	"notification-system/internal/api/middleware"
	"notification-system/internal/config"
	"notification-system/internal/observability"
	"notification-system/internal/queue"
	"notification-system/internal/repository/postgres"
	"notification-system/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api service failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	observability.SetupLogger(cfg.LogLevel)

	db, err := postgres.NewDB(cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := postgres.RunMigrations(ctx, db); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	rabbit, err := queue.New(cfg.RabbitMQURL, cfg.FanoutQueueName, cfg.DeliveryQueueName, cfg.DeliveryRetryQueuePrefix)
	if err != nil {
		return fmt.Errorf("rabbitmq connection failed: %w", err)
	}
	defer rabbit.Close()

	publisher, err := queue.NewPublisher(rabbit)
	if err != nil {
		return fmt.Errorf("rabbitmq publisher failed: %w", err)
	}
	defer publisher.Close()

	repo := postgres.NewNotificationRepository(db)
	notificationService := service.NewNotificationService(repo, publisher)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	rateLimiter, err := middleware.NewRateLimiter(cfg.RateLimitPerMinute, cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("redis rate limiter setup failed: %w", err)
	}
	defer rateLimiter.Close()
	router := api.NewRouter(notificationHandler, rateLimiter)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.RequestTimeout,
		WriteTimeout:      cfg.RequestTimeout,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Error("api shutdown error", "error", shutdownErr)
		}
	}()

	slog.Info("api service listening", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("api server error: %w", err)
	}

	return nil
}

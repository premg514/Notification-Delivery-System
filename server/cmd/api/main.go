package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-system/internal/api"
	"notification-system/internal/api/handlers"
	"notification-system/internal/api/middleware"
	"notification-system/internal/config"
	"notification-system/internal/queue"
	"notification-system/internal/repository/postgres"
	"notification-system/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := postgres.NewDB(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := postgres.RunMigrations(ctx, db); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	rabbit, err := queue.New(cfg.RabbitMQURL, cfg.FanoutQueueName, cfg.DeliveryQueueName, cfg.DeliveryRetryQueuePrefix)
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer rabbit.Close()

	publisher, err := queue.NewPublisher(rabbit)
	if err != nil {
		log.Fatalf("rabbitmq publisher failed: %v", err)
	}
	defer publisher.Close()

	repo := postgres.NewNotificationRepository(db)
	notificationService := service.NewNotificationService(repo, publisher)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitPerMinute)
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

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("api shutdown error: %v", err)
		}
	}()

	log.Printf("api service listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("api server error: %v", err)
	}
}

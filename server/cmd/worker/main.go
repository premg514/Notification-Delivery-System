package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"notification-system/internal/config"
	"notification-system/internal/domain/models"
	"notification-system/internal/observability"
	"notification-system/internal/queue"
	"notification-system/internal/repository/postgres"
	"notification-system/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker service failed", "error", err)
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

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

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

	fanoutProcessor := worker.NewFanoutProcessor(repo, publisher, cfg.FanoutBatchSize, cfg.DeliveryBatchSize)
	fanoutPool := worker.NewPool[models.FanoutJob](cfg.FanoutWorkerCount, "fanout", fanoutProcessor)
	fanoutPool.Start(ctx, cfg.FanoutWorkerCount)
	defer fanoutPool.Stop()

	deliveryProcessor := worker.NewDeliveryProcessor(repo, publisher, worker.LoggingSender{}, cfg.MaxRetries, cfg.DeliveryAttemptsPerBatch)
	deliveryPool := worker.NewPool[models.DeliveryBatchJob](cfg.DeliveryWorkerCount, "delivery", deliveryProcessor)
	deliveryPool.Start(ctx, cfg.DeliveryWorkerCount)
	defer deliveryPool.Stop()

	fanoutConsumer, err := queue.NewConsumer[models.FanoutJob](rabbit, rabbit.FanoutQueueName(), cfg.FanoutPrefetchCount, fanoutPool)
	if err != nil {
		return fmt.Errorf("rabbitmq fanout consumer failed: %w", err)
	}
	defer fanoutConsumer.Close()

	deliveryConsumer, err := queue.NewConsumer[models.DeliveryBatchJob](rabbit, rabbit.DeliveryQueueName(), cfg.DeliveryPrefetchCount, deliveryPool)
	if err != nil {
		return fmt.Errorf("rabbitmq delivery consumer failed: %w", err)
	}
	defer deliveryConsumer.Close()

	slog.Info("worker service started", "fanout_workers", cfg.FanoutWorkerCount, "delivery_workers", cfg.DeliveryWorkerCount)

	errCh := make(chan error, 2)
	go func() {
		errCh <- fanoutConsumer.Start(ctx)
	}()
	go func() {
		errCh <- deliveryConsumer.Start(ctx)
	}()

	var runErr error
	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil && ctx.Err() == nil {
			runErr = fmt.Errorf("worker consumer error: %w", err)
			cancel()
		}
	}

	return runErr
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-system/internal/config"
	"notification-system/internal/domain/models"
	"notification-system/internal/queue"
	"notification-system/internal/repository/postgres"
	"notification-system/internal/worker"
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

	fanoutProcessor := worker.NewFanoutProcessor(repo, publisher, cfg.FanoutBatchSize, cfg.DeliveryBatchSize)
	fanoutPool := worker.NewPool[models.FanoutJob](cfg.FanoutWorkerCount, fanoutProcessor)
	fanoutPool.Start(ctx, cfg.FanoutWorkerCount)
	defer fanoutPool.Stop()

	deliveryProcessor := worker.NewDeliveryProcessor(repo, publisher, worker.LoggingSender{}, cfg.MaxRetries, cfg.DeliveryAttemptsPerBatch)
	deliveryPool := worker.NewPool[models.DeliveryBatchJob](cfg.DeliveryWorkerCount, deliveryProcessor)
	deliveryPool.Start(ctx, cfg.DeliveryWorkerCount)
	defer deliveryPool.Stop()

	fanoutConsumer, err := queue.NewConsumer[models.FanoutJob](rabbit, rabbit.FanoutQueueName(), cfg.FanoutPrefetchCount, fanoutPool)
	if err != nil {
		log.Fatalf("rabbitmq fanout consumer failed: %v", err)
	}
	defer fanoutConsumer.Close()

	deliveryConsumer, err := queue.NewConsumer[models.DeliveryBatchJob](rabbit, rabbit.DeliveryQueueName(), cfg.DeliveryPrefetchCount, deliveryPool)
	if err != nil {
		log.Fatalf("rabbitmq delivery consumer failed: %v", err)
	}
	defer deliveryConsumer.Close()

	log.Printf("worker service started with fanout_workers=%d delivery_workers=%d", cfg.FanoutWorkerCount, cfg.DeliveryWorkerCount)

	errCh := make(chan error, 2)
	go func() {
		errCh <- fanoutConsumer.Start(ctx)
	}()
	go func() {
		errCh <- deliveryConsumer.Start(ctx)
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil && ctx.Err() == nil {
			log.Fatalf("worker consumer error: %v", err)
		}
	}
}

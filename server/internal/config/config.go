package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                     string
	LogLevel                 string
	PostgresURL              string
	RabbitMQURL              string
	RedisURL                 string
	FanoutQueueName          string
	DeliveryQueueName        string
	DeliveryRetryQueuePrefix string
	FanoutWorkerCount        int
	DeliveryWorkerCount      int
	FanoutPrefetchCount      int
	DeliveryPrefetchCount    int
	FanoutBatchSize          int
	DeliveryBatchSize        int
	DeliveryAttemptsPerBatch int
	RateLimitPerMinute       int
	RequestTimeout           time.Duration
	MaxRetries               int
}

func Load() Config {
	loadDotEnv()

	return Config{
		Port:                     getEnv("PORT", "8080"),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		PostgresURL:              mustGetEnv("POSTGRES_URL"),
		RabbitMQURL:              mustGetEnv("RABBITMQ_URL"),
		RedisURL:                 getEnv("REDIS_URL", "redis://localhost:6379/0"),
		FanoutQueueName:          getEnv("RABBITMQ_FANOUT_QUEUE", "notifications.fanout"),
		DeliveryQueueName:        getEnv("RABBITMQ_DELIVERY_QUEUE", "notifications.delivery"),
		DeliveryRetryQueuePrefix: getEnv("RABBITMQ_DELIVERY_RETRY_PREFIX", "notifications.delivery.retry"),
		FanoutWorkerCount:        getEnvInt("FANOUT_WORKER_COUNT", 8),
		DeliveryWorkerCount:      getEnvInt("DELIVERY_WORKER_COUNT", 128),
		FanoutPrefetchCount:      getEnvInt("RABBITMQ_FANOUT_PREFETCH_COUNT", 16),
		DeliveryPrefetchCount:    getEnvInt("RABBITMQ_DELIVERY_PREFETCH_COUNT", 256),
		FanoutBatchSize:          getEnvInt("FANOUT_BATCH_SIZE", 5000),
		DeliveryBatchSize:        getEnvInt("DELIVERY_BATCH_SIZE", 500),
		DeliveryAttemptsPerBatch: getEnvInt("DELIVERY_ATTEMPTS_PER_BATCH", 32),
		RateLimitPerMinute:       getEnvInt("RATE_LIMIT_PER_MINUTE", 120),
		RequestTimeout:           time.Duration(getEnvInt("REQUEST_TIMEOUT_SECONDS", 15)) * time.Second,
		MaxRetries:               getEnvInt("MAX_RETRIES", 3),
	}
}

func loadDotEnv() {
	candidates := []string{
		".env",
		filepath.Join("server", ".env"),
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if key != "" && os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}

		return
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func mustGetEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		panic(fmt.Sprintf("missing required environment variable %s", key))
	}
	return value
}

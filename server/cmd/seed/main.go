package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"notification-system/internal/config"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userCount := 100
	log.Printf("Seeding %d users into the database...", userCount)

	for i := 1; i <= userCount; i++ {

		id := service.NewID()

		email := fmt.Sprintf("user_%d_%s@example.com", i, id[:8])
		deviceToken := fmt.Sprintf("device_token_%s", id)

		_, err := db.Exec(ctx, `
			INSERT INTO users (id, email, device_token, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, id, email, deviceToken, time.Now().UTC())

		if err != nil {
			log.Fatalf("failed to insert user %d: %v", i, err)
		}
	}

	log.Println("✅ Successfully seeded 100 users!")
	log.Println("Check your database to grab a few UUIDs for testing.")
}

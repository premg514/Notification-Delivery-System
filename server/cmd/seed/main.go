package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"notification-system/internal/config"
	"notification-system/internal/domain/models"
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

	if err := postgres.RunMigrations(ctx, db); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	userCount := 10000
	departments := models.AllDepartments()
	log.Printf("Seeding %d users into the database...", userCount)

	for i := 1; i <= userCount; i++ {
		id := service.NewID()

		email := fmt.Sprintf("user_%d_%s@example.com", i, id[:8])
		deviceToken := fmt.Sprintf("device_token_%s", id)
		department := departments[(i-1)%len(departments)]

		_, err := db.Exec(ctx, `
			INSERT INTO users (id, email, device_token, department, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO NOTHING
		`, id, email, deviceToken, string(department), time.Now().UTC())
		if err != nil {
			log.Fatalf("failed to insert user %d: %v", i, err)
		}
	}

	log.Printf("Successfully seeded %d users!", userCount)
	log.Println("Check your database to grab a few UUIDs for testing.")
}

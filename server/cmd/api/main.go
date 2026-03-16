package main

import (
	"log"
	"os"

	"notification-system/internal/api"
	"notification-system/internal/config"
	"notification-system/internal/repository/postgres"
	"notification-system/internal/service"
)

func main() {
	if err := config.LoadEnvFile(".env"); err != nil && !os.IsNotExist(err) {
		log.Fatal("Failed to load .env:", err)
	}

	cfg := config.LoadConfig()

	// DB connection
	db, err := postgres.NewDB()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// repository
	repo := postgres.NewNotificationRepository(db)

	// service
	notificationService := service.NewNotificationService(repo)

	// router
	router := api.SetupRouter(notificationService)

	log.Println("API running on port:", cfg.APIPort)

	router.Run(":" + cfg.APIPort)
}

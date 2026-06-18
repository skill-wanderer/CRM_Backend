package main

import (
	"log"
	"os"

	"crm-backend/internal/database"
	"crm-backend/internal/handlers"
	"crm-backend/internal/repositories"
	"crm-backend/internal/routes"
	"crm-backend/internal/services"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (useful for local run, ignored in Docker if missing)
	_ = godotenv.Load()

	// Initialize DB Connection and Auto-Migrate
	database.Connect()

	db := database.DB

	// Repositories
	userRepo := repositories.NewUserRepository(db)
	templateRepo := repositories.NewTemplateRepository(db)
	leadRepo := repositories.NewLeadRepository(db)

	// Services
	authService := services.NewAuthService(userRepo)
	templateService := services.NewTemplateService(templateRepo)
	leadService := services.NewLeadService(leadRepo, templateRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	templateHandler := handlers.NewTemplateHandler(templateService)
	leadHandler := handlers.NewLeadHandler(leadService)

	// Setup Router
	r := routes.SetupRouter(authHandler, templateHandler, leadHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

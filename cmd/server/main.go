package main

import (
	"log"

	adminhandlers "crm-backend/internal/admin/handlers"
	adminrepo "crm-backend/internal/admin/repositories"
	adminsvc "crm-backend/internal/admin/services"
	"crm-backend/internal/auth"
	"crm-backend/internal/config"
	"crm-backend/internal/database"
	"crm-backend/internal/handlers"
	"crm-backend/internal/repositories"
	"crm-backend/internal/routes"
	"crm-backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (useful for local run, ignored in Docker if missing)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	gin.SetMode(cfg.GinMode)

	// Initialize DB Connection and Auto-Migrate
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Database startup failed: %v", err)
	}

	// Repositories
	templateRepo := repositories.NewTemplateRepository(db)
	leadRepo := repositories.NewLeadRepository(db)
	tenantRepo := adminrepo.NewTenantRepository(db)

	// Services
	templateService := services.NewTemplateService(templateRepo)
	leadService := services.NewLeadService(leadRepo, templateRepo)
	tenantService := adminsvc.NewTenantService(tenantRepo)

	// Handlers
	templateHandler := handlers.NewTemplateHandler(templateService)
	leadHandler := handlers.NewLeadHandler(leadService)
	tenantHandler := adminhandlers.NewTenantHandler(tenantService)

	adminVerifier := auth.NewRealmVerifier(cfg.Keycloak.Admin, cfg.Keycloak.TokenSkew, cfg.Keycloak.JWKSCacheTTL)
	clientVerifier := auth.NewRealmVerifier(cfg.Keycloak.Client, cfg.Keycloak.TokenSkew, cfg.Keycloak.JWKSCacheTTL)

	// Setup Router
	r := routes.SetupRouter(routes.Dependencies{
		Config:          cfg,
		DB:              db,
		TemplateHandler: templateHandler,
		LeadHandler:     leadHandler,
		TenantHandler:   tenantHandler,
		AdminVerifier:   adminVerifier,
		ClientVerifier:  clientVerifier,
	})

	log.Printf("Starting server on port %s...", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

package routes

import (
	"net/http"

	"crm-backend/docs"
	"crm-backend/internal/admin"
	adminhandlers "crm-backend/internal/admin/handlers"
	"crm-backend/internal/auth"
	"crm-backend/internal/client"
	"crm-backend/internal/config"
	"crm-backend/internal/handlers"
	"crm-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config          *config.Config
	DB              *gorm.DB
	TemplateHandler *handlers.TemplateHandler
	LeadHandler     *handlers.LeadHandler
	TenantHandler   *adminhandlers.TenantHandler
	AdminVerifier   auth.TokenVerifier
	ClientVerifier  auth.TokenVerifier
}

func SetupRouter(deps Dependencies) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.CORS(deps.Config.CORSAllowedOrigins, deps.Config.Tenancy.Header))
	registerSwagger(r)

	api := r.Group("/api")

	admin.RegisterRoutes(api, deps.TenantHandler, deps.AdminVerifier, deps.Config.Keycloak.AdminRole)
	client.RegisterRoutes(api, deps.ClientVerifier, deps.DB, deps.Config.Tenancy.Header)

	// Templates
	templates := api.Group("/templates")
	{
		templates.POST("", deps.TemplateHandler.CreateTemplate)
		templates.GET("", deps.TemplateHandler.GetTemplates)
		templates.GET("/:id", deps.TemplateHandler.GetTemplate)
		templates.PUT("/:id", deps.TemplateHandler.UpdateTemplate)
		templates.DELETE("/:id", deps.TemplateHandler.DeleteTemplate)

		// Fields — use same :id param to avoid Gin wildcard conflict
		templates.POST("/:id/fields", deps.TemplateHandler.AddField)
		templates.GET("/:id/fields", deps.TemplateHandler.GetFields)
		templates.GET("/:id/schema", deps.TemplateHandler.GetTemplateSchema)
	}

	// Fields directly (for PUT/DELETE as per requirements)
	fields := api.Group("/fields")
	{
		fields.PUT("/:fieldId", deps.TemplateHandler.UpdateField)
		fields.DELETE("/:fieldId", deps.TemplateHandler.DeleteField)
	}

	// Leads
	leads := api.Group("/leads")
	{
		leads.POST("", deps.LeadHandler.CreateLead)
		leads.GET("", deps.LeadHandler.GetLeads)
		leads.GET("/:id", deps.LeadHandler.GetLead)
		leads.PUT("/:id", deps.LeadHandler.UpdateLead)
		leads.DELETE("/:id", deps.LeadHandler.DeleteLead)
	}

	return r
}

func registerSwagger(r *gin.Engine) {
	const specPath = "/openapi.yaml"

	r.GET(specPath, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPIYAML)
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL(specPath),
	))
}

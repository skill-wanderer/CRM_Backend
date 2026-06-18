package routes

import (
	"crm-backend/internal/handlers"
	"crm-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	authHandler *handlers.AuthHandler,
	templateHandler *handlers.TemplateHandler,
	leadHandler *handlers.LeadHandler,
) *gin.Engine {
	r := gin.Default()

	// CORS middleware could be added here

	api := r.Group("/api")

	// Auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		// Templates
		templates := protected.Group("/templates")
		{
			templates.POST("", middleware.RBACMiddleware("admin"), templateHandler.CreateTemplate)
			templates.GET("", templateHandler.GetTemplates)
			templates.GET("/:id", templateHandler.GetTemplate)
			templates.PUT("/:id", middleware.RBACMiddleware("admin"), templateHandler.UpdateTemplate)
			templates.DELETE("/:id", middleware.RBACMiddleware("admin"), templateHandler.DeleteTemplate)

			// Fields — use same :id param to avoid Gin wildcard conflict
			templates.POST("/:id/fields", middleware.RBACMiddleware("admin"), templateHandler.AddField)
			templates.GET("/:id/fields", templateHandler.GetFields)
			templates.GET("/:id/schema", templateHandler.GetTemplateSchema)
		}

		// Fields directly (for PUT/DELETE as per requirements)
		fields := protected.Group("/fields")
		fields.Use(middleware.RBACMiddleware("admin"))
		{
			fields.PUT("/:fieldId", templateHandler.UpdateField)
			fields.DELETE("/:fieldId", templateHandler.DeleteField)
		}

		// Leads
		leads := protected.Group("/leads")
		{
			leads.POST("", leadHandler.CreateLead)
			leads.GET("", leadHandler.GetLeads)
			leads.GET("/:id", leadHandler.GetLead)
			leads.PUT("/:id", leadHandler.UpdateLead)
			leads.DELETE("/:id", leadHandler.DeleteLead)
		}
	}

	return r
}

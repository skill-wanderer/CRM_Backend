package admin

import (
	adminhandlers "crm-backend/internal/admin/handlers"
	"crm-backend/internal/auth"
	"crm-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(api *gin.RouterGroup, tenantHandler *adminhandlers.TenantHandler, verifier auth.TokenVerifier, requiredRole string) {
	tenants := api.Group("/admin/tenants")
	tenants.Use(middleware.Auth(verifier), middleware.RequireRealmRole(requiredRole))
	{
		tenants.POST("", tenantHandler.Create)
		tenants.GET("", tenantHandler.List)
		tenants.GET("/:id", tenantHandler.Get)
		tenants.PUT("/:id", tenantHandler.Update)
		tenants.DELETE("/:id", tenantHandler.Delete)
	}
}

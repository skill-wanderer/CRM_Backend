package client

import (
	"crm-backend/internal/auth"
	"crm-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(api *gin.RouterGroup, verifier auth.TokenVerifier, db *gorm.DB, tenantHeader string) {
	client := api.Group("/client")
	client.Use(
		middleware.Auth(verifier),
		middleware.UserSync(db),
		middleware.TenantScope(db, tenantHeader),
	)
}

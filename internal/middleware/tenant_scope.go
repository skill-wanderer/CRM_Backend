package middleware

import (
	"errors"
	"net/http"
	"strings"

	"crm-backend/internal/models"
	"crm-backend/internal/tenancy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TenantScope(db *gorm.DB, tenantHeader string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawTenantID := strings.TrimSpace(c.GetHeader(tenantHeader))
		if rawTenantID == "" {
			AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "missing tenant header")
			return
		}

		tenantID, err := uuid.Parse(rawTenantID)
		if err != nil {
			AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "malformed tenant header")
			return
		}

		userID, ok := UserIDFromContext(c)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "missing synced user")
			return
		}

		var tenant models.Tenant
		err = db.WithContext(c.Request.Context()).
			Where("id = ?", tenantID).
			First(&tenant).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "tenant is not available")
				return
			}
			AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load tenant")
			return
		}

		if tenant.Status != models.TenantStatusActive {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "tenant is not active")
			return
		}

		var memberships int64
		if err := db.WithContext(c.Request.Context()).
			Model(&models.UserTenant{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Count(&memberships).Error; err != nil {
			AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to check tenant membership")
			return
		}
		if memberships == 0 {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "user is not a member of tenant")
			return
		}

		tenancy.SetTenantID(c, tenantID)
		c.Next()
	}
}

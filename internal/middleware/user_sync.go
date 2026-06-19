package middleware

import (
	"net/http"
	"strings"
	"time"

	"crm-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const UserIDContextKey = "userID"

func UserSync(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "missing authentication claims")
			return
		}

		keycloakSub := strings.TrimSpace(claims.Subject)
		if keycloakSub == "" {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "token is missing subject")
			return
		}

		now := time.Now().UTC()
		user := models.User{
			KeycloakSub: keycloakSub,
			Email:       strings.TrimSpace(claims.Email),
			Name:        claims.DisplayName(),
			LastLoginAt: &now,
		}

		err := db.WithContext(c.Request.Context()).
			Clauses(
				clause.OnConflict{
					Columns: []clause.Column{{Name: "keycloak_sub"}},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"email":         user.Email,
						"name":          user.Name,
						"last_login_at": now,
						"updated_at":    now,
					}),
				},
				clause.Returning{Columns: []clause.Column{{Name: "id"}}},
			).
			Create(&user).Error
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to sync user")
			return
		}

		if user.ID == uuid.Nil {
			if err := db.WithContext(c.Request.Context()).
				Select("id").
				Where("keycloak_sub = ?", keycloakSub).
				First(&user).Error; err != nil {
				AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load synced user")
				return
			}
		}

		c.Set(UserIDContextKey, user.ID)
		c.Next()
	}
}

func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(UserIDContextKey)
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := value.(uuid.UUID)
	return userID, ok
}

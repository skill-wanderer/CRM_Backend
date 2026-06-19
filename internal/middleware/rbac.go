package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRealmRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "missing authentication claims")
			return
		}

		if !claims.HasRealmRole(role) {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", fmt.Sprintf("missing required role: %s", role))
			return
		}

		c.Next()
	}
}

package middleware

import (
	"errors"
	"net/http"
	"strings"

	"crm-backend/internal/auth"
	"github.com/gin-gonic/gin"
)

const ClaimsContextKey = "authClaims"

func Auth(verifier auth.TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken := bearerToken(c.GetHeader("Authorization"))
		if rawToken == "" {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "missing bearer token")
			return
		}

		claims, err := verifier.Verify(c.Request.Context(), rawToken)
		if err != nil {
			if errors.Is(err, auth.ErrBadAudience) {
				AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "token audience is not allowed")
				return
			}
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid bearer token")
			return
		}

		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

func ClaimsFromContext(c *gin.Context) (*auth.Claims, bool) {
	value, exists := c.Get(ClaimsContextKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*auth.Claims)
	return claims, ok
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

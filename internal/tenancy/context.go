package tenancy

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TenantIDGinKey = "tenantID"

type tenantIDContextKey struct{}

func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDContextKey{}, tenantID)
}

func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	value := ctx.Value(tenantIDContextKey{})
	if value == nil {
		return uuid.Nil, false
	}
	tenantID, ok := value.(uuid.UUID)
	return tenantID, ok
}

func SetTenantID(c *gin.Context, tenantID uuid.UUID) {
	c.Set(TenantIDGinKey, tenantID)
	c.Request = c.Request.WithContext(WithTenantID(c.Request.Context(), tenantID))
}

func TenantIDFromGin(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(TenantIDGinKey)
	if !exists {
		return uuid.Nil, false
	}
	tenantID, ok := value.(uuid.UUID)
	return tenantID, ok
}

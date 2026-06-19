package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"crm-backend/internal/models"
	"crm-backend/internal/tenancy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestTenantScopeAllowsActiveMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMiddlewareTestDB(t)

	user := models.User{KeycloakSub: "member-sub", Email: "member@example.com"}
	tenant := models.Tenant{Name: "Acme", Slug: "acme", Status: models.TenantStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if err := db.Create(&models.UserTenant{UserID: user.ID, TenantID: tenant.ID}).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}

	router := gin.New()
	router.GET(
		"/client",
		func(c *gin.Context) {
			c.Set(UserIDContextKey, user.ID)
			c.Next()
		},
		TenantScope(db, "X-Tenant-ID"),
		func(c *gin.Context) {
			if got, ok := tenancy.TenantIDFromContext(c.Request.Context()); !ok || got != tenant.ID {
				t.Fatalf("expected tenant id %s in context, got %s (ok=%v)", tenant.ID, got, ok)
			}
			c.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/client", nil)
	req.Header.Set("X-Tenant-ID", tenant.ID.String())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestTenantScopeRejectsMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMiddlewareTestDB(t)

	router := gin.New()
	router.GET(
		"/client",
		func(c *gin.Context) {
			c.Set(UserIDContextKey, uuid.New())
			c.Next()
		},
		TenantScope(db, "X-Tenant-ID"),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/client", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestTenantScopeRejectsNonMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMiddlewareTestDB(t)

	user := models.User{KeycloakSub: "non-member-sub", Email: "non-member@example.com"}
	tenant := models.Tenant{Name: "Acme", Slug: "acme", Status: models.TenantStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	router := gin.New()
	router.GET(
		"/client",
		func(c *gin.Context) {
			c.Set(UserIDContextKey, user.ID)
			c.Next()
		},
		TenantScope(db, "X-Tenant-ID"),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/client", nil)
	req.Header.Set("X-Tenant-ID", tenant.ID.String())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

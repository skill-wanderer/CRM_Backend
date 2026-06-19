package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crm-backend/internal/auth"
	"crm-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserSyncCreatesAndUpdatesUserFromClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMiddlewareTestDB(t)

	claims := &auth.Claims{
		Subject: "keycloak-sub-1",
		Email:   "first@example.com",
		Name:    "First Name",
	}

	router := gin.New()
	router.GET(
		"/sync",
		func(c *gin.Context) {
			c.Set(ClaimsContextKey, claims)
			c.Next()
		},
		UserSync(db),
		func(c *gin.Context) {
			if _, ok := UserIDFromContext(c); !ok {
				t.Fatal("expected synced user id in context")
			}
			c.Status(http.StatusNoContent)
		},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sync", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected create status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}

	claims.Email = "updated@example.com"
	claims.Name = "Updated Name"

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sync", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected update status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}

	var user models.User
	if err := db.Where("keycloak_sub = ?", "keycloak-sub-1").First(&user).Error; err != nil {
		t.Fatalf("expected synced user: %v", err)
	}
	if user.Email != "updated@example.com" {
		t.Fatalf("expected updated email, got %q", user.Email)
	}
	if user.Name != "Updated Name" {
		t.Fatalf("expected updated name, got %q", user.Name)
	}
	if user.LastLoginAt == nil {
		t.Fatal("expected last login to be set")
	}
}

func newMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	statements := []string{
		`CREATE TABLE tenants (
			id text PRIMARY KEY,
			name text NOT NULL,
			slug text NOT NULL,
			description text,
			status text NOT NULL DEFAULT 'active',
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE UNIQUE INDEX ux_tenants_slug_live ON tenants (slug) WHERE deleted_at IS NULL`,
		`CREATE TABLE users (
			id text PRIMARY KEY,
			keycloak_sub text NOT NULL UNIQUE,
			email text,
			name text,
			last_login_at datetime,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE user_tenants (
			user_id text NOT NULL,
			tenant_id text NOT NULL,
			created_at datetime,
			PRIMARY KEY (user_id, tenant_id)
		)`,
		`CREATE INDEX ix_user_tenants_tenant_id ON user_tenants (tenant_id)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("migrate sqlite db: %v", err)
		}
	}
	return db
}

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"crm-backend/internal/auth"
	"github.com/gin-gonic/gin"
)

type staticVerifier struct {
	claims *auth.Claims
	err    error
}

func (v staticVerifier) Verify(_ context.Context, _ string) (*auth.Claims, error) {
	return v.claims, v.err
}

func TestAuthAndRequireRealmRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/admin",
		Auth(staticVerifier{claims: &auth.Claims{Subject: "operator", RealmAccess: auth.RealmAccess{Roles: []string{"CRM"}}}}),
		RequireRealmRole("CRM"),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestRequireRealmRoleRejectsMissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/admin",
		Auth(staticVerifier{claims: &auth.Claims{Subject: "operator", RealmAccess: auth.RealmAccess{Roles: []string{"OTHER"}}}}),
		RequireRealmRole("CRM"),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

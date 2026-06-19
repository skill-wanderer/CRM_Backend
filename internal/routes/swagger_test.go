package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterSwaggerServesSpecAndUI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	registerSwagger(r)

	specRecorder := httptest.NewRecorder()
	specRequest := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	r.ServeHTTP(specRecorder, specRequest)

	if specRecorder.Code != http.StatusOK {
		t.Fatalf("expected openapi spec status 200, got %d", specRecorder.Code)
	}
	if contentType := specRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/yaml") {
		t.Fatalf("expected yaml content type, got %q", contentType)
	}
	if !strings.Contains(specRecorder.Body.String(), "openapi: 3.0.3") {
		t.Fatal("expected embedded OpenAPI document to be served")
	}

	uiRecorder := httptest.NewRecorder()
	uiRequest := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	r.ServeHTTP(uiRecorder, uiRequest)

	if uiRecorder.Code != http.StatusOK {
		t.Fatalf("expected swagger ui status 200, got %d", uiRecorder.Code)
	}
	if !strings.Contains(uiRecorder.Body.String(), "Swagger UI") {
		t.Fatal("expected Swagger UI page to be served")
	}
}

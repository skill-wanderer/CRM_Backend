package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	adminsvc "crm-backend/internal/admin/services"
	"crm-backend/internal/middleware"
	"crm-backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TenantHandler struct {
	service adminsvc.TenantService
}

func NewTenantHandler(service adminsvc.TenantService) *TenantHandler {
	return &TenantHandler{service: service}
}

type createTenantRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type updateTenantRequest struct {
	Name        *string              `json:"name"`
	Description *string              `json:"description"`
	Status      *models.TenantStatus `json:"status"`
}

func (h *TenantHandler) Create(c *gin.Context) {
	var req createTenantRequest
	if !decodeJSONStrict(c, &req) {
		return
	}

	tenant, err := h.service.Create(c.Request.Context(), adminsvc.CreateTenantInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		handleTenantError(c, err)
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

func (h *TenantHandler) List(c *gin.Context) {
	page, err := intQuery(c, "page", 1)
	if err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "page must be a number")
		return
	}

	pageSize, err := intQuery(c, "pageSize", 20)
	if err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "pageSize must be a number")
		return
	}

	result, err := h.service.List(c.Request.Context(), adminsvc.ListTenantInput{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Query:    c.Query("q"),
	})
	if err != nil {
		handleTenantError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     result.Data,
		"page":     result.Page,
		"pageSize": result.PageSize,
		"total":    result.Total,
	})
}

func (h *TenantHandler) Get(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok {
		return
	}

	tenant, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		handleTenantError(c, err)
		return
	}

	c.JSON(http.StatusOK, tenant)
}

func (h *TenantHandler) Update(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok {
		return
	}

	var req updateTenantRequest
	if !decodeJSONStrict(c, &req) {
		return
	}

	tenant, err := h.service.Update(c.Request.Context(), id, adminsvc.UpdateTenantInput{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		handleTenantError(c, err)
		return
	}

	c.JSON(http.StatusOK, tenant)
}

func (h *TenantHandler) Delete(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		handleTenantError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func decodeJSONStrict(c *gin.Context, dst interface{}) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		middleware.AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "request body must contain a single JSON object")
		return false
	}

	return true
}

func intQuery(c *gin.Context, key string, fallback int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func tenantIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "tenant id must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}

func handleTenantError(c *gin.Context, err error) {
	var validationErr adminsvc.ValidationError
	switch {
	case errors.Is(err, adminsvc.ErrTenantNotFound):
		middleware.AbortWithError(c, http.StatusNotFound, "NOT_FOUND", "tenant not found")
	case errors.Is(err, adminsvc.ErrDuplicateSlug):
		middleware.AbortWithError(c, http.StatusConflict, "CONFLICT", "tenant slug already exists")
	case errors.As(err, &validationErr):
		middleware.AbortWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message)
	default:
		middleware.AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected tenant error")
	}
}

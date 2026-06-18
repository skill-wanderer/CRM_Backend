package handlers

import (
	"net/http"
	"strconv"

	"crm-backend/internal/models"
	"crm-backend/internal/services"
	"github.com/gin-gonic/gin"
)

type TemplateHandler struct {
	service services.TemplateService
}

func NewTemplateHandler(s services.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: s}
}

func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req models.LeadTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assuming Auth middleware sets "userID"
	userID, _ := c.Get("userID")
	if userID != nil {
		req.CreatedBy = uint(userID.(float64))
	}

	if err := h.service.CreateTemplate(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	templates, err := h.service.GetTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	template, err := h.service.GetTemplateByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}
	c.JSON(http.StatusOK, template)
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req models.LeadTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uint(id)
	
	if err := h.service.UpdateTemplate(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.service.DeleteTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func (h *TemplateHandler) AddField(c *gin.Context) {
	templateID, _ := strconv.Atoi(c.Param("id"))
	var req models.LeadField
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddField(uint(templateID), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *TemplateHandler) GetFields(c *gin.Context) {
	templateID, _ := strconv.Atoi(c.Param("id"))
	fields, err := h.service.GetFields(uint(templateID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fields)
}

func (h *TemplateHandler) UpdateField(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("fieldId"))
	var req models.LeadField
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uint(id)

	if err := h.service.UpdateField(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *TemplateHandler) DeleteField(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("fieldId"))
	if err := h.service.DeleteField(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func (h *TemplateHandler) GetTemplateSchema(c *gin.Context) {
	templateID, _ := strconv.Atoi(c.Param("id"))
	
	// Check template exists
	_, err := h.service.GetTemplateByID(uint(templateID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	fields, err := h.service.GetFields(uint(templateID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	schema := gin.H{
		"templateId": templateID,
		"fields":     fields,
	}
	c.JSON(http.StatusOK, schema)
}

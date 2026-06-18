package handlers

import (
	"net/http"
	"strconv"

	"crm-backend/internal/services"
	"github.com/gin-gonic/gin"
)

type LeadHandler struct {
	service services.LeadService
}

func NewLeadHandler(s services.LeadService) *LeadHandler {
	return &LeadHandler{service: s}
}

type CreateLeadRequest struct {
	TemplateID uint              `json:"templateId" binding:"required"`
	Data       map[string]string `json:"data" binding:"required"`
}

func (h *LeadHandler) CreateLead(c *gin.Context) {
	var req CreateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lead, err := h.service.CreateLead(req.TemplateID, req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return 400 for validation errors
		return
	}
	c.JSON(http.StatusCreated, lead)
}

func (h *LeadHandler) GetLeads(c *gin.Context) {
	leads, err := h.service.GetLeads()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, leads)
}

func (h *LeadHandler) GetLead(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lead, err := h.service.GetLeadByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		return
	}
	c.JSON(http.StatusOK, lead)
}

func (h *LeadHandler) UpdateLead(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req CreateLeadRequest // Reuse structure as it takes map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lead, err := h.service.UpdateLead(uint(id), req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, lead)
}

func (h *LeadHandler) DeleteLead(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.service.DeleteLead(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

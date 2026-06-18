package services

import (
	"fmt"

	"crm-backend/internal/models"
	"crm-backend/internal/repositories"
	"crm-backend/internal/utils"
)

type LeadService interface {
	CreateLead(templateID uint, data map[string]string) (*models.Lead, error)
	GetLeads() ([]models.Lead, error)
	GetLeadByID(id uint) (*models.Lead, error)
	UpdateLead(id uint, data map[string]string) (*models.Lead, error)
	DeleteLead(id uint) error
}

type leadService struct {
	leadRepo     repositories.LeadRepository
	templateRepo repositories.TemplateRepository
}

func NewLeadService(lr repositories.LeadRepository, tr repositories.TemplateRepository) LeadService {
	return &leadService{leadRepo: lr, templateRepo: tr}
}

func (s *leadService) CreateLead(templateID uint, data map[string]string) (*models.Lead, error) {
	// 1. Fetch template fields
	fields, err := s.templateRepo.FindFieldsByTemplateID(templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// 2. Map fields by name for easy lookup
	fieldMap := make(map[string]models.LeadField)
	for _, f := range fields {
		fieldMap[f.FieldName] = f
	}

	// 3. Validate input data against fields
	var values []models.LeadValue
	for _, field := range fields {
		valStr := data[field.FieldName]

		if err := utils.ValidateFieldValue(field, valStr); err != nil {
			return nil, err
		}

		// Save value if it's not empty, or if we want to store explicit empty strings
		if valStr != "" {
			values = append(values, models.LeadValue{
				FieldID: field.ID,
				Value:   valStr,
			})
		}
	}

	lead := &models.Lead{
		TemplateID: templateID,
	}

	// 4. Save to DB
	if err := s.leadRepo.CreateLead(lead, values); err != nil {
		return nil, err
	}

	// Re-fetch to include relations
	return s.leadRepo.FindLeadByID(lead.ID)
}

func (s *leadService) GetLeads() ([]models.Lead, error) {
	return s.leadRepo.FindAllLeads()
}

func (s *leadService) GetLeadByID(id uint) (*models.Lead, error) {
	return s.leadRepo.FindLeadByID(id)
}

func (s *leadService) UpdateLead(id uint, data map[string]string) (*models.Lead, error) {
	lead, err := s.leadRepo.FindLeadByID(id)
	if err != nil {
		return nil, err
	}

	fields, err := s.templateRepo.FindFieldsByTemplateID(lead.TemplateID)
	if err != nil {
		return nil, err
	}

	var values []models.LeadValue
	for _, field := range fields {
		valStr := data[field.FieldName]
		if err := utils.ValidateFieldValue(field, valStr); err != nil {
			return nil, err
		}

		if valStr != "" {
			values = append(values, models.LeadValue{
				FieldID: field.ID,
				Value:   valStr,
			})
		}
	}

	if err := s.leadRepo.UpdateLead(lead, values); err != nil {
		return nil, err
	}

	return s.leadRepo.FindLeadByID(lead.ID)
}

func (s *leadService) DeleteLead(id uint) error {
	return s.leadRepo.DeleteLead(id)
}

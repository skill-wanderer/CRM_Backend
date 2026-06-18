package services

import (
	"crm-backend/internal/models"
	"crm-backend/internal/repositories"
)

type TemplateService interface {
	CreateTemplate(template *models.LeadTemplate) error
	GetTemplates() ([]models.LeadTemplate, error)
	GetTemplateByID(id uint) (*models.LeadTemplate, error)
	UpdateTemplate(template *models.LeadTemplate) error
	DeleteTemplate(id uint) error

	AddField(templateID uint, field *models.LeadField) error
	GetFields(templateID uint) ([]models.LeadField, error)
	UpdateField(field *models.LeadField) error
	DeleteField(id uint) error
}

type templateService struct {
	repo repositories.TemplateRepository
}

func NewTemplateService(repo repositories.TemplateRepository) TemplateService {
	return &templateService{repo: repo}
}

func (s *templateService) CreateTemplate(template *models.LeadTemplate) error {
	return s.repo.Create(template)
}

func (s *templateService) GetTemplates() ([]models.LeadTemplate, error) {
	return s.repo.FindAll()
}

func (s *templateService) GetTemplateByID(id uint) (*models.LeadTemplate, error) {
	return s.repo.FindByID(id)
}

func (s *templateService) UpdateTemplate(template *models.LeadTemplate) error {
	return s.repo.Update(template)
}

func (s *templateService) DeleteTemplate(id uint) error {
	return s.repo.Delete(id)
}

func (s *templateService) AddField(templateID uint, field *models.LeadField) error {
	field.TemplateID = templateID
	return s.repo.CreateField(field)
}

func (s *templateService) GetFields(templateID uint) ([]models.LeadField, error) {
	return s.repo.FindFieldsByTemplateID(templateID)
}

func (s *templateService) UpdateField(field *models.LeadField) error {
	return s.repo.UpdateField(field)
}

func (s *templateService) DeleteField(id uint) error {
	return s.repo.DeleteField(id)
}

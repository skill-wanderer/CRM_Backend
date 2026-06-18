package repositories

import (
	"crm-backend/internal/models"
	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(template *models.LeadTemplate) error
	FindAll() ([]models.LeadTemplate, error)
	FindByID(id uint) (*models.LeadTemplate, error)
	Update(template *models.LeadTemplate) error
	Delete(id uint) error

	CreateField(field *models.LeadField) error
	FindFieldsByTemplateID(templateID uint) ([]models.LeadField, error)
	FindFieldByID(id uint) (*models.LeadField, error)
	UpdateField(field *models.LeadField) error
	DeleteField(id uint) error
}

type templateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) Create(template *models.LeadTemplate) error {
	return r.db.Create(template).Error
}

func (r *templateRepository) FindAll() ([]models.LeadTemplate, error) {
	var templates []models.LeadTemplate
	err := r.db.Preload("Fields").Find(&templates).Error
	return templates, err
}

func (r *templateRepository) FindByID(id uint) (*models.LeadTemplate, error) {
	var template models.LeadTemplate
	err := r.db.Preload("Fields").First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *templateRepository) Update(template *models.LeadTemplate) error {
	return r.db.Save(template).Error
}

func (r *templateRepository) Delete(id uint) error {
	return r.db.Delete(&models.LeadTemplate{}, id).Error
}

func (r *templateRepository) CreateField(field *models.LeadField) error {
	return r.db.Create(field).Error
}

func (r *templateRepository) FindFieldsByTemplateID(templateID uint) ([]models.LeadField, error) {
	var fields []models.LeadField
	err := r.db.Where("template_id = ?", templateID).Find(&fields).Error
	return fields, err
}

func (r *templateRepository) FindFieldByID(id uint) (*models.LeadField, error) {
	var field models.LeadField
	err := r.db.First(&field, id).Error
	if err != nil {
		return nil, err
	}
	return &field, nil
}

func (r *templateRepository) UpdateField(field *models.LeadField) error {
	return r.db.Save(field).Error
}

func (r *templateRepository) DeleteField(id uint) error {
	return r.db.Delete(&models.LeadField{}, id).Error
}

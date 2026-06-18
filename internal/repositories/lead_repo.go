package repositories

import (
	"crm-backend/internal/models"
	"gorm.io/gorm"
)

type LeadRepository interface {
	CreateLead(lead *models.Lead, values []models.LeadValue) error
	FindAllLeads() ([]models.Lead, error)
	FindLeadByID(id uint) (*models.Lead, error)
	UpdateLead(lead *models.Lead, values []models.LeadValue) error
	DeleteLead(id uint) error
}

type leadRepository struct {
	db *gorm.DB
}

func NewLeadRepository(db *gorm.DB) LeadRepository {
	return &leadRepository{db: db}
}

func (r *leadRepository) CreateLead(lead *models.Lead, values []models.LeadValue) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(lead).Error; err != nil {
			return err
		}
		for i := range values {
			values[i].LeadID = lead.ID
		}
		if err := tx.Create(&values).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *leadRepository) FindAllLeads() ([]models.Lead, error) {
	var leads []models.Lead
	err := r.db.Preload("Values").Find(&leads).Error
	return leads, err
}

func (r *leadRepository) FindLeadByID(id uint) (*models.Lead, error) {
	var lead models.Lead
	err := r.db.Preload("Values").First(&lead, id).Error
	if err != nil {
		return nil, err
	}
	return &lead, nil
}

func (r *leadRepository) UpdateLead(lead *models.Lead, values []models.LeadValue) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(lead).Error; err != nil {
			return err
		}
		// Replace all values for simplicity: delete old, insert new
		if err := tx.Where("lead_id = ?", lead.ID).Delete(&models.LeadValue{}).Error; err != nil {
			return err
		}
		for i := range values {
			values[i].LeadID = lead.ID
		}
		if err := tx.Create(&values).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *leadRepository) DeleteLead(id uint) error {
	return r.db.Delete(&models.Lead{}, id).Error
}

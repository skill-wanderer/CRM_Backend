package models

import (
	"time"
)

// User represents a system user
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"not null" json:"name"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `gorm:"not null" json:"-"` // Omit password in JSON
	Role     string `gorm:"default:'user'" json:"role"` // 'admin' or 'user'
}

// LeadTemplate defines the structure of a dynamic lead form
type LeadTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CreatedBy uint      `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`

	// Associations
	Fields []LeadField `gorm:"foreignKey:TemplateID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"fields,omitempty"`
}

// LeadField represents a single dynamic field in a LeadTemplate
type LeadField struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	TemplateID  uint   `gorm:"index;not null" json:"templateId"`
	FieldName   string `gorm:"not null" json:"name"`
	FieldType   string `gorm:"not null" json:"type"`   // string, number, boolean, date, select, textarea
	FieldFormat string `gorm:"default:'none'" json:"format"` // email, url, phone, currency, percentage, none
	Required    bool   `gorm:"default:false" json:"required"`
}

// Lead represents an instance of a LeadTemplate
type Lead struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TemplateID uint      `gorm:"index;not null" json:"templateId"`
	CreatedAt  time.Time `json:"createdAt"`

	// Associations
	Values []LeadValue `gorm:"foreignKey:LeadID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"values,omitempty"`
}

// LeadValue stores the actual dynamic value for a given Lead and Field
type LeadValue struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	LeadID  uint   `gorm:"index;not null" json:"leadId"`
	FieldID uint   `gorm:"index;not null" json:"fieldId"`
	Value   string `json:"value"` // Store everything as string, parsed according to FieldType/FieldFormat
}

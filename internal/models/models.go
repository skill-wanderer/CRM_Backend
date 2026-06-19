package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
)

type Tenant struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(120);not null" json:"name"`
	Slug        string         `gorm:"type:varchar(63);not null;uniqueIndex:ux_tenants_slug_live,where:deleted_at IS NULL" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	Status      TenantStatus   `gorm:"type:varchar(20);not null;default:'active';check:status IN ('active','suspended')" json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	KeycloakSub string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	Email       string     `gorm:"type:varchar(255);index" json:"email"`
	Name        string     `gorm:"type:varchar(255)" json:"name"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	Tenants []Tenant `gorm:"many2many:user_tenants;" json:"tenants,omitempty"`
}

type UserTenant struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"userId"`
	TenantID  uuid.UUID `gorm:"type:uuid;primaryKey;index:ix_user_tenants_tenant_id" json:"tenantId"`
	CreatedAt time.Time `json:"createdAt"`

	User   User   `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Tenant Tenant `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

func (t *Tenant) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
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
	FieldType   string `gorm:"not null" json:"type"`         // string, number, boolean, date, select, textarea
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

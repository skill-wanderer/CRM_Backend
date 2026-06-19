package repositories

import (
	"context"
	"strings"

	"crm-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantFilters struct {
	Page     int
	PageSize int
	Status   models.TenantStatus
	Query    string
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	List(ctx context.Context, filters TenantFilters) ([]models.Tenant, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, tenant *models.Tenant) error
}

type tenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *tenantRepository) List(ctx context.Context, filters TenantFilters) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Tenant{})
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if strings.TrimSpace(filters.Query) != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(filters.Query)) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ?", like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.Page - 1) * filters.PageSize
	err := query.
		Order("created_at DESC").
		Limit(filters.PageSize).
		Offset(offset).
		Find(&tenants).Error
	return tenants, total, err
}

func (r *tenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.WithContext(ctx).First(&tenant, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}

func (r *tenantRepository) Delete(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Delete(tenant).Error
}

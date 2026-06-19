package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	adminrepo "crm-backend/internal/admin/repositories"
	"crm-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrDuplicateSlug  = errors.New("tenant slug already exists")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type CreateTenantInput struct {
	Name        string
	Slug        string
	Description string
}

type UpdateTenantInput struct {
	Name        *string
	Description *string
	Status      *models.TenantStatus
}

type ListTenantInput struct {
	Page     int
	PageSize int
	Status   string
	Query    string
}

type TenantList struct {
	Data     []models.Tenant
	Page     int
	PageSize int
	Total    int64
}

type TenantService interface {
	Create(ctx context.Context, input CreateTenantInput) (*models.Tenant, error)
	List(ctx context.Context, input ListTenantInput) (*TenantList, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateTenantInput) (*models.Tenant, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type tenantService struct {
	repo adminrepo.TenantRepository
}

func NewTenantService(repo adminrepo.TenantRepository) TenantService {
	return &tenantService{repo: repo}
}

func (s *tenantService) Create(ctx context.Context, input CreateTenantInput) (*models.Tenant, error) {
	name, err := validateName(input.Name)
	if err != nil {
		return nil, err
	}

	description, err := validateDescription(input.Description)
	if err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(name)
	} else {
		slug = strings.ToLower(slug)
	}
	if err := validateSlug(slug); err != nil {
		return nil, err
	}

	tenant := &models.Tenant{
		Name:        name,
		Slug:        slug,
		Description: description,
		Status:      models.TenantStatusActive,
	}
	if err := s.repo.Create(ctx, tenant); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateSlug
		}
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) List(ctx context.Context, input ListTenantInput) (*TenantList, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var status models.TenantStatus
	if strings.TrimSpace(input.Status) != "" {
		status = models.TenantStatus(strings.TrimSpace(input.Status))
		if err := validateStatus(status); err != nil {
			return nil, err
		}
	}

	tenants, total, err := s.repo.List(ctx, adminrepo.TenantFilters{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Query:    input.Query,
	})
	if err != nil {
		return nil, err
	}

	return &TenantList{
		Data:     tenants,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *tenantService) Get(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) Update(ctx context.Context, id uuid.UUID, input UpdateTenantInput) (*models.Tenant, error) {
	tenant, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name, err := validateName(*input.Name)
		if err != nil {
			return nil, err
		}
		tenant.Name = name
	}

	if input.Description != nil {
		description, err := validateDescription(*input.Description)
		if err != nil {
			return nil, err
		}
		tenant.Description = description
	}

	if input.Status != nil {
		if err := validateStatus(*input.Status); err != nil {
			return nil, err
		}
		tenant.Status = *input.Status
	}

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) Delete(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenant)
}

func validateName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ValidationError{Message: "name is required"}
	}
	if len(value) > 120 {
		return "", ValidationError{Message: "name must be 120 characters or fewer"}
	}
	return value, nil
}

func validateDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > 1000 {
		return "", ValidationError{Message: "description must be 1000 characters or fewer"}
	}
	return value, nil
}

func validateSlug(value string) error {
	if len(value) < 2 || len(value) > 63 {
		return ValidationError{Message: "slug must be 2 to 63 characters"}
	}
	if !slugPattern.MatchString(value) {
		return ValidationError{Message: "slug must contain only lowercase letters, numbers, and hyphens"}
	}
	return nil
}

func validateStatus(status models.TenantStatus) error {
	switch status {
	case models.TenantStatusActive, models.TenantStatusSuspended:
		return nil
	default:
		return ValidationError{Message: fmt.Sprintf("status must be one of: %s, %s", models.TenantStatusActive, models.TenantStatusSuspended)}
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastWasHyphen := false

	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastWasHyphen = false
			continue
		}
		if !lastWasHyphen && builder.Len() > 0 {
			builder.WriteByte('-')
			lastWasHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

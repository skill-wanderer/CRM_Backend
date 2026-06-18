package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"crm-backend/internal/models"
)

// ValidateFieldValue checks a single string value against its field definition
func ValidateFieldValue(field models.LeadField, value string) error {
	if field.Required && value == "" {
		return fmt.Errorf("field '%s' is required", field.FieldName)
	}

	// If empty and not required, skip format validation
	if value == "" {
		return nil
	}

	fieldType := strings.ToLower(field.FieldType)
	fieldFormat := strings.ToLower(field.FieldFormat)

	switch fieldType {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("field '%s' must be a valid number", field.FieldName)
		}
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("field '%s' must be 'true' or 'false'", field.FieldName)
		}
	case "date":
		// Expecting ISO 8601 YYYY-MM-DD or RFC3339
		_, err := time.Parse("2006-01-02", value)
		if err != nil {
			_, err = time.Parse(time.RFC3339, value)
		}
		if err != nil {
			return fmt.Errorf("field '%s' must be a valid date", field.FieldName)
		}
	case "string", "select", "textarea", "url":
		// Strings are broadly valid, we only care about format
	default:
		return fmt.Errorf("unsupported field type '%s' for field '%s'", field.FieldType, field.FieldName)
	}

	// Format validation
	switch fieldFormat {
	case "email":
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,4}$`)
		if !emailRegex.MatchString(value) {
			return fmt.Errorf("field '%s' must be a valid email address", field.FieldName)
		}
	case "url":
		u, err := url.ParseRequestURI(value)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("field '%s' must be a valid URL", field.FieldName)
		}
	case "phone":
		phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
		if !phoneRegex.MatchString(value) {
			return fmt.Errorf("field '%s' must be a valid phone number (E.164 format)", field.FieldName)
		}
	case "currency":
		valClean := value
		if strings.HasPrefix(valClean, "$") || strings.HasPrefix(valClean, "€") || strings.HasPrefix(valClean, "£") {
			valClean = strings.TrimPrefix(valClean, "$")
			valClean = strings.TrimPrefix(valClean, "€")
			valClean = strings.TrimPrefix(valClean, "£")
		}
		currencyRegex := regexp.MustCompile(`^\d+(\.\d{1,2})?$`)
		if !currencyRegex.MatchString(valClean) {
			return fmt.Errorf("field '%s' must be a valid currency amount", field.FieldName)
		}
	case "percentage":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil || val < 0 || val > 100 {
			return fmt.Errorf("field '%s' must be a valid percentage (0-100)", field.FieldName)
		}
	}

	return nil
}

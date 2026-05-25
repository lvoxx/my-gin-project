package handler

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

// formatBindingError converts a go-playground/validator ValidationErrors slice
// into a map[string]string of {field: reason} pairs suitable for returning in
// an apperror.NewValidation details payload.
//
// For non-ValidationErrors (e.g. JSON syntax errors), it returns the raw error
// message under the key "_body".
func formatBindingError(err error) map[string]string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return map[string]string{"_body": err.Error()}
	}

	fields := make(map[string]string, len(ve))
	for _, fe := range ve {
		field := toSnakeCase(fe.Field())
		fields[field] = validationMessage(fe)
	}
	return fields
}

// validationMessage returns a human-readable description for a single field error.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param() + " characters long"
	case "max":
		return "must be at most " + fe.Param() + " characters long"
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "uuid":
		return "must be a valid UUID"
	case "url":
		return "must be a valid URL"
	case "gt":
		return "must be greater than " + fe.Param()
	case "gte":
		return "must be at least " + fe.Param()
	case "lt":
		return "must be less than " + fe.Param()
	case "lte":
		return "must be at most " + fe.Param()
	default:
		return "failed validation: " + fe.Tag()
	}
}

// toSnakeCase converts a PascalCase or camelCase struct field name to
// snake_case so the error keys match the JSON field names in the DTO.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r | 0x20) // toLower for ASCII
	}
	return result.String()
}

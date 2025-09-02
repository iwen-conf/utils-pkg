package errors

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidationError represents a validation error with field-specific information
type ValidationError struct {
	*Error
	Field  string      `json:"field"`
	Value  interface{} `json:"value"`
	Rule   string      `json:"rule"`
	Params interface{} `json:"params,omitempty"`
}

// NewValidationError creates a new validation error
func NewValidationError(field, rule, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Error: New(CodeInvalidInput, message).
			WithContext("field", field).
			WithContext("rule", rule).
			WithContext("value", value),
		Field: field,
		Value: value,
		Rule:  rule,
	}
}

// WithParams adds validation parameters to the error
func (ve *ValidationError) WithParams(params interface{}) *ValidationError {
	ve.Params = params
	ve.Error.WithContext("params", params)
	return ve
}

// Validator provides validation methods
type Validator struct {
	errors []*ValidationError
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make([]*ValidationError, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(err *ValidationError) {
	v.errors = append(v.errors, err)
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors returns all validation errors
func (v *Validator) GetErrors() []*ValidationError {
	return v.errors
}

// GetError returns a combined error if there are validation errors
func (v *Validator) GetError() *Error {
	if !v.HasErrors() {
		return nil
	}

	if len(v.errors) == 1 {
		return v.errors[0].Error
	}

	// Combine multiple validation errors
	mainErr := New(CodeInvalidInput, "Validation failed")
	var messages []string
	var fields []string

	for i, err := range v.errors {
		messages = append(messages, err.Message)
		fields = append(fields, err.Field)
		mainErr.WithContext(fmt.Sprintf("error_%d", i), map[string]interface{}{
			"field":   err.Field,
			"rule":    err.Rule,
			"value":   err.Value,
			"message": err.Message,
		})
	}

	mainErr.WithDetails(strings.Join(messages, "; "))
	mainErr.WithContext("failed_fields", fields)
	mainErr.WithContext("error_count", len(v.errors))

	return mainErr
}

// Clear clears all validation errors
func (v *Validator) Clear() {
	v.errors = make([]*ValidationError, 0)
}

// Validation methods

// Required validates that a field is not empty
func (v *Validator) Required(field string, value interface{}) *Validator {
	if isEmpty(value) {
		v.AddError(NewValidationError(field, "required", fmt.Sprintf("Field '%s' is required", field), value))
	}
	return v
}

// MinLength validates minimum string length
func (v *Validator) MinLength(field string, value string, min int) *Validator {
	if len(value) < min {
		v.AddError(NewValidationError(field, "min_length",
			fmt.Sprintf("Field '%s' must be at least %d characters long", field, min), value).
			WithParams(map[string]interface{}{"min": min}))
	}
	return v
}

// MaxLength validates maximum string length
func (v *Validator) MaxLength(field string, value string, max int) *Validator {
	if len(value) > max {
		v.AddError(NewValidationError(field, "max_length",
			fmt.Sprintf("Field '%s' must be at most %d characters long", field, max), value).
			WithParams(map[string]interface{}{"max": max}))
	}
	return v
}

// Length validates exact string length
func (v *Validator) Length(field string, value string, length int) *Validator {
	if len(value) != length {
		v.AddError(NewValidationError(field, "length",
			fmt.Sprintf("Field '%s' must be exactly %d characters long", field, length), value).
			WithParams(map[string]interface{}{"length": length}))
	}
	return v
}

// Email validates email format
func (v *Validator) Email(field string, value string) *Validator {
	if value != "" {
		if _, err := mail.ParseAddress(value); err != nil {
			v.AddError(NewValidationError(field, "email",
				fmt.Sprintf("Field '%s' must be a valid email address", field), value))
		}
	}
	return v
}

// URL validates URL format
func (v *Validator) URL(field string, value string) *Validator {
	if value != "" {
		if _, err := url.ParseRequestURI(value); err != nil {
			v.AddError(NewValidationError(field, "url",
				fmt.Sprintf("Field '%s' must be a valid URL", field), value))
		}
	}
	return v
}

// Regex validates against a regular expression
func (v *Validator) Regex(field string, value string, pattern string, message ...string) *Validator {
	if value != "" {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil || !matched {
			msg := fmt.Sprintf("Field '%s' format is invalid", field)
			if len(message) > 0 {
				msg = message[0]
			}
			v.AddError(NewValidationError(field, "regex", msg, value).
				WithParams(map[string]interface{}{"pattern": pattern}))
		}
	}
	return v
}

// Numeric validates that a string is numeric
func (v *Validator) Numeric(field string, value string) *Validator {
	if value != "" {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			v.AddError(NewValidationError(field, "numeric",
				fmt.Sprintf("Field '%s' must be numeric", field), value))
		}
	}
	return v
}

// Integer validates that a string is an integer
func (v *Validator) Integer(field string, value string) *Validator {
	if value != "" {
		if _, err := strconv.Atoi(value); err != nil {
			v.AddError(NewValidationError(field, "integer",
				fmt.Sprintf("Field '%s' must be an integer", field), value))
		}
	}
	return v
}

// Min validates minimum numeric value
func (v *Validator) Min(field string, value float64, min float64) *Validator {
	if value < min {
		v.AddError(NewValidationError(field, "min",
			fmt.Sprintf("Field '%s' must be at least %g", field, min), value).
			WithParams(map[string]interface{}{"min": min}))
	}
	return v
}

// Max validates maximum numeric value
func (v *Validator) Max(field string, value float64, max float64) *Validator {
	if value > max {
		v.AddError(NewValidationError(field, "max",
			fmt.Sprintf("Field '%s' must be at most %g", field, max), value).
			WithParams(map[string]interface{}{"max": max}))
	}
	return v
}

// Range validates that a numeric value is within a range
func (v *Validator) Range(field string, value float64, min, max float64) *Validator {
	if value < min || value > max {
		v.AddError(NewValidationError(field, "range",
			fmt.Sprintf("Field '%s' must be between %g and %g", field, min, max), value).
			WithParams(map[string]interface{}{"min": min, "max": max}))
	}
	return v
}

// In validates that a value is in a list of allowed values
func (v *Validator) In(field string, value interface{}, allowed []interface{}) *Validator {
	found := false
	for _, item := range allowed {
		if value == item {
			found = true
			break
		}
	}
	if !found {
		v.AddError(NewValidationError(field, "in",
			fmt.Sprintf("Field '%s' must be one of the allowed values", field), value).
			WithParams(map[string]interface{}{"allowed": allowed}))
	}
	return v
}

// NotIn validates that a value is not in a list of forbidden values
func (v *Validator) NotIn(field string, value interface{}, forbidden []interface{}) *Validator {
	for _, item := range forbidden {
		if value == item {
			v.AddError(NewValidationError(field, "not_in",
				fmt.Sprintf("Field '%s' contains a forbidden value", field), value).
				WithParams(map[string]interface{}{"forbidden": forbidden}))
			break
		}
	}
	return v
}

// Date validates date format
func (v *Validator) Date(field string, value string, layout string) *Validator {
	if value != "" {
		if _, err := time.Parse(layout, value); err != nil {
			v.AddError(NewValidationError(field, "date",
				fmt.Sprintf("Field '%s' must be a valid date in format %s", field, layout), value).
				WithParams(map[string]interface{}{"layout": layout}))
		}
	}
	return v
}

// Before validates that a date is before another date
func (v *Validator) Before(field string, value time.Time, before time.Time) *Validator {
	if !value.Before(before) {
		v.AddError(NewValidationError(field, "before",
			fmt.Sprintf("Field '%s' must be before %s", field, before.Format("2006-01-02")), value).
			WithParams(map[string]interface{}{"before": before}))
	}
	return v
}

// After validates that a date is after another date
func (v *Validator) After(field string, value time.Time, after time.Time) *Validator {
	if !value.After(after) {
		v.AddError(NewValidationError(field, "after",
			fmt.Sprintf("Field '%s' must be after %s", field, after.Format("2006-01-02")), value).
			WithParams(map[string]interface{}{"after": after}))
	}
	return v
}

// Custom allows for custom validation logic
func (v *Validator) Custom(field string, value interface{}, rule string, validationFunc func(interface{}) bool, message string) *Validator {
	if !validationFunc(value) {
		v.AddError(NewValidationError(field, rule, message, value))
	}
	return v
}

// Helper functions

// isEmpty checks if a value is considered empty
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0
	case bool:
		return !v
	default:
		return false
	}
}

// Convenience validation functions

// ValidateRequired validates that a field is required
func ValidateRequired(field string, value interface{}) *ValidationError {
	if isEmpty(value) {
		return NewValidationError(field, "required", fmt.Sprintf("Field '%s' is required", field), value)
	}
	return nil
}

// ValidateEmail validates email format
func ValidateEmail(field string, value string) *ValidationError {
	if value != "" {
		if _, err := mail.ParseAddress(value); err != nil {
			return NewValidationError(field, "email", fmt.Sprintf("Field '%s' must be a valid email address", field), value)
		}
	}
	return nil
}

// ValidateLength validates string length
func ValidateLength(field string, value string, min, max int) *ValidationError {
	length := len(value)
	if length < min {
		return NewValidationError(field, "min_length",
			fmt.Sprintf("Field '%s' must be at least %d characters long", field, min), value).
			WithParams(map[string]interface{}{"min": min, "max": max})
	}
	if length > max {
		return NewValidationError(field, "max_length",
			fmt.Sprintf("Field '%s' must be at most %d characters long", field, max), value).
			WithParams(map[string]interface{}{"min": min, "max": max})
	}
	return nil
}

// ValidateRange validates numeric range
func ValidateRange(field string, value float64, min, max float64) *ValidationError {
	if value < min || value > max {
		return NewValidationError(field, "range",
			fmt.Sprintf("Field '%s' must be between %g and %g", field, min, max), value).
			WithParams(map[string]interface{}{"min": min, "max": max})
	}
	return nil
}

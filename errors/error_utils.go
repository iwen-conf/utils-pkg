package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"reflect"
	"strings"
)

// Is 委托标准库 errors.Is。
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As 委托标准库 errors.As。
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// IsCode 查找链中第一个 *Error 并比较 Code。
func IsCode(err error, code string) bool {
	if code == "" {
		return false
	}
	legacy, ok := asLegacyError(err)
	return ok && legacy.Code == code
}

func asLegacyError(err error) (*Error, bool) {
	var legacy *Error
	if err == nil || !stderrors.As(err, &legacy) {
		return nil, false
	}
	return legacy, true
}

func asRichError(err error) (*RichError, bool) {
	var rich *RichError
	if err == nil || !stderrors.As(err, &rich) {
		return nil, false
	}
	return rich, true
}

// GetCode extracts the error code from any error
func GetCode(err error) string {
	if err == nil {
		return ""
	}
	if customErr, ok := asLegacyError(err); ok {
		return customErr.Code
	}
	return "UNKNOWN_ERROR"
}

// GetSeverity extracts the severity from an error
func GetSeverity(err error) Severity {
	if err == nil {
		return SeverityLow
	}
	if customErr, ok := asLegacyError(err); ok {
		if severity, exists := customErr.Context["severity"]; exists {
			if sev, ok := severity.(Severity); ok {
				return sev
			}
		}
	}
	return SeverityLow
}

// GetCategory extracts the category from an error
func GetCategory(err error) Category {
	if err == nil {
		return CategorySystem
	}
	if customErr, ok := asLegacyError(err); ok {
		if category, exists := customErr.Context["category"]; exists {
			if cat, ok := category.(Category); ok {
				return cat
			}
		}
	}
	return CategorySystem
}

// GetContext extracts context information from an error
func GetContext(err error, key string) (interface{}, bool) {
	if err == nil {
		return nil, false
	}
	if customErr, ok := asLegacyError(err); ok {
		value, exists := customErr.Context[key]
		return value, exists
	}
	return nil, false
}

// GetAllContext returns all context information from an error
func GetAllContext(err error) map[string]interface{} {
	if err == nil {
		return nil
	}
	if customErr, ok := asLegacyError(err); ok {
		return customErr.Context
	}
	return nil
}

// Chain 沿单一 Unwrap 链收集错误。joined error 不猜测分支。
func Chain(err error) []error {
	if err == nil {
		return nil
	}
	var chain []error
	for err != nil {
		chain = append(chain, err)
		unwrapped := stderrors.Unwrap(err)
		if unwrapped == nil {
			break
		}
		err = unwrapped
	}
	return chain
}

// Root 返回单链最后一个 error。遇到无单一 Unwrap 的 joined error 时保留该对象。
func Root(err error) error {
	if err == nil {
		return nil
	}
	for {
		unwrapped := stderrors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// Format formats an error for different output purposes
func Format(err error, format string) string {
	if err == nil {
		return ""
	}

	customErr, ok := err.(*Error)
	if !ok {
		return err.Error()
	}

	switch strings.ToLower(format) {
	case "json":
		data, _ := json.Marshal(customErr)
		return string(data)
	case "short":
		return fmt.Sprintf("[%s] %s", customErr.Code, customErr.Message)
	case "detailed":
		var parts []string
		parts = append(parts, fmt.Sprintf("Code: %s", customErr.Code))
		parts = append(parts, fmt.Sprintf("Message: %s", customErr.Message))
		if customErr.Details != "" {
			parts = append(parts, fmt.Sprintf("Details: %s", customErr.Details))
		}
		parts = append(parts, fmt.Sprintf("Timestamp: %s", customErr.Timestamp.Format("2006-01-02 15:04:05")))
		if len(customErr.Context) > 0 {
			contextStr, _ := json.Marshal(customErr.Context)
			parts = append(parts, fmt.Sprintf("Context: %s", string(contextStr)))
		}
		if customErr.Original != nil {
			parts = append(parts, fmt.Sprintf("Original: %s", customErr.Original.Error()))
		}
		return strings.Join(parts, "\n")
	default:
		return customErr.Error()
	}
}

// Compare compares two errors for equality
func Compare(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 == nil || err2 == nil {
		return false
	}

	customErr1, ok1 := err1.(*Error)
	customErr2, ok2 := err2.(*Error)

	if !ok1 || !ok2 {
		return err1.Error() == err2.Error()
	}

	return customErr1.Code == customErr2.Code &&
		customErr1.Message == customErr2.Message &&
		customErr1.Details == customErr2.Details
}

// Merge combines multiple errors into a single error
func Merge(errors ...error) *Error {
	if len(errors) == 0 {
		return nil
	}

	// Filter out nil errors
	var validErrors []error
	for _, err := range errors {
		if err != nil {
			validErrors = append(validErrors, err)
		}
	}

	if len(validErrors) == 0 {
		return nil
	}

	if len(validErrors) == 1 {
		if customErr, ok := validErrors[0].(*Error); ok {
			return customErr
		}
		return Wrap(validErrors[0], "WRAPPED_ERROR", validErrors[0].Error())
	}

	// Create a merged error
	merged := New("MULTIPLE_ERRORS", "Multiple errors occurred")
	var messages []string
	var codes []string

	for i, err := range validErrors {
		if customErr, ok := err.(*Error); ok {
			codes = append(codes, customErr.Code)
			messages = append(messages, customErr.Message)
			merged.WithContext(fmt.Sprintf("error_%d", i), customErr)
		} else {
			codes = append(codes, "UNKNOWN")
			messages = append(messages, err.Error())
			merged.WithContext(fmt.Sprintf("error_%d", i), err.Error())
		}
	}

	merged.WithDetails(strings.Join(messages, "; "))
	merged.WithContext("error_codes", codes)
	merged.WithContext("error_count", len(validErrors))

	return merged
}

// IsRetryable checks if an error indicates a retryable condition
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	code := GetCode(err)
	retryableCodes := []string{
		CodeTimeout,
		CodeUnavailable,
		CodeNetworkError,
		CodeConnectionError,
		CodeExternalService,
	}

	for _, retryableCode := range retryableCodes {
		if code == retryableCode {
			return true
		}
	}

	return false
}

// IsCritical checks if an error is critical
func IsCritical(err error) bool {
	return GetSeverity(err) == SeverityCritical
}

// Clone creates a deep copy of an error
func Clone(err error) *Error {
	if err == nil {
		return nil
	}

	customErr, ok := err.(*Error)
	if !ok {
		return Wrap(err, "CLONED_ERROR", err.Error())
	}

	cloned := &Error{
		Code:      customErr.Code,
		Message:   customErr.Message,
		Details:   customErr.Details,
		Timestamp: customErr.Timestamp,
		Original:  customErr.Original,
		Context:   make(map[string]interface{}),
	}

	// Deep copy context
	for k, v := range customErr.Context {
		cloned.Context[k] = deepCopyValue(v)
	}

	return cloned
}

func deepCopyValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Map:
		dst := reflect.MakeMapWithSize(val.Type(), val.Len())
		for _, key := range val.MapKeys() {
			dst.SetMapIndex(key, reflect.ValueOf(deepCopyValue(val.MapIndex(key).Interface())))
		}
		return dst.Interface()
	case reflect.Slice:
		if val.IsNil() {
			return reflect.Zero(val.Type()).Interface()
		}
		dst := reflect.MakeSlice(val.Type(), val.Len(), val.Len())
		for i := 0; i < val.Len(); i++ {
			dst.Index(i).Set(reflect.ValueOf(deepCopyValue(val.Index(i).Interface())))
		}
		return dst.Interface()
	case reflect.Array:
		dst := reflect.New(val.Type()).Elem()
		for i := 0; i < val.Len(); i++ {
			dst.Index(i).Set(reflect.ValueOf(deepCopyValue(val.Index(i).Interface())))
		}
		return dst.Interface()
	case reflect.Ptr:
		if val.IsNil() {
			return reflect.Zero(val.Type()).Interface()
		}
		elem := deepCopyValue(val.Elem().Interface())
		ptr := reflect.New(val.Type().Elem())
		ptr.Elem().Set(reflect.ValueOf(elem))
		return ptr.Interface()
	case reflect.Interface:
		if val.IsNil() {
			return reflect.Zero(val.Type()).Interface()
		}
		return deepCopyValue(val.Elem().Interface())
	default:
		return v
	}
}

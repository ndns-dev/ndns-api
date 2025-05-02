package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Validator는 구조체 필드의 유효성을 검사하는 인터페이스입니다
type Validator interface {
	Validate(interface{}) ValidationErrors
}

// ValidationErrors는 모든 유효성 검사 오류를 저장합니다
type ValidationErrors map[string]string

// Add는 ValidationErrors에 새 오류를 추가합니다
func (v ValidationErrors) Add(field, message string) {
	v[field] = message
}

// HasErrors는 ValidationErrors에 오류가 있는지 확인합니다
func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

// Error는 ValidationErrors를 문자열로 반환합니다
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}

	var errors []string
	for field, message := range v {
		errors = append(errors, fmt.Sprintf("%s: %s", field, message))
	}

	return strings.Join(errors, ", ")
}

// StructValidator는 struct 태그를 기반으로 유효성을 검사합니다
type StructValidator struct{}

// NewValidator는 새 StructValidator를 생성합니다
func NewValidator() *StructValidator {
	return &StructValidator{}
}

// Validate는 구조체의 유효성을 검사합니다
// 지원되는 태그:
// - required: 필드가 비어있으면 안됨
// - min=n: 숫자 필드가 n 이상이어야 함, 문자열은 최소 길이
// - max=n: 숫자 필드가 n 이하여야 함, 문자열은 최대 길이
// - email: 이메일 형식이어야 함
// - regexp=pattern: 문자열은 정규식 패턴과 일치해야 함
func (v *StructValidator) Validate(data interface{}) ValidationErrors {
	errors := make(ValidationErrors)

	// 구조체가 아니면 유효성 검사 안함
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		errors.Add("_error", "유효성 검사는 구조체만 가능합니다")
		return errors
	}

	// 구조체의 각 필드에 대해 유효성 검사
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		typeField := typ.Field(i)

		// 내장된 구조체를 재귀적으로 검사
		if field.Kind() == reflect.Struct && typeField.Anonymous {
			nestedErrors := v.Validate(field.Interface())
			for k, v := range nestedErrors {
				errors.Add(k, v)
			}
			continue
		}

		// 유효성 검사 태그가 없으면 건너뜀
		validateTag := typeField.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// JSON 태그에서 필드 이름 가져오기 (없으면 구조체 필드 이름 사용)
		fieldName := typeField.Name
		jsonTag := typeField.Tag.Get("json")
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		// 유효성 검사 규칙 적용
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			if err := validateField(field, fieldName, rule); err != "" {
				errors.Add(fieldName, err)
				break // 하나의 필드에 대해 첫 번째 오류만 보고
			}
		}
	}

	return errors
}

// validateField는 단일 필드에 대한 유효성 검사 규칙을 적용합니다
func validateField(field reflect.Value, fieldName, rule string) string {
	// 규칙을 이름과 파라미터로 분리
	parts := strings.SplitN(rule, "=", 2)
	ruleName := parts[0]

	var param string
	if len(parts) > 1 {
		param = parts[1]
	}

	// 유효성 검사 규칙 적용
	switch ruleName {
	case "required":
		if isEmptyValue(field) {
			return "필수 항목입니다"
		}

	case "min":
		if err := checkMinValue(field, param); err != "" {
			return err
		}

	case "max":
		if err := checkMaxValue(field, param); err != "" {
			return err
		}

	case "email":
		if field.Kind() == reflect.String && field.String() != "" {
			if !isValidEmail(field.String()) {
				return "올바른 이메일 형식이 아닙니다"
			}
		}

	case "regexp":
		if field.Kind() == reflect.String && field.String() != "" {
			if !matchesRegexp(field.String(), param) {
				return fmt.Sprintf("형식이 올바르지 않습니다: %s", param)
			}
		}
	}

	return ""
}

// isEmptyValue는 필드가 비어있는지 확인합니다
func isEmptyValue(field reflect.Value) bool {
	if !field.IsValid() {
		return true
	}

	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return field.Float() == 0
	case reflect.Bool:
		return !field.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return field.IsNil()
	}

	return false
}

// checkMinValue는 최소값 규칙을 검사합니다
func checkMinValue(field reflect.Value, param string) string {
	min := 0
	fmt.Sscanf(param, "%d", &min)

	switch field.Kind() {
	case reflect.String:
		if field.String() != "" && len(field.String()) < min {
			return fmt.Sprintf("최소 %d자 이상이어야 합니다", min)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() < int64(min) {
			return fmt.Sprintf("최소 %d 이상이어야 합니다", min)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() < uint64(min) {
			return fmt.Sprintf("최소 %d 이상이어야 합니다", min)
		}
	case reflect.Float32, reflect.Float64:
		if field.Float() < float64(min) {
			return fmt.Sprintf("최소 %d 이상이어야 합니다", min)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if field.Len() < min {
			return fmt.Sprintf("최소 %d개 이상이어야 합니다", min)
		}
	}

	return ""
}

// checkMaxValue는 최대값 규칙을 검사합니다
func checkMaxValue(field reflect.Value, param string) string {
	max := 0
	fmt.Sscanf(param, "%d", &max)

	switch field.Kind() {
	case reflect.String:
		if field.String() != "" && len(field.String()) > max {
			return fmt.Sprintf("최대 %d자 이하여야 합니다", max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() > int64(max) {
			return fmt.Sprintf("최대 %d 이하여야 합니다", max)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() > uint64(max) {
			return fmt.Sprintf("최대 %d 이하여야 합니다", max)
		}
	case reflect.Float32, reflect.Float64:
		if field.Float() > float64(max) {
			return fmt.Sprintf("최대 %d 이하여야 합니다", max)
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if field.Len() > max {
			return fmt.Sprintf("최대 %d개 이하여야 합니다", max)
		}
	}

	return ""
}

// isValidEmail은 문자열이 이메일 형식인지 확인합니다
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// matchesRegexp는 문자열이 정규식 패턴과 일치하는지 확인합니다
func matchesRegexp(str, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(str)
}

func PaginationRequest(limit int, offset int) (int, int) {
	// 기본값 설정
	if limit <= 0 {
		limit = 10
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

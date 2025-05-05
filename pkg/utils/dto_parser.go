package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ParseAndValidate는 쿼리 파라미터를 DTO로 변환하고 검증합니다.
// queries: 요청 쿼리 맵
// dto: 변환될 DTO 구조체 포인터 (빈 구조체 전달)
// 반환값: 에러가 있으면 fiber.Error, 성공 시 nil 반환
func ParseAndValidate(queries map[string]string, dto interface{}) error {
	validate := NewValidator()
	if queries == nil {
		return fiber.NewError(fiber.StatusBadRequest, "queries가 nil입니다")
	}

	// 타입 검사 - dto는 포인터여야 함
	dtoValue := reflect.ValueOf(dto)
	if dtoValue.Kind() != reflect.Ptr || dtoValue.IsNil() {
		return fiber.NewError(fiber.StatusBadRequest, "DTO는 유효한 포인터 타입이어야 합니다")
	}

	// dto 구조체의 값과 타입 정보 가져오기
	dtoElem := dtoValue.Elem()
	dtoType := dtoElem.Type()

	// 구조체가 아니면 오류 반환
	if dtoElem.Kind() != reflect.Struct {
		return fiber.NewError(fiber.StatusBadRequest, "DTO는 구조체여야 합니다")
	}

	// 구조체의 각 필드를 쿼리 파라미터에서 채우기
	for i := 0; i < dtoElem.NumField(); i++ {
		field := dtoElem.Field(i)
		fieldType := dtoType.Field(i)

		// JSON 태그에서 필드 이름 가져오기
		fieldName := fieldType.Name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}
		// 쿼리 파라미터에서 값 가져오기
		queryValue, exists := queries[fieldName]
		if !exists || queryValue == "" {
			fmt.Printf("필드 %s의 값이 없습니다\n", fieldName)
			continue // 값이 없으면 건너뜀
		}

		// 필드 타입에 따라 값 변환
		switch field.Kind() {
		case reflect.String:
			field.SetString(queryValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(queryValue, 10, 64)
			if err == nil {
				field.SetInt(intVal)
			} else {
				fmt.Printf("필드 %s의 값 변환 실패: %v\n", fieldName, err)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintVal, err := strconv.ParseUint(queryValue, 10, 64)
			if err == nil {
				field.SetUint(uintVal)
			} else {
				fmt.Printf("필드 %s의 값 변환 실패: %v\n", fieldName, err)
			}
		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(queryValue, 64)
			if err == nil {
				field.SetFloat(floatVal)
			} else {
				fmt.Printf("필드 %s의 값 변환 실패: %v\n", fieldName, err)
			}
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(queryValue)
			if err == nil {
				field.SetBool(boolVal)
			} else {
				fmt.Printf("필드 %s의 값 변환 실패: %v\n", fieldName, err)
			}
		default:
			fmt.Printf("필드 %s의 타입 %s은(는) 지원되지 않습니다\n", fieldName, field.Kind())
		}
	}

	// 채워진 DTO 검증
	errors := validate.Validate(dto)
	if errors.HasErrors() {
		fmt.Printf("유효성 검증 실패: %s\n", errors.Error())
		return fiber.NewError(fiber.StatusBadRequest, errors.Error())
	}

	return nil
}

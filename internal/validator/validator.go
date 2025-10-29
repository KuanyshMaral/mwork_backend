package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError — это кастомный тип ошибки, который содержит
// карту ошибок "поле" -> "сообщение".
type ValidationError struct {
	Errors map[string]string
}

// Error реализует стандартный интерфейс error.
func (e *ValidationError) Error() string {
	var errMsgs []string
	for field, msg := range e.Errors {
		errMsgs = append(errMsgs, fmt.Sprintf("field '%s': %s", field, msg))
	}
	return "Validation failed: " + strings.Join(errMsgs, "; ")
}

// Validator — это наша обертка над go-playground/validator.
type Validator struct {
	validate *validator.Validate
}

// New создает новый экземпляр Validator.
func New() *Validator {
	v := validator.New()

	// Регистрируем функцию для использования JSON-тегов в сообщениях об ошибках.
	// Это позволяет нам возвращать клиенту имена полей в camelCase или snake_case,
	// как они определены в DTO, а не имена полей структуры Go.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// === ЗДЕСЬ МЫ БУДЕМ РЕГИСТРИРОВАТЬ КАСТОМНЫЕ ПРАВИЛА ИЗ rules.go ===
	//
	// Пример (когда rules.go будет написан):
	// if err := registerCustomRules(v); err != nil {
	// 	// Здесь лучше использовать логгер и паниковать,
	// 	// так как это ошибка времени запуска приложения.
	// 	log.Fatalf("failed to register custom validation rules: %v", err)
	// }
	//
	// =================================================================

	return &Validator{
		validate: v,
	}
}

// Validate выполняет валидацию переданной структуры.
// Если есть ошибки, возвращает *ValidationError.
func (v *Validator) Validate(i interface{}) error {
	// Выполняем валидацию
	err := v.validate.Struct(i)
	if err == nil {
		return nil // Ошибок нет
	}

	// Проверяем, является ли ошибка ошибкой валидации от go-playground
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// Это какая-то другая ошибка (например, ошибка рефлексии)
		return err
	}

	// Преобразуем ошибки в нашу кастомную карту map[string]string
	customErrors := make(map[string]string)

	for _, fe := range validationErrors {
		// fe.Field() вернет имя из json-тега благодаря RegisterTagNameFunc
		fieldName := fe.Field()

		// Генерируем простое сообщение об ошибке
		// В реальном проекте здесь можно добавить i18n-ключи
		// или более подробные сообщения на основе fe.Tag()
		customErrors[fieldName] = v.getErrorMessage(fe)
	}

	return &ValidationError{Errors: customErrors}
}

// getErrorMessage - вспомогательная функция для генерации сообщений.
// (Можно вынести в отдельный файл или расширить).
func (v *Validator) getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		// Для строк, срезов, карт
		if fe.Kind() == reflect.String || fe.Kind() == reflect.Slice || fe.Kind() == reflect.Map {
			return fmt.Sprintf("Must be at least %s items/characters long", fe.Param())
		}
		// Для чисел
		return fmt.Sprintf("Must be at least %s", fe.Param())
	case "max":
		// Аналогично min
		return fmt.Sprintf("Must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("Must be exactly %s items/characters long", fe.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", strings.Replace(fe.Param(), " ", ", ", -1))
	case "url":
		return "Must be a valid URL"
	default:
		// Для кастомных или необработанных тегов
		return fmt.Sprintf("Invalid value (failed on '%s' tag)", fe.Tag())
	}
}

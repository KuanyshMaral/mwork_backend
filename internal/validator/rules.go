package validator

import (
	"log"
	"mwork_backend/internal/models" // ❗️ Убедитесь, что путь к модулю верный
	"strings"

	"github.com/go-playground/validator/v10"
)

// registerCustomRules регистрирует все кастомные функции валидации в
// переданном экземпляре валидатора.
func registerCustomRules(v *validator.Validate) {

	// -----------------------------------------------------------------
	// ❗️ ОБЯЗАТЕЛЬНО: Обертка для обработки ошибок регистрации
	// -----------------------------------------------------------------
	mustRegister := func(tag string, fn validator.Func) {
		if err := v.RegisterValidation(tag, fn); err != nil {
			// Если правило не удалось зарегистрировать, приложение
			// не должно запускаться, так как это критическая ошибка.
			log.Fatalf("failed to register custom validation tag '%s': %v", tag, err)
		}
	}

	// -----------------------------------------------------------------
	// ➡️ Правила, основанные на 'statuses.go'
	// -----------------------------------------------------------------

	// 'is-user-role': Проверяет, что роль пользователя валидна
	mustRegister("is-user-role", validateUserRole)

	// 'is-casting-status': Проверяет, что статус кастинга валиден
	mustRegister("is-casting-status", validateCastingStatus)

	// 'is-response-status': Проверяет, что статус отклика валиден
	mustRegister("is-response-status", validateResponseStatus)

	// 'is-payment-status': Проверяет, что статус платежа валиден
	mustRegister("is-payment-status", validatePaymentStatus)

	// -----------------------------------------------------------------
	// ➡️ Другие правила на основе моделей
	// -----------------------------------------------------------------

	// 'is-gender': Проверяет пол (из ModelProfile)
	// (Мы можем предположить, какие значения здесь ожидаются)
	mustRegister("is-gender", validateGender)

	// 'is-job-type': Проверяет тип работы (из Casting)
	mustRegister("is-job-type", validateJobType)
}

// --- Функции валидации ---

func validateUserRole(fl validator.FieldLevel) bool {
	// Получаем значение поля как строку
	value := fl.Field().String()
	if value == "" {
		return true // Не проверяем пустые значения, для этого есть 'required'
	}

	// Проверяем, соответствует ли строка одному из наших типов
	switch models.UserRole(value) {
	case models.UserRoleModel, models.UserRoleEmployer, models.UserRoleAdmin:
		return true
	default:
		return false
	}
}

func validateCastingStatus(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // 'required' обрабатывает пустые
	}
	switch models.CastingStatus(value) {
	case models.CastingStatusDraft, models.CastingStatusActive, models.CastingStatusClosed, models.CastingStatusCancelled:
		return true
	default:
		return false
	}
}

func validateResponseStatus(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	switch models.ResponseStatus(value) {
	case models.ResponseStatusPending, models.ResponseStatusAccepted, models.ResponseStatusRejected, models.ResponseStatusWithdrawn:
		return true
	default:
		return false
	}
}

func validatePaymentStatus(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	switch models.PaymentStatus(value) {
	case models.PaymentStatusPending, models.PaymentStatusPaid, models.PaymentStatusFailed, models.PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

func validateGender(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	// Мы предполагаем, что вы используете эти значения.
	// Измените их, если у вас другие.
	switch strings.ToLower(value) {
	case "male", "female", "other", "any": // 'any' для кастингов
		return true
	default:
		return false
	}
}

func validateJobType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	// Из модели Casting, поле JobType
	switch value {
	case "one_time", "permanent":
		return true
	default:
		return false
	}
}

package apperrors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
)

// AppError - основная структура ошибки приложения
type AppError struct {
	Code     ErrorCode   `json:"code"`
	Domain   string      `json:"domain"` // 👈 Новое поле для контекста
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
	Err      error       `json:"-"`
	HTTPCode int         `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s (%v)", e.Domain, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Domain, e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New - базовый конструктор
func New(code ErrorCode, domain, message string, httpCode int) *AppError {
	return &AppError{
		Code:     code,
		Domain:   domain,
		Message:  message,
		HTTPCode: httpCode,
	}
}

// Wrap - оборачивает существующую ошибку в AppError
func Wrap(err error, code ErrorCode, domain, message string, httpCode int) *AppError {
	return &AppError{
		Code:     code,
		Domain:   domain,
		Message:  message,
		Err:      err,
		HTTPCode: httpCode,
	}
}

// Вспомогательные методы
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// MarshalJSON - для кастомного вывода JSON
func (e *AppError) MarshalJSON() ([]byte, error) {
	type alias struct {
		Code    ErrorCode   `json:"code"`
		Domain  string      `json:"domain"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	}
	return json.Marshal(&alias{
		Code:    e.Code,
		Domain:  e.Domain,
		Message: e.Message,
		Details: e.Details,
	})
}

// Is - обертка над стандартной функцией errors.Is
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As - обертка над стандартной функцией errors.As
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}

// --- ОБЩИЕ ХЕЛПЕРЫ (не-доменные) ---

// InternalError оборачивает неизвестную системную ошибку
func InternalError(err error) *AppError {
	return Wrap(err, CodeInternalError, "system", "Internal server error", http.StatusInternalServerError)
}

// ValidationError создает ошибку валидации с деталями
func ValidationError(details interface{}) *AppError {
	return New(CodeValidationFailed, "validation", "Validation failed", http.StatusBadRequest).WithDetails(details)
}

// NewUnauthorizedError создает ошибку авторизации
func NewUnauthorizedError(message string) *AppError {
	return New(CodeUnauthorized, "auth", message, http.StatusUnauthorized)
}

// NewForbiddenError создает ошибку доступа
func NewForbiddenError(message string) *AppError {
	return New(CodeForbidden, "auth", message, http.StatusForbidden)
}

// NewBadRequestError создает ошибку 400
func NewBadRequestError(message string) *AppError {
	return New(CodeValidationFailed, "request", message, http.StatusBadRequest)
}

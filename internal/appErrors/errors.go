package appErrors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
)

// ErrorCode - тип для кодов ошибок
type ErrorCode string

// AppError - основная структура ошибки приложения
type AppError struct {
	Code     ErrorCode   `json:"code"`
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
	Err      error       `json:"-"`
	HTTPCode int         `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Конструктор
func New(code ErrorCode, message string, httpCode int) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		HTTPCode: httpCode,
	}
}

// С цепочкой ошибок
func Wrap(err error, code ErrorCode, message string, httpCode int) *AppError {
	return &AppError{
		Code:     code,
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

// Для маршалинга в JSON
func (e *AppError) MarshalJSON() ([]byte, error) {
	type alias struct {
		Code    ErrorCode   `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	}
	return json.Marshal(&alias{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	})
}

// Is - обертка над стандартной функцией appErrors.Is
// Позволяет использовать appErrors.Is() с нашими ошибками
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As - обертка над стандартной функцией appErrors.As
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}

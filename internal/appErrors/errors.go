package appErrors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
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

// Is - обертка над стандартной функцией errors.Is
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As - обертка над стандартной функцией errors.As
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}

// Предопределенные ошибки
var (
	// Аутентификация
	ErrInvalidCredentials = New(CodeInvalidCredentials, "Invalid email or password", http.StatusUnauthorized)
	ErrUnauthorized       = New(CodeUnauthorized, "Authentication required", http.StatusUnauthorized)
	ErrForbidden          = New(CodeForbidden, "Access denied", http.StatusForbidden)
	ErrInvalidToken       = New(CodeInvalidToken, "Invalid or expired token", http.StatusUnauthorized)

	// Пользователи
	ErrUserNotFound            = New(CodeUserNotFound, "User not found", http.StatusNotFound)
	ErrEmailAlreadyExists      = New(CodeEmailAlreadyExists, "Email already exists", http.StatusConflict)
	ErrUserNotVerified         = New(CodeUserNotVerified, "User not verified", http.StatusForbidden)
	ErrUserSuspended           = New(CodeUserSuspended, "User account suspended", http.StatusForbidden)
	ErrUserBanned              = New(CodeUserBanned, "User account banned", http.StatusForbidden)
	ErrWeakPassword            = New(CodeWeakPassword, "Password must be at least 6 characters", http.StatusBadRequest)
	ErrInvalidUserRole         = New(CodeInvalidUserRole, "Invalid user role", http.StatusBadRequest)
	ErrCannotModifySelf        = New(CodeCannotModifySelf, "Cannot modify your own account", http.StatusBadRequest)
	ErrInsufficientPermissions = New(CodeInsufficientPermissions, "Insufficient permissions", http.StatusForbidden)

	// Профили
	ErrProfileNotFound  = New(CodeProfileNotFound, "Profile not found", http.StatusNotFound)
	ErrProfileNotPublic = New(CodeProfileNotPublic, "Profile is not public", http.StatusForbidden)

	// Валидация
	ErrValidationFailed = New(CodeValidationFailed, "Validation failed", http.StatusBadRequest)

	// Подписки
	ErrSubscriptionCancelled = New("SUBSCRIPTION_CANCELLED", "Subscription is already cancelled", http.StatusBadRequest)

	// Робокасса
	ErrRobokassaError       = New("ROBOCASSA_ERROR", "Robokassa callback verification failed", http.StatusBadRequest)
	ErrInvalidPaymentAmount = New("INVALID_PAYMENT_AMOUNT", "Payment amount does not match", http.StatusBadRequest)

	ErrInvalidSearchCriteria = New("INVALID_SEARCH_CRITERIA", "Invalid search criteria", http.StatusBadRequest)
	ErrSearchTimeout         = New("SEARCH_TIMEOUT", "Search operation timed out", http.StatusBadRequest)

	// Кастинги
	ErrInvalidCastingStatus      = New("INVALID_CASTING_STATUS", "Casting status is invalid", http.StatusBadRequest)
	ErrCastingNotActive          = New("CASTING_NOT_ACTIVE", "Casting is not active", http.StatusBadRequest)
	ErrCastingExpired            = New("CASTING_EXPIRED", "Casting has expired", http.StatusBadRequest)
	ErrCannotRespondToOwnCasting = New("CANNOT_RESPOND_TO_OWN_CASTING", "Cannot respond to your own casting", http.StatusBadRequest)
	ErrResponseAlreadyExists     = New("RESPONSE_ALREADY_EXISTS", "Response already exists for this casting", http.StatusConflict)

	ErrDialogNotFound      = New("DIALOG_NOT_FOUND", "Dialog not found", http.StatusNotFound)
	ErrMessageNotFound     = New("MESSAGE_NOT_FOUND", "Message not found", http.StatusNotFound)
	ErrParticipantNotFound = New("PARTICIPANT_NOT_FOUND", "Participant not found", http.StatusNotFound)
	ErrUserNotInDialog     = New("USER_NOT_IN_DIALOG", "User is not a participant in this dialog", http.StatusForbidden)
	ErrDialogAccessDenied  = New("DIALOG_ACCESS_DENIED", "Access to dialog denied", http.StatusForbidden)
	ErrCastingDialogExists = New("CASTING_DIALOG_EXISTS", "Dialog for this casting already exists", http.StatusConflict)
	ErrInvalidMessageType  = New("INVALID_MESSAGE_TYPE", "Invalid message type", http.StatusBadRequest)
	ErrFileTooLarge        = New("FILE_TOO_LARGE", "File too large", http.StatusBadRequest)
	ErrInvalidFileType     = New("INVALID_FILE_TYPE", "Invalid file type", http.StatusBadRequest)
	ErrCannotDeleteMessage = New("CANNOT_DELETE_MESSAGE", "Cannot delete message", http.StatusBadRequest)

	// Подписки
	ErrSubscriptionLimit = New("SUBSCRIPTION_LIMIT", "Subscription limit reached", http.StatusForbidden)

	// Загрузка файлов и хранилище
	ErrInvalidUploadUsage   = New("INVALID_UPLOAD_USAGE", "Invalid upload usage", http.StatusBadRequest)
	ErrStorageLimitExceeded = New("STORAGE_LIMIT_EXCEEDED", "Storage limit exceeded", http.StatusForbidden)
)

// Функции-помощники для создания ошибок с деталями
func ValidationError(details interface{}) *AppError {
	return ErrValidationFailed.WithDetails(details)
}

func NotFound(resource string) *AppError {
	return New(CodeUserNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

func InternalError(err error) *AppError {
	return Wrap(err, CodeInternalError, "Internal server error", http.StatusInternalServerError)
}

// Функции-помощники для создания стандартных ошибок
func NewConflictError(message string) *AppError {
	return New(CodeEmailAlreadyExists, message, http.StatusConflict)
}

func NewInternalError(message string) *AppError {
	return New(CodeInternalError, message, http.StatusInternalServerError)
}

func NewUnauthorizedError(message string) *AppError {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

func NewForbiddenError(message string) *AppError {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func NewNotFoundError(message string) *AppError {
	return New(CodeUserNotFound, message, http.StatusNotFound)
}

func NewBadRequestError(message string) *AppError {
	return New(CodeValidationFailed, message, http.StatusBadRequest)
}

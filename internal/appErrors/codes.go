package appErrors

import (
	"fmt"
	"net/http"
)

// Коды ошибок сгруппированные по доменам
const (
	// Аутентификация и авторизация
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	CodeForbidden          ErrorCode = "FORBIDDEN"
	CodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	CodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"

	// Валидация
	CodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	CodeInvalidEmail     ErrorCode = "INVALID_EMAIL"
	CodeWeakPassword     ErrorCode = "WEAK_PASSWORD"
	CodeInvalidUserRole  ErrorCode = "INVALID_USER_ROLE"

	// Ресурсы
	CodeUserNotFound      ErrorCode = "USER_NOT_FOUND"
	CodeProfileNotFound   ErrorCode = "PROFILE_NOT_FOUND"
	CodeCastingNotFound   ErrorCode = "CASTING_NOT_FOUND"
	CodePortfolioNotFound ErrorCode = "PORTFOLIO_NOT_FOUND"

	// Бизнес-логика
	CodeEmailAlreadyExists      ErrorCode = "EMAIL_ALREADY_EXISTS"
	CodeUserNotVerified         ErrorCode = "USER_NOT_VERIFIED"
	CodeUserSuspended           ErrorCode = "USER_SUSPENDED"
	CodeUserBanned              ErrorCode = "USER_BANNED"
	CodeProfileNotPublic        ErrorCode = "PROFILE_NOT_PUBLIC"
	CodeCannotModifySelf        ErrorCode = "CANNOT_MODIFY_SELF"
	CodeInsufficientPermissions ErrorCode = "INSUFFICIENT_PERMISSIONS"

	// Системные ошибки
	CodeInternalError        ErrorCode = "INTERNAL_ERROR"
	CodeDatabaseError        ErrorCode = "DATABASE_ERROR"
	CodeExternalServiceError ErrorCode = "EXTERNAL_SERVICE_ERROR"
)

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

	ErrInvalidSearchCriteria = New("invalid search criteria", "invalid search criteria", http.StatusBadRequest)
	ErrSearchTimeout         = New("search operation timed out", "search operation timed out", http.StatusBadRequest)

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

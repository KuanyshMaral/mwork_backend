package apperrors

import (
	"net/http"
)

/*
Этот файл содержит фабрики и предопределенные переменные
для общих ошибок бизнес-логики и домена.
*/

// =========================================================================
// Фабричные ФУНКЦИИ (Используются для оборачивания ошибок, напр. из репозитория)
// =========================================================================

// ErrNotFound - фабрика для ошибки "не найдено" (404)
// Решает 'Unresolved reference 'ErrNotFound”
// Используется, когда ошибка репозитория (типа gorm.ErrRecordNotFound)
// должна быть преобразована в AppError.
func ErrNotFound(err error) *AppError {
	return Wrap(err, CodeNotFound, "resource", "Resource not found", http.StatusNotFound)
}

// ErrAlreadyExists - фабрика для ошибки "уже существует" (409)
// Решает 'Unresolved reference 'ErrAlreadyExists”
func ErrAlreadyExists(err error) *AppError {
	return Wrap(err, CodeAlreadyExists, "resource", "Resource already exists", http.StatusConflict)
}

// ErrConflict - общая фабрика для конфликтов (409)
func ErrConflict(err error, domain, message string) *AppError {
	return Wrap(err, CodeConflict, domain, message, http.StatusConflict)
}

// =========================================================================
// Фабричные ФУНКЦИИ (Для создания новых ошибок)
// =========================================================================

// ErrInvalidOperation - фабрика для невалидных операций (400)
func ErrInvalidOperation(domain, message string) *AppError {
	return New(CodeInvalidOperation, domain, message, http.StatusBadRequest)
}

// ErrInvalidStatus - фабрика для невалидных статусов (400)
func ErrInvalidStatus(domain, message string) *AppError {
	return New(CodeInvalidStatus, domain, message, http.StatusBadRequest)
}

// =========================================================================
// Предопределенные ПЕРЕМЕННЫЕ (Для частых, статичных ошибок)
// =========================================================================

// ErrInvalidUserRole - используется, когда операция не предусмотрена для роли пользователя.
// (Решает 'Unresolved reference 'ErrInvalidUserRole”)
var ErrInvalidUserRole = New(
	CodeInvalidOperation,
	"business_logic",
	"Invalid user role for this operation",
	http.StatusBadRequest, // 400 - это логическая ошибка, а не ошибка прав
)

// ErrCannotModifySelf - используется, когда пользователь (напр. админ) пытается изменить себя.
// (Решает 'Unresolved reference 'ErrCannotModifySelf”)
var ErrCannotModifySelf = New(
	CodeForbidden,
	"business_logic",
	"Operation on self is not allowed",
	http.StatusForbidden, // 403 - это явный запрет
)

// ErrInsufficientPermissions - используется, когда не-админ пытается выполнить админ-действие.
// (Решает 'Unresolved reference 'ErrInsufficientPermissions”)
var ErrInsufficientPermissions = New(
	CodeForbidden,
	"auth",
	"Insufficient permissions",
	http.StatusForbidden, // 403 - классическая ошибка прав
)

// --- Uploads & Files (НОВЫЙ РАЗДЕЛ) ---

// ErrFileTooLarge - файл превышает максимальный размер для одного запроса.
// (Решает 'Unresolved reference 'ErrFileTooLarge”)
var ErrFileTooLarge = New(
	CodeLimitExceeded,
	"validation",
	"File size exceeds the allowed limit",
	http.StatusRequestEntityTooLarge, // 413
)

// ErrInvalidFileType - MIME-тип файла не разрешен.
// (Решает 'Unresolved reference 'ErrInvalidFileType”)
var ErrInvalidFileType = New(
	CodeValidationFailed,
	"validation",
	"The provided file type is not allowed",
	http.StatusUnsupportedMediaType, // 415
)

// ErrInvalidUploadUsage - параметр 'usage' невалиден для 'entityType'.
// (Решает 'Unresolved reference 'ErrInvalidUploadUsage”)
var ErrInvalidUploadUsage = New(
	CodeValidationFailed,
	"validation",
	"Invalid 'usage' parameter for this entity type",
	http.StatusBadRequest, // 400
)

// ErrStorageLimitExceeded - загрузка превысит квоту хранилища пользователя.
// (Решает 'Unresolved reference 'ErrStorageLimitExceeded”)
var ErrStorageLimitExceeded = New(
	CodeLimitExceeded,
	"storage", // Это правило квоты, а не просто валидации
	"User storage quota exceeded",
	http.StatusForbidden, // 403 (Запрещено использовать больше места)
)

// --- Subscriptions & Payments (НОВЫЙ РАЗДЕЛ) ---

// ErrSubscriptionCancelled - подписка уже отменена.
// (Решает 'Unresolved reference 'ErrSubscriptionCancelled”)
var ErrSubscriptionCancelled = New(
	CodeInvalidOperation,
	"subscription",
	"Subscription is already cancelled",
	http.StatusBadRequest, // 400
)

// ErrInvalidPaymentAmount - сумма платежа не совпадает.
// (Решает 'Unresolved reference 'ErrInvalidPaymentAmount”)
var ErrInvalidPaymentAmount = New(
	CodeConflict,
	"payment",
	"Invalid payment amount",
	http.StatusConflict, // 409
)

// ErrRobokassaError - общая ошибка интеграции с Robokassa (напр. неверная подпись).
// (Решает 'Unresolved reference 'ErrRobokassaError”)
var ErrRobokassaError = New(
	CodeExternalServiceError,
	"payment",
	"Payment provider error",
	http.StatusServiceUnavailable, // 503
)

// --- Chat ---

// ErrDialogNotFound - диалог не найден.
// (Решает 'Unresolved reference 'ErrDialogNotFound”)
// *Примечание: в handleChatError у тебя также есть repositories.ErrDialogNotFound, что является
// более предпочтительным паттерном. Прямое использование этой ошибки в сервисе - исключение.
var ErrDialogNotFound = New(
	CodeNotFound,
	"chat",
	"Dialog not found",
	http.StatusNotFound, // 404
)

// ErrDialogAccessDenied - пользователь не является участником диалога.
// (Решает 'Unresolved reference 'ErrDialogAccessDenied”)
var ErrDialogAccessDenied = New(
	CodeForbidden,
	"chat",
	"Access to dialog denied",
	http.StatusForbidden, // 403
)

// ErrParticipantNotFound - участник не найден в диалоге.
// (Решает 'Unresolved reference 'ErrParticipantNotFound”)
var ErrParticipantNotFound = New(
	CodeNotFound,
	"chat",
	"Participant not found in this dialog",
	http.StatusNotFound, // 404
)

// ErrInvalidMessageType - неверный тип сообщения.
// (Решает 'Unresolved reference 'ErrInvalidMessageType”)
var ErrInvalidMessageType = New(
	CodeValidationFailed,
	"validation",
	"Invalid message type",
	http.StatusBadRequest, // 400
)

// ErrCannotDeleteMessage - нет прав на удаление этого сообщения.
// (Решает 'Unresolved reference 'ErrCannotDeleteMessage”)
var ErrCannotDeleteMessage = New(
	CodeForbidden,
	"chat",
	"You do not have permission to delete this message",
	http.StatusForbidden, // 403
)

// --- Casting (НОВЫЙ РАЗДЕЛ) ---

// ErrSubscriptionLimit - превышен лимит подписки (напр. кол-во публикаций).
// (Решает 'Unresolved reference 'ErrSubscriptionLimit”)
var ErrSubscriptionLimit = New(
	CodeLimitExceeded,
	"subscription",
	"Subscription limit for this feature has been reached",
	http.StatusForbidden, // 403
)

// ErrInvalidCastingStatus - операция невозможна в текущем статусе кастинга.
// (Решает 'Unresolved reference 'ErrInvalidCastingStatus”)
var ErrInvalidCastingStatus = New(
	CodeInvalidStatus,
	"casting",
	"Operation not allowed for the current casting status",
	http.StatusConflict, // 409
)

// --- Auth & User Status (НОВЫЙ РАЗДЕЛ) ---

// ErrWeakPassword - пароль слишком слабый.
// (Решает 'Unresolved reference 'ErrWeakPassword”)
var ErrWeakPassword = New(
	CodeValidationFailed,
	"validation",
	"Password is too weak. Minimum 6 characters required.",
	http.StatusBadRequest, // 400
)

// ErrEmailAlreadyExists - email уже используется.
// (Решает 'Unresolved reference 'ErrEmailAlreadyExists”)
var ErrEmailAlreadyExists = New(
	CodeAlreadyExists,
	"auth",
	"Email already in use",
	http.StatusConflict, // 409
)

// ErrInvalidCredentials - неверный email или пароль.
// (Решает 'Unresolved reference 'ErrInvalidCredentials”)
var ErrInvalidCredentials = New(
	CodeInvalidCredentials,
	"auth",
	"Invalid email or password",
	http.StatusUnauthorized, // 401
)

// ErrInvalidToken - неверный или просроченный токен (refresh, verify, reset).
// (Решает 'Unresolved reference 'ErrInvalidToken”)
var ErrInvalidToken = New(
	CodeInvalidToken,
	"auth",
	"Invalid or expired token",
	http.StatusUnauthorized, // 401
)

// ErrUserSuspended - аккаунт временно заблокирован.
// (Решает 'Unresolved reference 'ErrUserSuspended”)
var ErrUserSuspended = New(
	CodeForbidden,
	"auth",
	"Your account has been suspended",
	http.StatusForbidden, // 403
)

// ErrUserBanned - аккаунт забанен.
// (Решает 'Unresolved reference 'ErrUserBanned”)
var ErrUserBanned = New(
	CodeForbidden,
	"auth",
	"Your account has been banned",
	http.StatusForbidden, // 403
)

// ErrUserNotVerified - email не подтвержден.
// (Решает 'Unresolved reference 'ErrUserNotVerified”)
var ErrUserNotVerified = New(
	CodeForbidden,
	"auth",
	"Please verify your email address",
	http.StatusForbidden, // 403
)

// --- Profile (НОВЫЙ РАЗДЕЛ) ---

// ErrProfileNotPublic - профиль скрыт и недоступен.
// (Решает 'Unresolved reference 'ErrProfileNotPublic”)
var ErrProfileNotPublic = New(
	CodeForbidden,
	"profile",
	"This profile is private",
	http.StatusForbidden, // 403
)

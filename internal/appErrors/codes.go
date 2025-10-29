package appErrors

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

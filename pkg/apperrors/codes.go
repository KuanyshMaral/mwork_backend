package apperrors

// ErrorCode - тип для кодов ошибок
type ErrorCode string

// Общие, не-доменные коды ошибок
const (
	// Системные и неизвестные ошибки
	CodeInternalError        ErrorCode = "INTERNAL_ERROR"
	CodeDatabaseError        ErrorCode = "DATABASE_ERROR"
	CodeExternalServiceError ErrorCode = "EXTERNAL_SERVICE_ERROR"
	CodeUnknownError         ErrorCode = "UNKNOWN_ERROR"

	// Общие ошибки бизнес-логики (используются фабриками)
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	CodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeLimitExceeded    ErrorCode = "LIMIT_EXCEEDED"
	CodeInvalidStatus    ErrorCode = "INVALID_STATUS"
	CodeInvalidOperation ErrorCode = "INVALID_OPERATION"

	// Аутентификация и Авторизация (они сквозные)
	CodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	CodeForbidden          ErrorCode = "FORBIDDEN"
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	CodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
)

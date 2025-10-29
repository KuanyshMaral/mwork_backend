package logger

import (
	"context"
	"log/slog"
)

// Ключи для context
type contextKey string

const (
	requestIDKey     contextKey = "request_id"
	userIDKey        contextKey = "user_id"
	correlationIDKey contextKey = "correlation_id"
)

// ============================================
// Context operations
// ============================================

// WithRequestID добавляет request ID в context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithUserID добавляет user ID в context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithCorrelationID добавляет correlation ID в context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// GetRequestID извлекает request ID из context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID извлекает user ID из context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// ============================================
// Context-aware логирование
// ============================================

// FromContext создает логгер с полями из context
// Автоматически добавляет request_id, user_id, correlation_id если есть в контексте
func FromContext(ctx context.Context) *slog.Logger {
	logger := GetLogger()

	// Добавляем все доступные поля из контекста
	var fields []any

	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, "request_id", requestID)
	}

	if userID := GetUserID(ctx); userID != "" {
		fields = append(fields, "user_id", userID)
	}

	if correlationID, ok := ctx.Value(correlationIDKey).(string); ok && correlationID != "" {
		fields = append(fields, "correlation_id", correlationID)
	}

	if len(fields) > 0 {
		logger = logger.With(fields...)
	}

	return logger
}

// ============================================
// Convenience функции с context
// ============================================

// CtxDebug логирует debug с контекстом
func CtxDebug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// CtxInfo логирует info с контекстом
func CtxInfo(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// CtxWarn логирует warning с контекстом
func CtxWarn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// CtxError логирует error с контекстом
func CtxError(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}

// CtxWithError логирует error с error объектом
func CtxWithError(ctx context.Context, msg string, err error, args ...any) {
	fields := append([]any{"error", err.Error()}, args...)
	FromContext(ctx).Error(msg, fields...)
}

// ============================================
// Примеры использования
// ============================================

/*

// 1. Базовое логирование (без context)
logger.Info("server started", "port", 8080)
logger.Error("failed to connect", "error", err)
logger.Debug("processing request", "user_id", 123)

// 2. Логирование с дополнительными полями
logger.With("user_id", 123, "action", "login").Info("user action")

// 3. Логирование с context (в HTTP handlers)
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Логгер автоматически включит request_id и user_id из контекста
    logger.CtxInfo(ctx, "fetching user", "user_id", userID)

    // Или создать логгер один раз
    log := logger.FromContext(ctx)
    log.Info("step 1")
    log.Info("step 2")
}

// 4. Специализированное логирование
logger.HTTPLog("GET", "/api/users", 200, 50*time.Millisecond, 1024)
logger.DBLog("SELECT", "SELECT * FROM users WHERE id = $1", 10*time.Millisecond, nil)
logger.WorkerLog("email_worker", "send_email", nil)

// 5. Добавление context в middleware
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        ctx := logger.WithRequestID(r.Context(), requestID)

        // Теперь все логи в этом запросе будут иметь request_id
        logger.CtxInfo(ctx, "request started")

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 6. Добавление user_id после аутентификации
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := getUserIDFromToken(r)
        ctx := logger.WithUserID(r.Context(), userID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 7. Логирование в сервисах
func (s *UserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) error {
    log := logger.FromContext(ctx)

    log.Info("creating user", "email", req.Email)

    user, err := s.repo.Create(ctx, req)
    if err != nil {
        log.Error("failed to create user", "error", err)
        return err
    }

    log.Info("user created successfully", "user_id", user.ID)
    return nil
}

// 8. Fatal ошибки (прерывают выполнение)
if err := db.Ping(); err != nil {
    logger.Fatal("failed to connect to database", "error", err)
    // Программа завершится с exit code 1
}
*/

package logger

import (
	"log/slog"
	"os"
	"time"
)

var log *slog.Logger

// Init инициализирует глобальный логгер
// env: "development" или "production"
func Init(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true, // Добавляет файл и строку где вызван лог
	}

	if env == "development" {
		// Development: читаемый текстовый формат с цветами
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// Production: JSON формат для парсинга
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	log = slog.New(handler)
	slog.SetDefault(log) // Устанавливаем как default для всего приложения
}

// GetLogger возвращает глобальный логгер
func GetLogger() *slog.Logger {
	if log == nil {
		// Fallback если Init не вызван
		Init("development")
	}
	return log
}

// ============================================
// Convenience функции для быстрого логирования
// ============================================

// Debug логирует debug сообщение
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// Info логирует info сообщение
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Warn логирует warning сообщение
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Error логирует error сообщение
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// Fatal логирует fatal ошибку и завершает программу
func Fatal(msg string, args ...any) {
	GetLogger().Error(msg, args...)
	os.Exit(1)
}

// ============================================
// Логирование с дополнительными полями
// ============================================

// With создает новый логгер с дополнительными полями
// Пример: logger.With("user_id", 123, "action", "login").Info("user logged in")
func With(args ...any) *slog.Logger {
	return GetLogger().With(args...)
}

// WithError создает логгер с полем error
func WithError(err error) *slog.Logger {
	return GetLogger().With("error", err.Error())
}

// ============================================
// Специализированные логгеры
// ============================================

// HTTPLog логирует HTTP запрос
func HTTPLog(method, path string, status int, duration time.Duration, size int) {
	GetLogger().Info("http request",
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"size_bytes", size,
	)
}

// DBLog логирует database операцию
func DBLog(operation, query string, duration time.Duration, err error) {
	fields := []any{
		"operation", operation,
		"query", query,
		"duration_ms", duration.Milliseconds(),
	}

	if err != nil {
		fields = append(fields, "error", err.Error())
		GetLogger().Error("database operation failed", fields...)
	} else {
		GetLogger().Debug("database operation", fields...)
	}
}

// WorkerLog логирует background worker операцию
func WorkerLog(worker, operation string, err error) {
	fields := []any{
		"worker", worker,
		"operation", operation,
	}

	if err != nil {
		fields = append(fields, "error", err.Error())
		GetLogger().Error("worker operation failed", fields...)
	} else {
		GetLogger().Info("worker operation completed", fields...)
	}
}

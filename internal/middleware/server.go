package middleware

import (
	"log/slog"
	"mwork_backend/internal/logger"
	"mwork_backend/pkg/contextkeys"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		log := logger.FromContext(c.Request.Context())
		fields := []any{
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.Int("status", c.Writer.Status()),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Duration("duration", duration),
			slog.Int("size_bytes", c.Writer.Size()),
		}
		if c.Writer.Status() >= 500 {
			log.Error("HTTP Server Error", fields...)
		} else if c.Writer.Status() >= 400 {
			log.Warn("HTTP Client Error", fields...)
		} else {
			log.Info("HTTP Request", fields...)
		}
	}
}

func DBMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbKey := string(contextkeys.DBContextKey)
		tx, ok := c.Request.Context().Value(contextkeys.DBContextKey).(*gorm.DB)

		if ok && tx != nil {
			c.Set(dbKey, tx)
		} else {
			c.Set(dbKey, db)
		}

		c.Next()
	}
}

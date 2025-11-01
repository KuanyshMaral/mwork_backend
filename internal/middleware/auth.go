package middleware

import (
	"mwork_backend/internal/auth"
	"mwork_backend/internal/logger" // <-- 2. ДОБАВЛЕН ИМПОРТ
	"mwork_backend/internal/models"
	"mwork_backend/pkg/apperrors" // <-- 1. ДОБАВЛЕН ИМПОРТ
	// "net/http" // <-- Больше не нужен
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware - middleware проверки JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// 3. Стандартизируем ошибку
			apperrors.HandleError(c, apperrors.NewUnauthorizedError("Authorization header missing or invalid"))
			c.Abort() // Abort, т.к. HandleError не прерывает
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseToken(tokenStr)
		if err != nil {
			// 3. Стандартизируем ошибку
			apperrors.HandleError(c, apperrors.NewUnauthorizedError("Invalid token"))
			c.Abort()
			return
		}

		// --- 4. 📍 ВОТ ГЛАВНОЕ ИЗМЕНЕНИЕ ---

		// а) Поместить ID в Gin-контекст (для h.GetAndAuthorizeUserID)
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		// б) Поместить ID в Context (для logger.Ctx...)
		ctx := logger.WithUserID(c.Request.Context(), claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		// --- Конец ---

		c.Next()
	}
}

// RoleMiddleware - middleware ограничения по ролям
func RoleMiddleware(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			// 3. Стандартизируем ошибку
			apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied: no role"))
			c.Abort()
			return
		}

		role, ok := roleVal.(models.UserRole)
		if !ok {
			// Попытка преобразовать из string, если роль сохранена как строка
			roleStr, isString := roleVal.(string)
			if !isString {
				apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied: invalid role type"))
				c.Abort()
				return
			}
			role = models.UserRole(roleStr)
		}

		if role != requiredRole {
			apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied: insufficient permissions"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRoles - middleware для проверки нескольких возможных ролей (альтернативный вариант)
func RequireRoles(roles ...models.UserRole) gin.HandlerFunc {
	roleSet := make(map[models.UserRole]bool)
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied: no role"))
			c.Abort()
			return
		}

		role, ok := roleVal.(models.UserRole)
		if !ok {
			roleStr, isString := roleVal.(string)
			if !isString {
				apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied: invalid role type"))
				c.Abort()
				return
			}
			role = models.UserRole(roleStr)
		}

		if !roleSet[role] {
			apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied: insufficient role"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// 5. --- ФУНКЦИЯ GetUserID() УДАЛЕНА ---
//
// ❗️ Она больше не нужна.
// Все хэндлеры теперь должны использовать h.GetAndAuthorizeUserID(c)
// из BaseHandler, который автоматически проверяет наличие userID и
// отправляет 401 ошибку, если его нет.
//

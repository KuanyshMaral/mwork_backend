package middleware

import (
	"mwork_backend/internal/auth"
	"mwork_backend/internal/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware - middleware проверки JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or invalid"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Сохраняем claims в контекст
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RoleMiddleware - middleware ограничения по ролям
func RoleMiddleware(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: no role"})
			return
		}

		role, ok := roleVal.(models.UserRole)
		if !ok {
			// Попытка преобразовать из string, если роль сохранена как строка
			roleStr, isString := roleVal.(string)
			if !isString {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: invalid role type"})
				return
			}
			role = models.UserRole(roleStr)
		}

		if role != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: insufficient permissions"})
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
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: no role"})
			return
		}

		role, ok := roleVal.(models.UserRole)
		if !ok {
			roleStr, isString := roleVal.(string)
			if !isString {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: invalid role type"})
				return
			}
			role = models.UserRole(roleStr)
		}

		if !roleSet[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: insufficient role"})
			return
		}

		c.Next()
	}
}

// GetUserID извлекает ID пользователя из контекста
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}

	id, ok := userID.(string)
	if !ok {
		return ""
	}

	return id
}

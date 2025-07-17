package middleware

import (
	"mwork_front_fn/backend/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Middleware проверки JWT
func JWTAuthMiddleware() gin.HandlerFunc {
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

// Middleware ограничения по ролям
func RequireRoles(roles ...string) gin.HandlerFunc {
	roleSet := make(map[string]bool)
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: no role"})
			return
		}

		role := roleVal.(string)
		if !roleSet[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: insufficient role"})
			return
		}

		c.Next()
	}
}

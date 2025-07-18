package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/handlers"
)

func SetupPublicRoutes(r *gin.Engine, authHandler *handlers.AuthHandler) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)    // регистрация
		auth.POST("/login", authHandler.Login)          // логин
		auth.POST("/refresh", authHandler.RefreshToken) // получить новый access_token
		auth.POST("/logout", authHandler.Logout)        // разлогиниться (удалить refresh)
	}
}

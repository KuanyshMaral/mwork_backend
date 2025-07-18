package routes

import (
	"mwork_backend/internal/middlewares"
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
)

func SetupWebSocketRoutes(r *gin.Engine, wsHandler *ws.WebSocketHandler) {
	// 💬 WebSocket endpoint (только авторизованные пользователи)
	wsGroup := r.Group("/ws")
	wsGroup.Use(middleware.JWTAuthMiddleware()) // поправь путь, если у тебя middleware, а не middlewares
	{
		// Прокси обёртка, чтобы адаптировать http.HandlerFunc под gin.HandlerFunc
		wsGroup.GET("/connect", func(c *gin.Context) {
			wsHandler.HandleWebSocketConnection(c.Writer, c.Request)
		})
	}
}

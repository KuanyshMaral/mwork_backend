package routes

import (
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/logger"
	"mwork_backend/internal/middleware"
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes регистрирует все HTTP и WebSocket маршруты.
func RegisterRoutes(
	ginRouter *gin.Engine,
	appHandlers *handlers.AppHandlers, // <-- Принимаем ГОТОВЫЕ хэндлеры
	wsHandler *ws.WebSocketHandler,
) {
	// Регистрация HTTP API v1
	api := ginRouter.Group("/api/v1")
	{
		appHandlers.AuthHandler.RegisterRoutes(api)
		appHandlers.FileHandler.RegisterRoutes(api)
		appHandlers.UserHandler.RegisterRoutes(api)
		appHandlers.ProfileHandler.RegisterRoutes(api)
		appHandlers.CastingHandler.RegisterRoutes(api)
		appHandlers.ResponseHandler.RegisterRoutes(api)
		appHandlers.ReviewHandler.RegisterRoutes(api)
		appHandlers.PortfolioHandler.RegisterRoutes(api)
		appHandlers.MatchingHandler.RegisterRoutes(api)
		appHandlers.NotificationHandler.RegisterRoutes(api)
		appHandlers.SubscriptionHandler.RegisterRoutes(api)
		appHandlers.SearchHandler.RegisterRoutes(api)
		appHandlers.AnalyticsHandler.RegisterRoutes(api)
		appHandlers.ChatHandler.RegisterRoutes(api)
		appHandlers.UploadHandler.RegisterRoutes(api)
	}

	// Регистрация WebSocket
	wsGroup := ginRouter.Group("/ws")
	wsGroup.Use(middleware.AuthMiddleware()) // <-- AuthMiddleware должно быть в пакете middleware
	{
		wsGroup.GET("", wsHandler.ServeWS)
	}
	logger.Info("WebSocket route /ws registered")
}

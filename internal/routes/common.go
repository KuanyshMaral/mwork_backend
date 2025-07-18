package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/middlewares"
)

func SetupCommonRoutes(
	r *gin.Engine,
	userHandler *handlers.UserHandler,
	chatHandler *handlers.ChatHandler,
	uploadHandler *handlers.UploadHandler,
	subscriptionHandler *handlers.SubscriptionHandler,
) {
	// 👤 Users (любая авторизация)
	users := r.Group("/users")
	users.Use(middleware.JWTAuthMiddleware())
	{
		users.GET("/:id", userHandler.GetUser)
		users.PUT("/:id", userHandler.UpdateUser)
		users.DELETE("/:id", userHandler.DeleteUser)
	}

	// 💬 Chat (авторизованные пользователи)
	chat := r.Group("/chat")
	chat.Use(middleware.JWTAuthMiddleware())
	{
		chat.POST("/dialogs", chatHandler.CreateDialog)
		chat.POST("/messages/send", chatHandler.SendMessage)
		chat.GET("/dialogs/:id/messages", chatHandler.GetMessages)
		chat.POST("/dialogs/:id/read", chatHandler.MarkAllAsRead)
		chat.GET("/dialogs/:id/files", chatHandler.GetDialogFiles)
		chat.POST("/reactions/toggle", chatHandler.ToggleReaction)
		chat.POST("/dialogs/:id/leave", chatHandler.LeaveDialog)
		chat.GET("/dialogs/:id/unread", chatHandler.GetUnreadCount)
	}

	// 📤 Uploads (для всех авторизованных)
	upload := r.Group("/uploads")
	upload.Use(middleware.JWTAuthMiddleware())
	{
		upload.POST("/", uploadHandler.Upload)
	}

	subscription := r.Group("/subscription")
	subscription.Use(middleware.JWTAuthMiddleware())
	{
		subscription.GET("/plans", subscriptionHandler.GetAllPlans)
		subscription.POST("/user", subscriptionHandler.CreateSubscription)
		subscription.POST("/user/cancel", subscriptionHandler.CancelMySubscription)
		subscription.GET("/user", subscriptionHandler.GetUserSubscriptions)
		subscription.POST("/initiate-payment", subscriptionHandler.InitiatePayment)
		subscription.GET("/plans/revenue", subscriptionHandler.GetRevenueByPeriod)
		subscription.GET("/plans/stats", subscriptionHandler.GetPlansWithStats)
	}
}

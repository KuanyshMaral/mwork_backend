package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_front_fn/backend/handlers"
	"mwork_front_fn/backend/middlewares"
)

func SetupRoutes(
	r *gin.Engine,
	userHandler *handlers.UserHandler,
	authHandler *handlers.AuthHandler,
	modelProfileHandler *handlers.ModelProfileHandler,
	employerProfileHandler *handlers.EmployerProfileHandler,
	castingHandler *handlers.CastingHandler,
	responseHandler *handlers.ResponseHandler,
	chatHandler *handlers.ChatHandler,
	subscriptionHandler *handlers.SubscriptionHandler,
	uploadHandler *handlers.UploadHandler,
	analyticsHandler *handlers.AnalyticsHandler,
) *gin.Engine {

	// 🔐 Auth (без ограничений)
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)    // регистрация
		auth.POST("/login", authHandler.Login)          // логин
		auth.POST("/refresh", authHandler.RefreshToken) // получить новый access_token
		auth.POST("/logout", authHandler.Logout)        // разлогиниться (удалить refresh)
	}

	// 👤 Users (любая авторизация)
	users := r.Group("/users")
	users.Use(middleware.JWTAuthMiddleware())
	{
		users.GET("/:id", userHandler.GetUser)
		users.PUT("/:id", userHandler.UpdateUser)
		users.DELETE("/:id", userHandler.DeleteUser)
	}

	// 👗 Model Profile (только для модели)
	modelProfile := r.Group("/model-profiles")
	modelProfile.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("model"))
	{
		modelProfile.POST("/", modelProfileHandler.CreateProfile)
		modelProfile.GET("/:user_id", modelProfileHandler.GetProfile)
	}

	// 🏢 Employer Profile (только для работодателя)
	employerProfile := r.Group("/employer-profiles")
	employerProfile.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("employer"))
	{
		employerProfile.POST("/", employerProfileHandler.CreateProfile)
		employerProfile.GET("/:user_id", employerProfileHandler.GetProfile)
	}

	// 🎬 Casting (только для работодателя)
	casting := r.Group("/castings")
	casting.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("employer"))
	{
		casting.POST("/", castingHandler.Create)
		casting.GET("/employer", castingHandler.ListByEmployer)
	}

	// 📩 Response (только для модели)
	response := r.Group("/responses")
	response.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("model"))
	{
		response.POST("/", responseHandler.Create)
		response.GET("/", responseHandler.ListByCasting)
		response.GET("/:id", responseHandler.GetByID)
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

	// 💳 Subscriptions (авторизованные пользователи)
	subscription := r.Group("/subscriptions")
	subscription.Use(middleware.JWTAuthMiddleware())
	{
		subscription.GET("/plans", subscriptionHandler.GetPlans)
		subscription.GET("/user/:userID", subscriptionHandler.GetUserSubscription)
		subscription.POST("/create", subscriptionHandler.CreateSubscription)
		subscription.GET("/check-usage", subscriptionHandler.CheckUsageLimit)
	}

	// 📤 Uploads (для всех авторизованных)
	upload := r.Group("/uploads")
	upload.Use(middleware.JWTAuthMiddleware())
	{
		upload.POST("/", uploadHandler.Upload)
	}

	// 📊 Analytics (только для модели)
	analytics := r.Group("/analytics")
	analytics.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("model"))
	{
		analytics.GET("/model", analyticsHandler.GetModelAnalytics)
	}

	return r
}

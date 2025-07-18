package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/middlewares"
)

func SetupAdminRoutes(
	r *gin.Engine,
	userHandler *handlers.UserHandler,
	modelProfileHandler *handlers.ModelProfileHandler,
	employerProfileHandler *handlers.EmployerProfileHandler,
	castingHandler *handlers.CastingHandler,
	subscriptionHandler *handlers.SubscriptionHandler,
	uploadHandler *handlers.UploadHandler,
	analyticsHandler *handlers.AnalyticsHandler,
) {
	admin := r.Group("/admin")
	admin.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("admin"))
	{
		// 📋 Users
		admin.DELETE("/users/:id", userHandler.DeleteUser)

		// 💳 Subscriptions
		admin.GET("/subscriptions/stats", subscriptionHandler.GetSubscriptionStats)
		admin.POST("/subscriptions/force-cancel/:id", subscriptionHandler.ForceCancelSubscription)
		admin.POST("/subscriptions/force-extend/:id", subscriptionHandler.ForceExtendSubscription)
		admin.POST("/plans", subscriptionHandler.CreatePlan)
		admin.DELETE("/plans/:id", subscriptionHandler.DeletePlan)

	}
}

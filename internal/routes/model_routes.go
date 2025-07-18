package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/middlewares"
)

func SetupModelRoutes(
	r *gin.Engine,
	modelProfileHandler *handlers.ModelProfileHandler,
	responseHandler *handlers.ResponseHandler,
	analyticsHandler *handlers.AnalyticsHandler,
) {
	// 👗 Model Profile (только для модели)
	modelProfile := r.Group("/model-profiles")
	modelProfile.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("model"))
	{
		modelProfile.POST("/", modelProfileHandler.CreateProfile)
		modelProfile.GET("/:user_id", modelProfileHandler.GetProfile)
	}

	// 📩 Response (только для модели)
	response := r.Group("/responses")
	response.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("model"))
	{
		response.POST("/", responseHandler.Create)
		response.GET("/", responseHandler.ListByCasting)
		response.GET("/:id", responseHandler.GetByID)
	}

	// 📊 Analytics (только для модели)
	analytics := r.Group("/analytics")
	analytics.Use(middleware.JWTAuthMiddleware(), middleware.RequireRoles("model"))
	{
		analytics.GET("/model", analyticsHandler.GetModelAnalytics)
	}
}

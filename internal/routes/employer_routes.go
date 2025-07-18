package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/middlewares"
)

func SetupEmployerRoutes(
	r *gin.Engine,
	employerProfileHandler *handlers.EmployerProfileHandler,
	castingHandler *handlers.CastingHandler,
) {
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
}

package routes

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/handlers"
)

func RegisterAllRoutes(
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
) {
	// 🔓 Общие маршруты (регистрация, логин, публичные запросы)
	SetupCommonRoutes(r, userHandler, chatHandler, uploadHandler, subscriptionHandler)

	// 🧍 Роль: model
	SetupModelRoutes(r, modelProfileHandler, responseHandler, analyticsHandler)

	// 🧑‍💼 Роль: employer
	SetupEmployerRoutes(r, employerProfileHandler, castingHandler)

	// 🛡️ Админ
	SetupAdminRoutes(
		r,
		userHandler,
		modelProfileHandler,
		employerProfileHandler,
		castingHandler,
		subscriptionHandler,
		uploadHandler,
		analyticsHandler,
	)

	SetupPublicRoutes(r, authHandler)
}

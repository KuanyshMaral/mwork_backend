package routes

import (
	"github.com/gin-gonic/gin"
	// "github.com/gorilla/mux" // <-- Удалено
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/services"
	"mwork_backend/internal/validator" // <-- 1. Добавлен импорт валидатора
)

type Handlers struct {
	UserHandler         *handlers.UserHandler
	ProfileHandler      *handlers.ProfileHandler
	CastingHandler      *handlers.CastingHandler
	ResponseHandler     *handlers.ResponseHandler
	ReviewHandler       *handlers.ReviewHandler
	PortfolioHandler    *handlers.PortfolioHandler
	MatchingHandler     *handlers.MatchingHandler
	NotificationHandler *handlers.NotificationHandler
	SubscriptionHandler *handlers.SubscriptionHandler
	SearchHandler       *handlers.SearchHandler
	AnalyticsHandler    *handlers.AnalyticsHandler
	ChatHandler         *handlers.ChatHandler
}

// NewHandlers - 2. Полностью переработан для внедрения BaseHandler
func NewHandlers(
	userService services.UserService,
	profileService services.ProfileService,
	castingService services.CastingService,
	responseService services.ResponseService,
	reviewService services.ReviewService,
	portfolioService services.PortfolioService,
	matchingService services.MatchingService,
	notificationService services.NotificationService,
	subscriptionService services.SubscriptionService,
	searchService services.SearchService,
	analyticsService services.AnalyticsService,
	chatService services.ChatService,
	authService services.AuthService,
) *Handlers {

	// 3. Создаем BaseHandler один раз
	customValidator := validator.New()
	baseHandler := handlers.NewBaseHandler(customValidator)

	// 4. Внедряем baseHandler во все конструкторы
	return &Handlers{
		UserHandler:         handlers.NewUserHandler(baseHandler, userService, authService),
		ProfileHandler:      handlers.NewProfileHandler(baseHandler, profileService),
		CastingHandler:      handlers.NewCastingHandler(baseHandler, castingService),
		ResponseHandler:     handlers.NewResponseHandler(baseHandler, responseService),
		ReviewHandler:       handlers.NewReviewHandler(baseHandler, reviewService),
		PortfolioHandler:    handlers.NewPortfolioHandler(baseHandler, portfolioService),
		MatchingHandler:     handlers.NewMatchingHandler(baseHandler, matchingService),
		NotificationHandler: handlers.NewNotificationHandler(baseHandler, notificationService),
		SubscriptionHandler: handlers.NewSubscriptionHandler(baseHandler, subscriptionService),
		SearchHandler:       handlers.NewSearchHandler(baseHandler, searchService),
		AnalyticsHandler:    handlers.NewAnalyticsHandler(baseHandler, analyticsService),
		ChatHandler:         handlers.NewChatHandler(baseHandler, chatService),
	}
}

// RegisterGinRoutes регистрирует все Gin-based routes
func (h *Handlers) RegisterGinRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// User and Auth routes
	h.UserHandler.RegisterRoutes(api)

	// Profile routes
	h.ProfileHandler.RegisterRoutes(api)

	// Casting routes
	h.CastingHandler.RegisterRoutes(api)

	// Response routes
	h.ResponseHandler.RegisterRoutes(api)

	// Review routes
	h.ReviewHandler.RegisterRoutes(api)

	// Portfolio routes
	h.PortfolioHandler.RegisterRoutes(api)

	// Matching routes
	h.MatchingHandler.RegisterRoutes(api)

	// Notification routes
	h.NotificationHandler.RegisterRoutes(api)

	// Subscription routes
	h.SubscriptionHandler.RegisterRoutes(api)

	// Search routes
	h.SearchHandler.RegisterRoutes(api)

	// Analytics routes
	h.registerAnalyticsRoutes(api) // <-- registerAnalyticsRoutes уже определен ниже

	// Chat routes
	h.ChatHandler.RegisterRoutes(api)
}

// registerAnalyticsRoutes регистрирует analytics routes for Gin
func (h *Handlers) registerAnalyticsRoutes(r *gin.RouterGroup) {
	analytics := r.Group("/analytics")
	{
		// Platform overview
		analytics.GET("/platform/overview", h.AnalyticsHandler.GetPlatformOverview)
		analytics.GET("/platform/growth", h.AnalyticsHandler.GetPlatformGrowthMetrics)
		analytics.GET("/platform/health", h.AnalyticsHandler.GetPlatformHealthMetrics)

		// User analytics
		analytics.GET("/users", h.AnalyticsHandler.GetUserAnalytics)
		analytics.GET("/users/acquisition", h.AnalyticsHandler.GetUserAcquisitionMetrics)
		analytics.GET("/users/retention", h.AnalyticsHandler.GetUserRetentionMetrics)
		analytics.GET("/users/active/count", h.AnalyticsHandler.GetActiveUsersCount)

		// Casting analytics
		analytics.GET("/castings", h.AnalyticsHandler.GetCastingAnalytics)
		analytics.GET("/castings/:employer_id/performance", h.AnalyticsHandler.GetCastingPerformanceMetrics)

		// Matching analytics
		analytics.GET("/matching", h.AnalyticsHandler.GetMatchingAnalytics)
		analytics.GET("/matching/efficiency", h.AnalyticsHandler.GetMatchingEfficiencyMetrics)

		// Financial analytics
		analytics.GET("/financial", h.AnalyticsHandler.GetFinancialAnalytics)

		// Geographic analytics
		analytics.GET("/geographic", h.AnalyticsHandler.GetGeographicAnalytics)
		analytics.GET("/geographic/cities", h.AnalyticsHandler.GetCityPerformanceMetrics)

		// Category analytics
		analytics.GET("/categories", h.AnalyticsHandler.GetCategoryAnalytics)
		analytics.GET("/categories/popular", h.AnalyticsHandler.GetPopularCategories)

		// Performance metrics
		analytics.GET("/performance", h.AnalyticsHandler.GetPerformanceMetrics)
		analytics.GET("/realtime", h.AnalyticsHandler.GetRealTimeMetrics)

		// System health
		analytics.GET("/system/health", h.AnalyticsHandler.GetSystemHealthMetrics)

		// Admin dashboard
		analytics.GET("/admin/dashboard", h.AnalyticsHandler.GetAdminDashboard)

		// Custom reports
		analytics.POST("/reports/custom", h.AnalyticsHandler.GenerateCustomReport)
		analytics.GET("/reports/predefined", h.AnalyticsHandler.GetPredefinedReports)
	}
}

// SetupRoutes initializes all routes for Gin
func SetupRoutes(ginRouter *gin.Engine, handlers *Handlers) {
	// Register Gin routes
	handlers.RegisterGinRoutes(ginRouter)
}

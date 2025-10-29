package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/services"
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

func NewHandlers(
	userService *services.UserService,
	profileService *services.ProfileService,
	castingService *services.CastingService,
	responseService services.ResponseService,
	reviewService services.ReviewService,
	portfolioService services.PortfolioService,
	matchingService services.MatchingService,
	notificationService services.NotificationService,
	subscriptionService services.SubscriptionService,
	searchService services.SearchService,
	analyticsService services.AnalyticsService,
	chatService services.ChatService,
) *Handlers {
	return &Handlers{
		UserHandler:         handlers.NewUserHandler(userService),
		ProfileHandler:      handlers.NewProfileHandler(profileService),
		CastingHandler:      handlers.NewCastingHandler(castingService),
		ResponseHandler:     handlers.NewResponseHandler(responseService),
		ReviewHandler:       handlers.NewReviewHandler(reviewService),
		PortfolioHandler:    handlers.NewPortfolioHandler(portfolioService),
		MatchingHandler:     handlers.NewMatchingHandler(matchingService),
		NotificationHandler: handlers.NewNotificationHandler(notificationService),
		SubscriptionHandler: handlers.NewSubscriptionHandler(subscriptionService),
		SearchHandler:       handlers.NewSearchHandler(searchService),
		AnalyticsHandler:    handlers.NewAnalyticsHandler(analyticsService),
		ChatHandler:         handlers.NewChatHandler(chatService),
	}
}

// RegisterGinRoutes registers all Gin-based routes
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

	// Analytics routes (if using Gin - needs conversion from current implementation)
	h.registerAnalyticsRoutes(api)
}

// RegisterMuxRoutes registers all Gorilla Mux-based routes (for Chat handler)
func (h *Handlers) RegisterMuxRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// Chat routes
	h.ChatHandler.RegisterRoutes(api)
}

// registerAnalyticsRoutes registers analytics routes for Gin
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

// SetupRoutes initializes all routes for both Gin and Mux routers
func SetupRoutes(ginRouter *gin.Engine, muxRouter *mux.Router, handlers *Handlers) {
	// Register Gin routes
	handlers.RegisterGinRoutes(ginRouter)

	// Register Mux routes (for Chat)
	handlers.RegisterMuxRoutes(muxRouter)
}

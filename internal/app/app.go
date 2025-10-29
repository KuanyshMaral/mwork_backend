package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mwork_backend/database"
	"mwork_backend/internal/config"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services"
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// AppHandlers содержит все хендлеры приложения
type AppHandlers struct {
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

func Run() {
	// Загружаем конфигурацию
	config.LoadConfig()
	cfg := config.AppConfig

	// Подключение к БД
	fmt.Println("👉 Строка подключения к БД:", cfg.Database.DSN)

	// GORM подключение
	gormDB, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к GORM: %v", err)
	}

	// Автоматическая миграция моделей
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ AutoMigrate ошибка: %v", err)
	}
	fmt.Println("✅ AutoMigrate выполнен успешно")

	// Стандартный sql.DB
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Ошибка получения *sql.DB из GORM: %v", err)
	}
	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}
	fmt.Println("✅ База данных подключена")

	ctx := context.Background()

	// Инициализация сервисов
	serviceContainer := initializeServices(cfg, gormDB, sqlDB)

	// Инициализация хендлеров
	appHandlers := initializeHandlers(serviceContainer)

	// 💬 WebSocket
	wsManager := ws.NewWebSocketManager(
		serviceContainer.ChatService,
	)
	go wsManager.Run()

	wsHandler := ws.NewWebSocketHandler(
		wsManager,
		serviceContainer.ChatService,
	)

	// Инициализация роутеров
	ginRouter := initializeGinRouter()

	// Настройка маршрутов
	setupRoutes(ginRouter, appHandlers, wsHandler)

	// Запускаем сервер
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("🚀 Сервер запущен на %s\n", address)

	if err := ginRouter.Run(address); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// ServiceContainer содержит все сервисы приложения
type ServiceContainer struct {
	UserService         *services.UserService
	ProfileService      *services.ProfileService
	CastingService      *services.CastingService
	ResponseService     *services.ResponseService
	ReviewService       *services.ReviewService
	PortfolioService    *services.PortfolioService
	MatchingService     *services.MatchingService
	NotificationService *services.NotificationService
	SubscriptionService *services.SubscriptionService
	SearchService       *services.SearchService
	AnalyticsService    *services.AnalyticsService
	ChatService         services.ChatService
	EmailService        *services.EmailService
}

// initializeServices инициализирует все сервисы приложения
func initializeServices(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB) *ServiceContainer {
	// Инициализация email service
	emailService := services.NewEmailService(cfg.SMTP)

	// Репозитории
	userRepo := repositories.NewUserRepository(gormDB)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(gormDB)
	profileRepo := repositories.NewProfileRepository(gormDB)
	castingRepo := repositories.NewCastingRepository(gormDB)
	responseRepo := repositories.NewResponseRepository(gormDB)
	notificationRepo := repositories.NewNotificationRepository(gormDB)
	portfolioRepo := repositories.NewPortfolioRepository(gormDB)
	reviewRepo := repositories.NewReviewRepository(gormDB)
	uploadRepo := repositories.NewUploadRepository(gormDB)
	analyticsRepo := repositories.NewAnalyticsRepository(gormDB)
	subscriptionRepo := repositories.NewSubscriptionRepository(gormDB)
	chatRepo := repositories.NewChatRepository(gormDB)

	// Сервисы
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, emailService)
	profileService := services.NewProfileService(profileRepo)
	castingService := services.NewCastingService(castingRepo)
	responseService := services.NewResponseService(responseRepo)
	notificationService := services.NewNotificationService(notificationRepo, emailService)
	portfolioService := services.NewPortfolioService(portfolioRepo)
	reviewService := services.NewReviewService(reviewRepo)
	searchService := services.NewSearchService(castingRepo, profileRepo)
	matchingService := services.NewMatchingService(castingRepo, profileRepo, notificationService)
	analyticsService := services.NewAnalyticsService(analyticsRepo)
	uploadService := services.NewUploadService(uploadRepo)
	moderationService := services.NewModerationService(userRepo, profileRepo, castingRepo)
	usageService := services.NewUsageService(subscriptionRepo)
	subscriptionService := services.NewSubscriptionService(subscriptionRepo)
	chatService := services.NewChatService(chatRepo)

	return &ServiceContainer{
		UserService:         userService,
		AuthService:         authService,
		ProfileService:      profileService,
		CastingService:      castingService,
		ResponseService:     responseService,
		ReviewService:       reviewService,
		PortfolioService:    portfolioService,
		MatchingService:     matchingService,
		NotificationService: notificationService,
		SubscriptionService: subscriptionService,
		SearchService:       searchService,
		AnalyticsService:    analyticsService,
		ChatService:         chatService,
		EmailService:        emailService,
		UploadService:       uploadService,
		ModerationService:   moderationService,
		UsageService:        usageService,
	}
}

// initializeHandlers инициализирует все хендлеры приложения
func initializeHandlers(services *ServiceContainer) *AppHandlers {
	return &AppHandlers{
		UserHandler:         handlers.NewUserHandler(services.UserService, services.AuthService),
		ProfileHandler:      handlers.NewProfileHandler(services.ProfileService),
		CastingHandler:      handlers.NewCastingHandler(services.CastingService),
		ResponseHandler:     handlers.NewResponseHandler(services.ResponseService),
		ReviewHandler:       handlers.NewReviewHandler(services.ReviewService),
		PortfolioHandler:    handlers.NewPortfolioHandler(services.PortfolioService),
		MatchingHandler:     handlers.NewMatchingHandler(services.MatchingService),
		NotificationHandler: handlers.NewNotificationHandler(services.NotificationService),
		SubscriptionHandler: handlers.NewSubscriptionHandler(services.SubscriptionService),
		SearchHandler:       handlers.NewSearchHandler(services.SearchService),
		AnalyticsHandler:    handlers.NewAnalyticsHandler(services.AnalyticsService),
		ChatHandler:         handlers.NewChatHandler(services.ChatService),
		UploadHandler:       handlers.NewUploadHandler(services.UploadService),
		ModerationHandler:   handlers.NewModerationHandler(services.ModerationService),
	}
}

// initializeGinRouter инициализирует и настраивает Gin роутер
func initializeGinRouter() *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.CORSMiddleware())

	return router
}

// setupRoutes настраивает все маршруты приложения
func setupRoutes(ginRouter *gin.Engine, handlers *AppHandlers, wsHandler *ws.WebSocketHandler) {
	// Регистрация API маршрутов
	api := ginRouter.Group("/api/v1")

	// User and Auth routes
	handlers.UserHandler.RegisterRoutes(api)

	// Profile routes
	handlers.ProfileHandler.RegisterRoutes(api)

	// Casting routes
	handlers.CastingHandler.RegisterRoutes(api)

	// Response routes
	handlers.ResponseHandler.RegisterRoutes(api)

	// Review routes
	handlers.ReviewHandler.RegisterRoutes(api)

	// Portfolio routes
	handlers.PortfolioHandler.RegisterRoutes(api)

	// Matching routes
	handlers.MatchingHandler.RegisterRoutes(api)

	// Notification routes
	handlers.NotificationHandler.RegisterRoutes(api)

	// Subscription routes
	handlers.SubscriptionHandler.RegisterRoutes(api)

	// Search routes
	handlers.SearchHandler.RegisterRoutes(api)

	// Analytics routes
	handlers.AnalyticsHandler.RegisterRoutes(api)

	// Upload routes
	handlers.UploadHandler.RegisterRoutes(api)

	// Moderation routes
	handlers.ModerationHandler.RegisterRoutes(api)

	// Chat routes
	handlers.ChatHandler.RegisterRoutes(api)

	// WebSocket маршруты
	setupWebSocketRoutes(ginRouter, wsHandler)
}

// setupWebSocketRoutes настраивает WebSocket маршруты
func setupWebSocketRoutes(router *gin.Engine, wsHandler *ws.WebSocketHandler) {
	wsGroup := router.Group("/ws")
	{
		wsGroup.GET("/chat", func(c *gin.Context) {
			wsHandler.HandleWebSocket(c.Writer, c.Request)
		})
		wsGroup.GET("/chat/:dialog_id", func(c *gin.Context) {
			wsHandler.HandleWebSocket(c.Writer, c.Request)
		})
	}
}

package app

import (
	"database/sql"
	"fmt"
	"log"
	"mwork_backend/internal/config"
	"mwork_backend/internal/email"
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
	ChatHandler         *handlers.ChatHandler // Этот хендлер теперь будет для /api/v1/chat/... (не-WS)
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

	/*// Автоматическая миграция моделей
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ AutoMigrate ошибка: %v", err)
	}
	fmt.Println("✅ AutoMigrate выполнен успешно")
	*/

	// Стандартный sql.DB
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Ошибка получения *sql.DB из GORM: %v", err)
	}
	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}
	fmt.Println("✅ База данных подключена")

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
	)

	// Инициализация роутеров
	ginRouter := initializeGinRouter()

	// Настройка маршрутов
	// ПРИМЕЧАНИЕ: wsHandler передается отдельно от appHandlers
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
	UserService         services.UserService
	AuthService         services.AuthService
	ProfileService      services.ProfileService
	CastingService      services.CastingService
	ResponseService     services.ResponseService
	ReviewService       services.ReviewService
	PortfolioService    services.PortfolioService
	MatchingService     services.MatchingService
	NotificationService services.NotificationService
	SubscriptionService services.SubscriptionService
	SearchService       services.SearchService
	AnalyticsService    services.AnalyticsService
	ChatService         services.ChatService
	EmailService        email.Provider
}

// initializeServices инициализирует все сервисы приложения
func initializeServices(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB) *ServiceContainer {
	// 1. Создаем структуру конфигурации для EmailService
	emailServiceConfig := services.EmailServiceConfig{
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUsername: cfg.Email.SMTPUsername,
		SMTPPassword: cfg.Email.SMTPPassword,
		FromEmail:    cfg.Email.FromEmail,
		FromName:     cfg.Email.FromName,
		UseTLS:       cfg.Email.UseTLS,
		TemplatesDir: cfg.Email.TemplatesDir,
	}

	// 2. Используем правильный конструктор NewEmailServiceWithConfig
	emailService, err := services.NewEmailServiceWithConfig(emailServiceConfig)
	if err != nil {
		log.Fatalf("Ошибка инициализации EmailService: %v", err)
	}

	// Репозитории
	userRepo := repositories.NewUserRepository(gormDB)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(gormDB)
	profileRepo := repositories.NewProfileRepository(gormDB)
	castingRepo := repositories.NewCastingRepository(gormDB)
	responseRepo := repositories.NewResponseRepository(gormDB)
	notificationRepo := repositories.NewNotificationRepository(gormDB)
	portfolioRepo := repositories.NewPortfolioRepository(gormDB)
	reviewRepo := repositories.NewReviewRepository(gormDB)
	subscriptionRepo := repositories.NewSubscriptionRepository(gormDB)
	chatRepo := repositories.NewChatRepository(gormDB)
	analyticsRepo := repositories.NewAnalyticsRepository(gormDB)

	// Сервисы
	userService := services.NewUserService(userRepo, profileRepo)
	authService := services.NewAuthService(
		userRepo,
		profileRepo,
		subscriptionRepo,
		emailService,
		refreshTokenRepo,
	)
	profileService := services.NewProfileService(
		profileRepo,
		userRepo,
		portfolioRepo,
		reviewRepo,
		notificationRepo,
	)
	castingService := services.NewCastingService(
		castingRepo,
		userRepo,
		profileRepo,
		subscriptionRepo,
		notificationRepo,
		reviewRepo,
		responseRepo,
	)
	responseService := services.NewResponseService(
		responseRepo,
		castingRepo,
		userRepo,
		subscriptionRepo,
		notificationRepo,
		reviewRepo,
	)
	notificationService := services.NewNotificationService(
		notificationRepo,
		userRepo,
		profileRepo,
	)
	portfolioService := services.NewPortfolioService(
		portfolioRepo,
		userRepo,
		profileRepo,
	)
	reviewService := services.NewReviewService(
		reviewRepo,
		userRepo,
		profileRepo,
		castingRepo,
		notificationRepo,
	)
	searchService := services.NewSearchService(
		castingRepo,
		profileRepo,
		portfolioRepo,
		reviewRepo,
	)
	matchingService := services.NewMatchingService(
		profileRepo,
		castingRepo,
		reviewRepo,
		portfolioRepo,
		notificationRepo,
	)
	analyticsService := services.NewAnalyticsService(
		userRepo,
		profileRepo,
		castingRepo,
		reviewRepo,
		notificationRepo,
		portfolioRepo,
		subscriptionRepo,
		chatRepo,
		analyticsRepo,
	)
	subscriptionService := services.NewSubscriptionService(
		subscriptionRepo,
		userRepo,
		notificationRepo,
	)
	chatService := services.NewChatService(
		chatRepo,
		userRepo,
		castingRepo,
		profileRepo,
		notificationRepo,
	)

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
	}
}

// initializeHandlers инициализирует все хендлеры приложения
func initializeHandlers(services *ServiceContainer) *AppHandlers {
	return &AppHandlers{
		// <-- ИСПРАВЛЕНО: Добавлен services.AuthService
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
		// ChatHandler используется для API-маршрутов чата (например, /api/v1/chat/history)
		ChatHandler: handlers.NewChatHandler(services.ChatService),
	}
}

// initializeGinRouter инициализирует и настраивает Gin роутер
func initializeGinRouter() *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(middleware.CORSMiddleware())

	return router
}

// setupRoutes настраивает все маршруты приложения
func setupRoutes(ginRouter *gin.Engine, handlers *AppHandlers, wsHandler *ws.WebSocketHandler) {
	// Регистрация API маршрутов
	api := ginRouter.Group("/api/v1")

	// ВСЕ handler.RegisterRoutes ДОЛЖНЫ принимать *gin.RouterGroup
	// Убедитесь, что *каждый* хендлер в пакете 'handlers'
	// имеет метод RegisterRoutes(router *gin.RouterGroup)
	handlers.UserHandler.RegisterRoutes(api)
	handlers.ProfileHandler.RegisterRoutes(api)
	handlers.CastingHandler.RegisterRoutes(api)
	handlers.ResponseHandler.RegisterRoutes(api)
	handlers.ReviewHandler.RegisterRoutes(api)
	handlers.PortfolioHandler.RegisterRoutes(api)
	handlers.MatchingHandler.RegisterRoutes(api)
	handlers.NotificationHandler.RegisterRoutes(api)
	handlers.SubscriptionHandler.RegisterRoutes(api)
	handlers.SearchHandler.RegisterRoutes(api)
	handlers.AnalyticsHandler.RegisterRoutes(api)

	// ИСПРАВЛЕНИЕ: ChatHandler регистрирует свои *API* маршруты (например, история чата)
	// WebSocket маршрут регистрируется отдельно ниже.
	handlers.ChatHandler.RegisterRoutes(api)

	// WebSocket маршруты
	// ИСПРАВЛЕНИЕ: Вызываем функцию, которая теперь определена
	setupWebSocketRoutes(ginRouter, wsHandler)
}

// setupWebSocketRoutes настраивает маршруты для WebSocket
// ИСПРАВЛЕНИЕ: Реализация недостающей функции
func setupWebSocketRoutes(ginRouter *gin.Engine, wsHandler *ws.WebSocketHandler) {
	// Вы можете поместить /ws в /api/v1/ws, если хотите
	// apiV1 := ginRouter.Group("/api/v1")
	// apiV1.GET("/ws", wsHandler.ServeWS)

	// Или оставить его в корне
	ginRouter.GET("/ws", wsHandler.ServeWS)
	fmt.Println("🔌 WebSocket маршрут /ws зарегистрирован")
}

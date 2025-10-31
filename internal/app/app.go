package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"mwork_backend/internal/config"
	"mwork_backend/internal/email"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/logger"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services"
	"mwork_backend/internal/storage"
	"mwork_backend/internal/validator"
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// AppHandlers (без изменений)
type AppHandlers struct {
	// ...
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
	FileHandler         *handlers.FileHandler
}

func Run() {
	// 1. Загружаем конфигурацию
	config.LoadConfig()
	cfg := config.AppConfig

	// 2. Инициализируем логгер
	logger.Init(cfg.Server.Env)
	logger.Info("Logger initialized", "env", cfg.Server.Env)

	// 3. Подключение к БД
	logger.Info("Connecting to database...", "dsn", cfg.Database.DSN)

	gormDB, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to GORM", "error", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		logger.Fatal("Failed to get *sql.DB from GORM", "error", err)
	}
	if err = sqlDB.Ping(); err != nil {
		logger.Fatal("Database unavailable", "error", err)
	}
	logger.Info("Database connected")

	// 4. ✅ Инициализация роутера (теперь вызывает новую функцию)
	ginRouter := SetupRouter(cfg, gormDB, sqlDB)

	// 5. Запускаем сервер
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info(fmt.Sprintf("🚀 Server starting on %s", address))

	if err := ginRouter.Run(address); err != nil {
		logger.Fatal("Server startup error", "error", err)
	}
}

// ✅
// 5. ✅ НОВАЯ ЭКСПОРТИРУЕМАЯ ФУНКЦИЯ, КОТОРУЮ БУДЕТ ИСПОЛЬЗОВАТЬ ТЕСТ
// ✅
func SetupRouter(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB) *gin.Engine {
	storageInstance, err := storage.NewStorage(storage.Config{
		Type:       cfg.Storage.Type,
		BasePath:   cfg.Storage.BasePath,
		BaseURL:    cfg.Storage.BaseURL,
		Bucket:     cfg.Storage.Bucket,
		Region:     cfg.Storage.Region,
		AccessKey:  cfg.Storage.AccessKey,
		SecretKey:  cfg.Storage.SecretKey,
		Endpoint:   cfg.Storage.Endpoint,
		UseSSL:     cfg.Storage.UseSSL,
		PublicRead: cfg.Storage.PublicRead,
	})
	if err != nil {
		logger.Fatal("Failed to initialize storage", "error", err)
	}
	logger.Info("Storage initialized", "type", cfg.Storage.Type)

	// Инициализация сервисов
	serviceContainer := initializeServices(cfg, gormDB, sqlDB, storageInstance)

	// Инициализация хендлеров (с BaseHandler)
	appHandlers := initializeHandlers(serviceContainer, storageInstance, gormDB)

	// WebSocket
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
	setupRoutes(ginRouter, appHandlers, wsHandler)

	return ginRouter
}

// ServiceContainer (без изменений)
type ServiceContainer struct {
	// ...
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
	storage             storage.Storage
}

// initializeServices (без изменений)
func initializeServices(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB, storageInstance storage.Storage) *ServiceContainer {
	// ... (без изменений) ...
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

	emailService, err := services.NewEmailServiceWithConfig(emailServiceConfig)
	if err != nil {
		logger.Fatal("Failed to initialize EmailService", "error", err)
	}

	// ... (все репозитории и сервисы как были) ...
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
		storageInstance,
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

// initializeHandlers (без изменений)
func initializeHandlers(services *ServiceContainer, storageInstance storage.Storage, gormDB *gorm.DB) *AppHandlers {
	customValidator := validator.New()
	baseHandler := handlers.NewBaseHandler(customValidator)
	portfolioRepo := repositories.NewPortfolioRepository(gormDB)

	return &AppHandlers{
		UserHandler:         handlers.NewUserHandler(baseHandler, services.UserService, services.AuthService),
		ProfileHandler:      handlers.NewProfileHandler(baseHandler, services.ProfileService),
		CastingHandler:      handlers.NewCastingHandler(baseHandler, services.CastingService),
		ResponseHandler:     handlers.NewResponseHandler(baseHandler, services.ResponseService),
		ReviewHandler:       handlers.NewReviewHandler(baseHandler, services.ReviewService),
		PortfolioHandler:    handlers.NewPortfolioHandler(baseHandler, services.PortfolioService),
		MatchingHandler:     handlers.NewMatchingHandler(baseHandler, services.MatchingService),
		NotificationHandler: handlers.NewNotificationHandler(baseHandler, services.NotificationService),
		SubscriptionHandler: handlers.NewSubscriptionHandler(baseHandler, services.SubscriptionService),
		SearchHandler:       handlers.NewSearchHandler(baseHandler, services.SearchService),
		AnalyticsHandler:    handlers.NewAnalyticsHandler(baseHandler, services.AnalyticsService),
		ChatHandler:         handlers.NewChatHandler(baseHandler, services.ChatService),
		FileHandler:         handlers.NewFileHandler(baseHandler, storageInstance, portfolioRepo), // Added FileHandler
	}
}

// initializeGinRouter (без изменений)
func initializeGinRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(middleware.CORSMiddleware())
	return router
}

// setupRoutes (без изменений)
func setupRoutes(ginRouter *gin.Engine, handlers *AppHandlers, wsHandler *ws.WebSocketHandler) {
	api := ginRouter.Group("/api/v1")

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
	handlers.ChatHandler.RegisterRoutes(api)

	setupWebSocketRoutes(ginRouter, wsHandler)
}

// setupWebSocketRoutes (без изменений)
func setupWebSocketRoutes(ginRouter *gin.Engine, wsHandler *ws.WebSocketHandler) {
	// ... (без изменений) ...
	ginRouter.GET("/ws", wsHandler.ServeWS)
	logger.Info("WebSocket route /ws registered")
}

// Middleware (RequestIDMiddleware, LoggingMiddleware) (без изменений)
func RequestIDMiddleware() gin.HandlerFunc {
	// ... (без изменений) ...
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func LoggingMiddleware() gin.HandlerFunc {
	// ... (без изменений) ...
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		log := logger.FromContext(c.Request.Context())
		fields := []any{
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.Int("status", c.Writer.Status()),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Duration("duration", duration),
			slog.Int("size_bytes", c.Writer.Size()),
		}
		if c.Writer.Status() >= 500 {
			log.Error("HTTP Server Error", fields...)
		} else if c.Writer.Status() >= 400 {
			log.Warn("HTTP Client Error", fields...)
		} else {
			log.Info("HTTP Request", fields...)
		}
	}
}

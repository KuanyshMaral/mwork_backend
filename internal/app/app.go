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
	"mwork_backend/pkg/contextkeys" // <-- ✅ 1. ДОБАВЛЕН ИМПОРТ
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ... (структура AppHandlers без изменений) ...
type AppHandlers struct {
	AuthHandler         *handlers.AuthHandler
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

// ... (функция Run без изменений) ...
func Run() {
	config.LoadConfig()
	cfg := config.AppConfig
	logger.Init(cfg.Server.Env)
	logger.Info("Logger initialized", "env", cfg.Server.Env)
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
	ginRouter := SetupRouter(cfg, gormDB, sqlDB)
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info(fmt.Sprintf("🚀 Server starting on %s", address))
	if err := ginRouter.Run(address); err != nil {
		logger.Fatal("Server startup error", "error", err)
	}
}

func SetupRouter(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB) *gin.Engine {
	// ... (storageInstance, serviceContainer, appHandlers, wsManager... без изменений) ...
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
	serviceContainer := initializeServices(cfg, gormDB, sqlDB, storageInstance)
	appHandlers := initializeHandlers(serviceContainer, storageInstance, gormDB)
	wsManager := ws.NewWebSocketManager(
		serviceContainer.ChatService,
		gormDB,
	)
	go wsManager.Run()
	wsHandler := ws.NewWebSocketHandler(
		wsManager,
	)

	// Инициализация роутеров
	ginRouter := initializeGinRouter(gormDB) // <-- ✅ 2. ПЕРЕДАЕМ gormDB СЮДА

	// Настройка маршрутов
	setupRoutes(ginRouter, appHandlers, wsHandler)

	return ginRouter
}

// ... (ServiceContainer, initializeServices, initializeHandlers... без изменений) ...
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
	storage             storage.Storage
}

func initializeServices(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB, storageInstance storage.Storage) *ServiceContainer {
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
	userRepo := repositories.NewUserRepository()
	refreshTokenRepo := repositories.NewRefreshTokenRepository()
	profileRepo := repositories.NewProfileRepository()
	castingRepo := repositories.NewCastingRepository()
	responseRepo := repositories.NewResponseRepository()
	notificationRepo := repositories.NewNotificationRepository()
	portfolioRepo := repositories.NewPortfolioRepository()
	reviewRepo := repositories.NewReviewRepository()
	subscriptionRepo := repositories.NewSubscriptionRepository()
	chatRepo := repositories.NewChatRepository()
	analyticsRepo := repositories.NewAnalyticsRepository()
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
		userRepo,
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
		responseRepo,
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

func initializeHandlers(services *ServiceContainer, storageInstance storage.Storage, gormDB *gorm.DB) *AppHandlers {
	customValidator := validator.New()
	baseHandler := handlers.NewBaseHandler(customValidator)
	portfolioRepo := repositories.NewPortfolioRepository()
	return &AppHandlers{
		AuthHandler:         handlers.NewAuthHandler(baseHandler, services.AuthService),
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
		FileHandler:         handlers.NewFileHandler(baseHandler, storageInstance, portfolioRepo),
	}
}

// initializeGinRouter (теперь принимает db)
func initializeGinRouter(db *gorm.DB) *gin.Engine { // <-- ✅ 3. ПРИНИМАЕМ gormDB
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(middleware.CORSMiddleware())

	router.Use(DBMiddleware(db)) // <-- ✅ 4. ДОБАВЛЯЕМ DBMiddleware

	return router
}

// ... (setupRoutes, setupWebSocketRoutes, RequestIDMiddleware, LoggingMiddleware... без изменений) ...
func setupRoutes(ginRouter *gin.Engine, handlers *AppHandlers, wsHandler *ws.WebSocketHandler) {
	api := ginRouter.Group("/api/v1")
	handlers.AuthHandler.RegisterRoutes(api)
	handlers.FileHandler.RegisterRoutes(api)
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

func setupWebSocketRoutes(ginRouter *gin.Engine, wsHandler *ws.WebSocketHandler) {
	ginRouter.GET("/ws", wsHandler.ServeWS)
	logger.Info("WebSocket route /ws registered")
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func LoggingMiddleware() gin.HandlerFunc {
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

// ✅ 5. ДОБАВЬ ЭТУ ФУНКЦИЮ В КОНЕЦ ФАЙЛА
//
// DBMiddleware добавляет *gorm.DB в контекст Gin
func DBMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ключ "db" (значение из contextkeys.DBContextKey) - это то,
		// что ищет твой BaseHandler.
		c.Set(string(contextkeys.DBContextKey), db)
		c.Next()
	}
}

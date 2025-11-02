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
	"mwork_backend/pkg/contextkeys"
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
	UploadHandler       *handlers.UploadHandler // <-- ✅ 1. ДОБАВЛЕН UploadHandler
}

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

	ginRouter := initializeGinRouter(gormDB)
	setupRoutes(ginRouter, appHandlers, wsHandler)

	return ginRouter
}

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
	UploadService       services.UploadService // <-- ✅ 2. ДОБАВЛЕН UploadService
	EmailService        email.Provider
	storage             storage.Storage
}

func initializeServices(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB, storageInstance storage.Storage) *ServiceContainer {
	/* ВРЕМЕННО ВЫКЛЮЧАЮ ВНЕШНИЙ СЕРВИС emailServiceConfig := services.EmailServiceConfig{
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

	*/

	// ❗️ 1. ОБЪЯВЛЯЕМ ИНТЕРФЕЙС
	var emailService email.Provider

	if cfg.Server.Env == "test" {
		// ❗️ 2. ЕСЛИ ЭТО ТЕСТ - ИСПОЛЬЗУЕМ MOCK
		logger.Info("Using MOCK Email Provider for test environment")
		emailService = &MockEmailProvider{} // (MockEmailProvider нужно будет создать)
	} else {
		// ❗️ 3. ЕСЛИ ЭТО PROD/DEV - ИСПОЛЬЗУЕМ НАСТОЯЩИЙ СЕРВИС
		emailServiceConfig := services.EmailServiceConfig{
			// ... (все ваши настройки)
			TemplatesDir: cfg.Email.TemplatesDir,
		}
		var err error
		emailService, err = services.NewEmailServiceWithConfig(emailServiceConfig)
		if err != nil {
			logger.Fatal("Failed to initialize EmailService", "error", err)
		}
	}

	// --- Инициализация репозиториев ---
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
	uploadRepo := repositories.NewUploadRepository() // <-- ✅ 3. Создаем UploadRepo

	// --- Инициализация сервисов ---

	// ✅ 4. Создаем UploadService
	uploadConfig := services.GetDefaultUploadConfig() // (Или из 'cfg' если настроено)
	uploadService := services.NewUploadService(uploadRepo, storageInstance, uploadConfig)

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

	// ▼▼▼ ИСПРАВЛЕНА ОШИБКА 1 ▼▼▼
	portfolioService := services.NewPortfolioService(
		portfolioRepo,
		userRepo,
		profileRepo,
		uploadService, // <-- ✅ 5. Передаем UploadService (вместо storageInstance)
	)
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

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

	// ▼▼▼ ИСПРАВЛЕНА ОШИБКА 2 ▼▼▼
	chatService := services.NewChatService(
		chatRepo,
		userRepo,
		castingRepo,
		profileRepo,
		notificationRepo,
		responseRepo,
		uploadService, // <-- ✅ 6. Добавляем UploadService
	)
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

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
		UploadService:       uploadService, // <-- ✅ 7. Добавляем в контейнер
		EmailService:        emailService,
	}
}

func initializeHandlers(services *ServiceContainer, storageInstance storage.Storage, gormDB *gorm.DB) *AppHandlers {
	customValidator := validator.New()
	baseHandler := handlers.NewBaseHandler(customValidator) // (DBMiddleware позаботится о 'db')

	// ▼▼▼ ИСПРАВЛЕНА ОШИБКА 3 ▼▼▼
	uploadRepo := repositories.NewUploadRepository() // <-- ✅ 8. Создаем UploadRepo
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

	return &AppHandlers{
		AuthHandler:         handlers.NewAuthHandler(baseHandler, services.AuthService),
		UserHandler:         handlers.NewUserHandler(baseHandler, services.UserService, services.AuthService),
		ProfileHandler:      handlers.NewProfileHandler(baseHandler, services.ProfileService),
		CastingHandler:      handlers.NewCastingHandler(baseHandler, services.CastingService, services.ResponseService),
		ResponseHandler:     handlers.NewResponseHandler(baseHandler, services.ResponseService),
		ReviewHandler:       handlers.NewReviewHandler(baseHandler, services.ReviewService),
		PortfolioHandler:    handlers.NewPortfolioHandler(baseHandler, services.PortfolioService),
		MatchingHandler:     handlers.NewMatchingHandler(baseHandler, services.MatchingService),
		NotificationHandler: handlers.NewNotificationHandler(baseHandler, services.NotificationService),
		SubscriptionHandler: handlers.NewSubscriptionHandler(baseHandler, services.SubscriptionService),
		SearchHandler:       handlers.NewSearchHandler(baseHandler, services.SearchService),
		AnalyticsHandler:    handlers.NewAnalyticsHandler(baseHandler, services.AnalyticsService),
		ChatHandler:         handlers.NewChatHandler(baseHandler, services.ChatService),
		// ▼▼▼ ИСПРАВЛЕНА ОШИБКА 3 ▼▼▼
		FileHandler:   handlers.NewFileHandler(baseHandler, storageInstance, uploadRepo), // <-- ✅ 9. Передаем uploadRepo
		UploadHandler: handlers.NewUploadHandler(baseHandler, services.UploadService),    // <-- ✅ 10. Создаем UploadHandler
		// ▲▲▲ ИСПРАВЛЕНО ▲▲▲
	}
}

// initializeGinRouter (сохраняем ваши изменения)
func initializeGinRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(DBMiddleware(db)) // <-- ✅ Сохранено
	return router
}

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
	handlers.UploadHandler.RegisterRoutes(api) // <-- ✅ 11. Регистрируем маршруты UploadHandler
	setupWebSocketRoutes(ginRouter, wsHandler)
}

func setupWebSocketRoutes(ginRouter *gin.Engine, wsHandler *ws.WebSocketHandler) {
	// (Этот маршрут должен быть защищен AuthMiddleware, если ServeWS ожидает userID)
	// ginRouter.GET("/ws", wsHandler.ServeWS)

	// ▼▼▼ ИСПРАВЛЕНИЕ: WS должен быть защищен ▼▼▼
	wsGroup := ginRouter.Group("/ws")
	wsGroup.Use(middleware.AuthMiddleware())
	{
		wsGroup.GET("", wsHandler.ServeWS)
	}
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

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

// DBMiddleware (сохраняем ваши изменения)
func DBMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbKey := string(contextkeys.DBContextKey)

		// 1. Пытаемся получить 'tx' (транзакцию) из контекста HTTP-запроса,
		//    которую туда положил testserver.go
		tx, ok := c.Request.Context().Value(contextkeys.DBContextKey).(*gorm.DB)

		if ok && tx != nil {
			// 2. ✅ УСПЕХ: Это тестовый запрос.
			//    Мы кладем в gin-контекст именно эту транзакцию 'tx'.
			c.Set(dbKey, tx)
		} else {
			// 3. ❌ ПРОВАЛ: Это обычный (не тестовый) запрос.
			//    Мы кладем в gin-контекст ОБЩИЙ пул 'db'.
			c.Set(dbKey, db)
		}

		c.Next()
	}
}

type MockEmailProvider struct{}

func (m *MockEmailProvider) Send(email *email.Email) error { return nil }
func (m *MockEmailProvider) SendWithTemplate(templateName string, data email.TemplateData, emailMsg *email.Email) error {
	return nil
}
func (m *MockEmailProvider) SendVerification(email string, token string) error { return nil }
func (m *MockEmailProvider) SendTemplate(to []string, subject string, templateName string, data email.TemplateData) error {
	return nil
}
func (m *MockEmailProvider) Validate() error { return nil }
func (m *MockEmailProvider) Close() error    { return nil }

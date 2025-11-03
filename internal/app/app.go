package app

import (
	"database/sql"
	"errors"
	"fmt"

	"mwork_backend/internal/config"
	"mwork_backend/internal/email"
	"mwork_backend/internal/handlers"
	"mwork_backend/internal/logger"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/routes"
	"mwork_backend/internal/services"
	"mwork_backend/internal/storage"
	"mwork_backend/internal/validator"
	"mwork_backend/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mwork_backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// ▼▼▼ УДАЛЕНЫ определения struct: AppHandlers и ServiceContainer ▼▼▼

func Run() {
	// ... (LoadConfig, Init, GORM, sqlDB... всё это остается как есть) ...
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

	if err := seedFirstAdmin(gormDB, cfg); err != nil {
		// Если не удалось создать админа (проблемы с БД и т.д.) - не запускаем сервер
		logger.Fatal("Failed to seed first admin user", "error", err)
	}

	// ▼▼▼ ИЗМЕНЕНИЕ: SetupRouter теперь просто возвращает *gin.Engine ▼▼▼
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

	// 1. Инициализируем сервисы
	serviceContainer := initializeServices(cfg, gormDB, sqlDB, storageInstance)

	// 2. Инициализируем хэндлеры
	appHandlers := initializeHandlers(serviceContainer, storageInstance, gormDB)

	// 3. Инициализируем WebSocket
	wsManager := ws.NewWebSocketManager(
		serviceContainer.ChatService,
		gormDB,
	)
	go wsManager.Run()
	wsHandler := ws.NewWebSocketHandler(
		wsManager,
	)

	// 4. Инициализируем Gin
	ginRouter := initializeGinRouter(gormDB)

	// 5. ▼▼▼ ГЛАВНОЕ ИЗМЕНЕНИЕ: Делегируем регистрацию маршрутов пакету 'routes' ▼▼▼
	routes.RegisterRoutes(ginRouter, appHandlers, wsHandler)
	// ▲▲▲

	return ginRouter
}

// ▼▼▼ ИЗМЕНЕНИЕ: Функция теперь возвращает *services.ServiceContainer ▼▼▼
func initializeServices(cfg *config.Config, gormDB *gorm.DB, sqlDB *sql.DB, storageInstance storage.Storage) *services.ServiceContainer {

	// ... (логика с MockEmailProvider остается) ...
	var emailService email.Provider
	logger.Warn("--- [ВРЕМЕННО] Email-сервис отключен. Используется MOCK. ---")
	emailService = &MockEmailProvider{} // (MockEmailProvider теперь в mocks.go)

	// --- Инициализация репозиториев ---
	// ... (NewUserRepository, NewRefreshTokenRepository... и т.д.) ...
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
	uploadRepo := repositories.NewUploadRepository()

	// --- Инициализация сервисов ---
	// ... (NewUploadService, NewUserService, NewAuthService... и т.д.) ...
	uploadConfig := services.GetDefaultUploadConfig()
	uploadService := services.NewUploadService(uploadRepo, storageInstance, uploadConfig)
	userService := services.NewUserService(userRepo, profileRepo)
	authService := services.NewAuthService(userRepo, profileRepo, subscriptionRepo, emailService, refreshTokenRepo)
	profileService := services.NewProfileService(profileRepo, userRepo, portfolioRepo, reviewRepo, notificationRepo)
	castingService := services.NewCastingService(castingRepo, userRepo, profileRepo, subscriptionRepo, notificationRepo, reviewRepo, responseRepo)
	responseService := services.NewResponseService(responseRepo, castingRepo, userRepo, subscriptionRepo, notificationRepo, reviewRepo)
	notificationService := services.NewNotificationService(notificationRepo, userRepo, profileRepo)
	portfolioService := services.NewPortfolioService(portfolioRepo, userRepo, profileRepo, uploadService)
	reviewService := services.NewReviewService(reviewRepo, userRepo, profileRepo, castingRepo, notificationRepo)
	searchService := services.NewSearchService(castingRepo, profileRepo, portfolioRepo, reviewRepo)
	matchingService := services.NewMatchingService(profileRepo, castingRepo, reviewRepo, portfolioRepo, notificationRepo, userRepo)
	analyticsService := services.NewAnalyticsService(userRepo, profileRepo, castingRepo, reviewRepo, notificationRepo, portfolioRepo, subscriptionRepo, chatRepo, analyticsRepo)
	subscriptionService := services.NewSubscriptionService(subscriptionRepo, userRepo, notificationRepo)
	chatService := services.NewChatService(chatRepo, userRepo, castingRepo, profileRepo, notificationRepo, responseRepo, uploadService)

	// ▼▼▼ ИЗМЕНЕНИЕ: Возвращаем *services.ServiceContainer ▼▼▼
	return &services.ServiceContainer{
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
		UploadService:       uploadService,
		EmailService:        emailService,
	}
}

// ▼▼▼ ИЗMENT: Принимает *services.ServiceContainer, возвращает *handlers.AppHandlers ▼▼▼
func initializeHandlers(services *services.ServiceContainer, storageInstance storage.Storage, gormDB *gorm.DB) *handlers.AppHandlers {
	customValidator := validator.New()
	baseHandler := handlers.NewBaseHandler(customValidator)

	uploadRepo := repositories.NewUploadRepository()

	// ▼▼▼ ИЗМЕНЕНИЕ: Возвращаем *handlers.AppHandlers ▼▼▼
	return &handlers.AppHandlers{
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
		FileHandler:         handlers.NewFileHandler(baseHandler, storageInstance, uploadRepo),
		UploadHandler:       handlers.NewUploadHandler(baseHandler, services.UploadService),
	}
}

// ▼▼▼ ИЗМЕНЕНИЕ: Используем middleware из пакета 'middleware' ▼▼▼
func initializeGinRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware()) // <-- ИЗМЕНЕНО
	router.Use(middleware.LoggingMiddleware())   // <-- ИЗМЕНЕНО
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.DBMiddleware(db)) // <-- ИЗМЕНЕНО
	return router
}

func seedFirstAdmin(db *gorm.DB, cfg *config.Config) error {
	adminEmail := cfg.FirstAdminEmail
	adminPassword := cfg.FirstAdminPassword

	if adminEmail == "" || adminPassword == "" {
		logger.Warn("FIRST_ADMIN_EMAIL or FIRST_ADMIN_PASSWORD is not set in .env. Skipping admin seeding.")
		return nil
	}

	// ⭐️ ИСПОЛЬЗУЕМ ТРАНЗАКЦИЮ (чтобы создать и юзера, и профиль)
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer tx.Rollback() // Откат, если что-то пойдет не так

	// 2. Ищем юзера (используем 'tx')
	var adminUser models.User
	result := tx.Where("email = ?", adminEmail).First(&adminUser)

	if result.Error == nil {
		logger.Info("Admin user already exists. Skipping creation.", "email", adminEmail)
		tx.Rollback() // Все в порядке, просто откатываем
		return nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check for admin user: %w", result.Error)
	}

	// 5. (gorm.ErrRecordNotFound) - Юзера нет. Создаем.
	logger.Warn("No admin user found with specified email. Creating first admin...", "email", adminEmail)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	newAdmin := &models.User{
		Email:        adminEmail,
		PasswordHash: string(hashedPassword),
		Role:         models.UserRoleAdmin,
		Status:       models.UserStatusActive,
		IsVerified:   true,
	}

	// 6. Сохраняем ЮЗЕРА (используем 'tx')
	if err := tx.Create(newAdmin).Error; err != nil {
		return fmt.Errorf("failed to create admin user in database: %w", err)
	}

	// 7. ⭐️ НОВОЕ: СОЗДАЕМ ПРОФИЛЬ РАБОТОДАТЕЛЯ ДЛЯ АДМИНА
	//    (Это нужно, чтобы удовлетворить 'fk_casting_employer')
	adminProfile := &models.EmployerProfile{
		UserID:      newAdmin.ID,
		CompanyName: "MWork Administration", // Можешь написать что угодно
		IsVerified:  true,
		City:        "Platform", // Можешь написать что угодно
	}

	if err := tx.Create(adminProfile).Error; err != nil {
		return fmt.Errorf("failed to create admin employer profile: %w", err)
	}
	// ⭐️ КОНЕЦ НОВОГО БЛОКА

	logger.Info("✅ Successfully created first admin user AND profile", "email", adminEmail)

	// 8. Коммитим транзакцию
	return tx.Commit().Error
}

// ▼▼▼ УДАЛЕНЫ: setupRoutes, setupWebSocketRoutes ▼▼▼
// ▼▼▼ УДАЛЕНЫ: RequestIDMiddleware, LoggingMiddleware, DBMiddleware ▼▼▼
// ▼▼▼ УДАЛЕНО: MockEmailProvider ▼▼▼

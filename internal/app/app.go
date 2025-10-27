package app

import (
	"context"
	"fmt"
	"log"
	"mwork_backend/database"
	"mwork_backend/internal/config"
	"mwork_backend/internal/handlers/old_shit"
	"mwork_backend/internal/middlewares"
	"mwork_backend/internal/repositories/old_bullshit"
	"mwork_backend/internal/repositories/old_bullshit/chat"
	"mwork_backend/internal/repositories/old_bullshit/subscription"
	"mwork_backend/internal/routes"
	"mwork_backend/internal/services"
	"mwork_backend/internal/utils"
	"mwork_backend/internal/workers"

	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	swaggerFiles "github.com/swaggo/files"
	_ "github.com/swaggo/gin-swagger"
	_ "mwork_backend/docs"

	chatservices "mwork_backend/internal/services/chat"

	subscriptionservices "mwork_backend/internal/services/subscription"

	ws "mwork_backend/ws"
)

func Run() {
	// Загружаем конфигурацию
	config.LoadConfig()
	cfg := config.AppConfig

	// Подключение к БД
	fmt.Println("👉 Строка подключения к БД:", cfg.Database.DSN)

	//gorm
	gormDB, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к GORM: %v", err)
	}

	// Автоматическая миграция моделей
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ AutoMigrate ошибка: %v", err)
	}
	fmt.Println("✅ AutoMigrate выполнен успешно")

	//стандартный sql
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Ошибка получения *sql.DB из GORM: %v", err)
	}
	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}
	fmt.Println("✅ База данных подключена")

	ctx := context.Background()

	castingWorker := workers.NewCastingWorker(gormDB)
	go castingWorker.Start(ctx)
	fmt.Println("✅ Casting worker started")

	subscriptionWorker := workers.NewSubscriptionWorker(gormDB)
	go subscriptionWorker.Start(ctx)
	fmt.Println("✅ Subscription worker started")

	// Инициализация email sender & service
	emailSender := utils.NewEmailSender(cfg)
	emailService := services.NewEmailService(emailSender)

	// User
	userRepo := old_bullshit.NewUserRepository(sqlDB)
	userService := services.NewUserService(userRepo)
	userHandler := old_shit.NewUserHandler(userService)

	// Refresh token
	refreshRepo := old_bullshit.NewRefreshTokenRepository(sqlDB)
	refreshService := services.NewRefreshTokenService(refreshRepo, userRepo)

	// Auth
	authService := services.NewAuthService(userRepo, emailService, refreshService)
	authHandler := old_shit.NewAuthHandler(authService)

	// Model profile
	modelProfileRepo := old_bullshit.NewModelProfileRepository(sqlDB)
	modelProfileService := services.NewModelProfileService(modelProfileRepo)
	modelProfileHandler := old_shit.NewModelProfileHandler(modelProfileService)

	// Employer profile
	employerProfileRepo := old_bullshit.NewEmployerProfileRepository(sqlDB)
	employerProfileService := services.NewEmployerProfileService(employerProfileRepo)
	employerProfileHandler := old_shit.NewEmployerProfileHandler(employerProfileService)

	castingRepoGorm := old_bullshit.NewCastingRepository(sqlDB)
	modelRepoGorm := old_bullshit.NewModelRepository(gormDB)
	responseRepoGorm := old_bullshit.NewResponseRepository(gormDB)
	notificationRepoGorm := old_bullshit.NewNotificationRepository(gormDB)
	portfolioRepoGorm := old_bullshit.NewPortfolioRepository(gormDB)
	reviewRepoGorm := old_bullshit.NewReviewRepository(gormDB)
	uploadRepoGorm := old_bullshit.NewUploadRepository(sqlDB)
	chatRepoGorm := old_bullshit.NewChatRepository(gormDB)

	// Casting
	castingRepo := old_bullshit.NewCastingRepository(sqlDB)
	castingService := services.NewCastingService(castingRepo)
	castingHandler := old_shit.NewCastingHandler(castingService)

	// Casting response
	responseRepo := old_bullshit.NewResponseRepository(sqlDB)
	responseService := services.NewResponseService(responseRepo)
	responseHandler := old_shit.NewResponseHandler(responseService)

	// 💬 Chat: репозитории
	dialogRepo := chat.NewDialogRepository(gormDB)
	participantRepo := chat.NewDialogParticipantRepository(gormDB)
	messageRepo := chat.NewMessageRepository(gormDB)
	attachmentRepo := chat.NewMessageAttachmentRepository(gormDB)
	reactionRepo := chat.NewMessageReactionRepository(gormDB)
	readReceiptRepo := chat.NewMessageReadReceiptRepository(gormDB)

	// 💬 Chat: сервисы
	chatService := chatservices.NewChatService(dialogRepo, participantRepo, messageRepo, readReceiptRepo)
	attachmentService := chatservices.NewAttachmentService(attachmentRepo)
	reactionService := chatservices.NewReactionService(reactionRepo)
	readReceiptService := chatservices.NewReadReceiptService(readReceiptRepo, messageRepo)

	// 💬 Chat: handler
	chatHandler := old_shit.NewChatHandler(chatService, attachmentService, reactionService, readReceiptService)

	// Subscription
	usersubscriptionRepo := subscription.NewUserSubscriptionRepository(sqlDB)
	plansubscriptionRepo := subscription.NewSubscriptionPlanRepository(sqlDB)
	usersubscriptionService := subscriptionservices.NewUserSubscriptionService(usersubscriptionRepo)
	plansubscriptionService := subscriptionservices.NewPlanService(plansubscriptionRepo)
	robokassaService := subscriptionservices.NewRobokassaService()

	subscriptionHandler := old_shit.NewSubscriptionHandler(plansubscriptionService, usersubscriptionService, robokassaService)

	notificationService := services.NewNotificationService(notificationRepoGorm, emailService)
	usageService := services.NewUsageService(usersubscriptionRepo)
	searchService := services.NewSearchService(castingRepoGorm, modelRepoGorm)
	matchingService := services.NewMatchingService(castingRepoGorm, modelRepoGorm, notificationService)
	portfolioService := services.NewPortfolioService(portfolioRepoGorm, uploadRepoGorm)
	reviewService := services.NewReviewService(reviewRepoGorm, modelRepoGorm, notificationService)
	moderationService := services.NewModerationService(userRepo, employerProfileRepo, castingRepoGorm)

	// Enhanced casting service with validation and transactions
	castingServiceEnhanced := services.NewCastingServiceEnhanced(
		gormDB,
		castingRepoGorm,
		usersubscriptionRepo,
		notificationService,
	)

	// Enhanced response service
	responseServiceEnhanced := services.NewResponseService(responseRepoGorm)
	responseServiceEnhanced.SetDependencies(castingRepoGorm, chatRepoGorm, notificationService, usersubscriptionRepo)

	searchHandler := old_shit.NewSearchHandler(searchService)
	matchingHandler := old_shit.NewMatchingHandler(matchingService)
	notificationHandler := old_shit.NewNotificationHandler(notificationService)
	portfolioHandler := old_shit.NewPortfolioHandler(portfolioService)
	reviewHandler := old_shit.NewReviewHandler(reviewService)
	moderationHandler := old_shit.NewModerationHandler(moderationService)

	// Upload
	uploadRepo := old_bullshit.NewUploadRepository(sqlDB)
	uploadService := services.NewUploadService(uploadRepo, "/mwork-front-fn/uploads", "/mwork-front-fn/uploads")
	uploadHandler := old_shit.NewUploadHandler(uploadService)

	// Analytics
	analyticsRepo := old_bullshit.NewAnalyticsRepository(sqlDB)
	analyticsService := services.NewAnalyticsService(analyticsRepo)
	analyticsHandler := old_shit.NewAnalyticsHandler(analyticsService, modelProfileRepo)

	// 💬 WebSocket
	wsManager := ws.NewWebSocketManager(chatService, attachmentService, reactionService, readReceiptService)
	go wsManager.Run()

	wsHandler := ws.NewWebSocketHandler(
		wsManager,
		chatService,
		attachmentService,
		reactionService,
		readReceiptService,
	)

	// Инициализируем Gin
	router := gin.Default()

	router.Use(middlewares.ErrorHandler())
	router.Use(middlewares.CORS())

	// Подключение WebSocket-маршрутов
	routes.SetupWebSocketRoutes(router, wsHandler)

	routes.RegisterAllRoutes(
		router,
		userHandler,
		authHandler,
		modelProfileHandler,
		employerProfileHandler,
		castingHandler,
		responseHandler,
		chatHandler,
		subscriptionHandler,
		uploadHandler,
		analyticsHandler,
		searchHandler,
		matchingHandler,
		notificationHandler,
		portfolioHandler,
		reviewHandler,
		moderationHandler,
		usageService,
	)

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Запускаем сервер
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("🚀 Сервер запущен на %s\n", address)
	if err := router.Run(address); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

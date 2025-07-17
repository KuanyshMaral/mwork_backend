package app

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"mwork_front_fn/database"

	_ "database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"log"
	"mwork_front_fn/internal/config"
	"mwork_front_fn/internal/handlers"
	"mwork_front_fn/internal/repositories"
	chatrepositories "mwork_front_fn/internal/repositories/chat"
	"mwork_front_fn/internal/routes"
	"mwork_front_fn/internal/services"
	chatservices "mwork_front_fn/internal/services/chat"
	"mwork_front_fn/internal/utils"

	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	_ "mwork_front_fn/docs"
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

	// Инициализация email sender & service
	emailSender := utils.NewEmailSender(cfg)
	emailService := services.NewEmailService(emailSender)

	// User
	userRepo := repositories.NewUserRepository(sqlDB)
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	// Refresh token
	refreshRepo := repositories.NewRefreshTokenRepository(sqlDB)
	refreshService := services.NewRefreshTokenService(refreshRepo, userRepo)

	// Auth
	authService := services.NewAuthService(userRepo, emailService, refreshService)
	authHandler := handlers.NewAuthHandler(authService)

	// Model profile
	modelProfileRepo := repositories.NewModelProfileRepository(sqlDB)
	modelProfileService := services.NewModelProfileService(modelProfileRepo)
	modelProfileHandler := handlers.NewModelProfileHandler(modelProfileService)

	// Employer profile
	employerProfileRepo := repositories.NewEmployerProfileRepository(sqlDB)
	employerProfileService := services.NewEmployerProfileService(employerProfileRepo)
	employerProfileHandler := handlers.NewEmployerProfileHandler(employerProfileService)

	// Casting
	castingRepo := repositories.NewCastingRepository(sqlDB)
	castingService := services.NewCastingService(castingRepo)
	castingHandler := handlers.NewCastingHandler(castingService)

	// Casting response
	responseRepo := repositories.NewResponseRepository(sqlDB)
	responseService := services.NewResponseService(responseRepo)
	responseHandler := handlers.NewResponseHandler(responseService)

	// 💬 Chat: репозитории
	dialogRepo := chatrepositories.NewDialogRepository(gormDB)
	participantRepo := chatrepositories.NewDialogParticipantRepository(gormDB)
	messageRepo := chatrepositories.NewMessageRepository(gormDB)
	attachmentRepo := chatrepositories.NewMessageAttachmentRepository(gormDB)
	reactionRepo := chatrepositories.NewMessageReactionRepository(gormDB)
	readReceiptRepo := chatrepositories.NewMessageReadReceiptRepository(gormDB)

	// 💬 Chat: сервисы
	chatService := chatservices.NewChatService(dialogRepo, participantRepo, messageRepo, readReceiptRepo)
	attachmentService := chatservices.NewAttachmentService(attachmentRepo)
	reactionService := chatservices.NewReactionService(reactionRepo)
	readReceiptService := chatservices.NewReadReceiptService(readReceiptRepo, messageRepo)

	// 💬 Chat: handler
	chatHandler := handlers.NewChatHandler(chatService, attachmentService, reactionService, readReceiptService)

	// Subscription
	subscriptionRepo := repositories.NewSubscriptionRepository(sqlDB)
	subscriptionService := services.NewSubscriptionService(subscriptionRepo)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)

	// Upload
	uploadRepo := repositories.NewUploadRepository(sqlDB)
	uploadService := services.NewUploadService(uploadRepo, "/mwork-front-fn/uploads", "/mwork-front-fn/uploads")
	uploadHandler := handlers.NewUploadHandler(uploadService)

	// Analytics
	analyticsRepo := repositories.NewAnalyticsRepository(sqlDB)
	analyticsService := services.NewAnalyticsService(analyticsRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService, modelProfileRepo)

	// Инициализируем Gin
	router := gin.Default()

	// Регистрируем маршруты
	routes.SetupRoutes(
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

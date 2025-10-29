package database

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"mwork_backend/internal/config"
	"mwork_backend/internal/models"
	chatmodels "mwork_backend/internal/models/chat"
	// "strings" // Больше не нужен
)

var gormDB *gorm.DB

// ConnectGorm инициализирует GORM с URL из config.yaml
func ConnectGorm() (*gorm.DB, error) {
	if gormDB != nil {
		return gormDB, nil
	}

	cfg := config.GetConfig()
	dsn := cfg.Database.DSN

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GORM: %w", err)
	}

	gormDB = db
	return db, nil
}

// AutoMigrate выполняет миграцию всех моделей
func AutoMigrate() error {
	db, err := ConnectGorm()
	if err != nil {
		return err
	}

	// 🔻🔻🔻 ЭТОТ БЛОК УДАЛЕН 🔻🔻🔻
	// // Попытка удалить constraint, если он есть
	// if err := db.Migrator().DropConstraint(&models.RefreshToken{}, "uni_refresh_tokens_token"); err != nil {
	//    if !strings.Contains(err.Error(), "не существует") {
	//       log.Fatalf("Ошибка DropConstraint: %v", err)
	//    }
	// }
	// 🔺🔺🔺 БЛОК УДАЛЕН 🔺🔺🔺

	// Миграция
	err = db.AutoMigrate(
		&models.User{},
		&models.ModelProfile{},
		&models.EmployerProfile{},
		&models.Casting{},
		&models.Upload{},
		&models.SubscriptionPlan{},
		&models.RefreshToken{},
		&models.UserSubscription{},
		&models.CastingResponse{},
		// chat модуль
		&chatmodels.Dialog{},
		&chatmodels.DialogParticipant{},
		&chatmodels.Message{},
		&chatmodels.MessageAttachment{},
		&chatmodels.MessageReaction{},
		&chatmodels.MessageReadReceipt{},
	)

	if err != nil {
		log.Fatalf("❌ AutoMigrate ошибка: %v", err)
	}

	// Добавим лог, чтобы видеть, что все прошло успешно
	log.Println("✅ AutoMigrate успешно завершен.")
	return nil
}

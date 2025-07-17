package database

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mwork_front_fn/backend/config"
	"mwork_front_fn/backend/models"
	chatmodels "mwork_front_fn/backend/models/chat"
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

	// Попытка удалить constraint, если он есть
	if err := db.Migrator().DropConstraint(&models.RefreshToken{}, "uni_refresh_tokens_token"); err != nil {
		if !strings.Contains(err.Error(), "не существует") {
			log.Fatalf("Ошибка DropConstraint: %v", err)
		}
	}

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
		&models.Rating{},
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

	return nil
}

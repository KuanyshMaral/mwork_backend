package integration_test

import (
	"log"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"os"
	"sync"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Глобальные переменные для общего состояния
var (
	globalTestServer *helpers.TestServer
	serverOnce       sync.Once
)

// createFreePlan создает бесплатный план подписки
func createFreePlan(t *testing.T, db *gorm.DB) {
	var count int64
	db.Model(&models.SubscriptionPlan{}).Where("name = ?", "Free").Count(&count)
	if count > 0 {
		return
	}

	limitsJSON := datatypes.JSON(`{"publications": 5, "responses": 10}`)
	freePlan := models.SubscriptionPlan{
		Name:     "Free",
		Price:    0,
		Currency: "KZT",
		Duration: "unlimited",
		Limits:   limitsJSON,
		IsActive: true,
	}
	if err := db.Create(&freePlan).Error; err != nil {
		t.Fatalf("Failed to create free subscription plan: %v", err)
	}
}

// GetTestServer возвращает тестовый сервер (создает при первом вызове)
func GetTestServer(t *testing.T) *helpers.TestServer {
	serverOnce.Do(func() {
		// Устанавливаем тестовые environment variables
		os.Setenv("SERVER_PORT", "4001")
		os.Setenv("SERVER_ENV", "test")
		os.Setenv("DATABASE_URL", "postgres://postgres:Sagster-2020@localhost:5432/mwork_test?sslmode=disable")
		os.Setenv("JWT_SECRET", "my_super_secret_key_for_tests_12345")
		os.Setenv("TEMPLATES_DIR", "internal/email/templates")

		log.Println("--- [GetTestServer] Initializing test server... ---")
		globalTestServer = helpers.NewTestServer(t)

		// Создаем Free план один раз при инициализации
		createFreePlan(t, globalTestServer.DB)
		log.Println("--- [GetTestServer] Test server ready ---")
	})
	return globalTestServer
}

// TestMain теперь только для глобальной инициализации
func TestMain(m *testing.M) {
	code := m.Run()

	// Очистка после ВСЕХ тестов
	if globalTestServer != nil {
		log.Println("--- [TestMain] Cleaning up... ---")
		globalTestServer.Close()
	}

	os.Exit(code)
}

// CreateTestCasting в транзакции
func CreateTestCasting(t *testing.T, tx *gorm.DB, employerID string, title string, city string) models.Casting {
	casting := models.Casting{
		EmployerID:  employerID,
		Title:       title,
		City:        city,
		Description: "Test description",
		Status:      models.CastingStatusActive,
	}
	if err := tx.Create(&casting).Error; err != nil {
		t.Fatalf("Failed to create test casting: %v", err)
	}
	return casting
}

// CreateTestReview в транзакции
func CreateTestReview(t *testing.T, tx *gorm.DB, employerID, modelID string, castingID *string, rating int, text string) models.Review {
	review := models.Review{
		EmployerID: employerID,
		ModelID:    modelID,
		CastingID:  castingID,
		Rating:     rating,
		ReviewText: text,
		Status:     models.ReviewStatusApproved,
	}
	if err := tx.Create(&review).Error; err != nil {
		t.Fatalf("Failed to create test review: %v", err)
	}
	return review
}

// CreateTestResponse в транзакции
func CreateTestResponse(t *testing.T, tx *gorm.DB, castingID, modelID string, status models.ResponseStatus) models.CastingResponse {
	response := models.CastingResponse{
		CastingID: castingID,
		ModelID:   modelID,
		Status:    status,
	}
	if err := tx.Create(&response).Error; err != nil {
		t.Fatalf("Failed to create test response: %v", err)
	}
	return response
}

package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"mwork_backend/internal/models"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser создает пользователя в транзакции с автоматическим хешированием пароля
func CreateUser(t *testing.T, db *gorm.DB, user *models.User) error {
	// ✅ Проверяем, нужно ли хешировать пароль
	if user.PasswordHash != "" && !strings.HasPrefix(user.PasswordHash, "$2a$") {
		rawPassword := user.PasswordHash
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("Не удалось хешировать пароль: %v", err)
		}
		user.PasswordHash = string(hashedPassword)

		// Сохраняем сырой пароль в поле для последующего использования
		// (это нужно только для тестов!)
		user.ResetToken = rawPassword // Временно используем это поле
	}

	// ✅ По умолчанию - активный и верифицированный
	if user.Status == "" {
		user.Status = models.UserStatusActive
	}
	user.IsVerified = true

	result := db.Create(user)
	if result.Error != nil {
		t.Logf("ОШИБКА: не удалось создать пользователя %s: %v", user.Email, result.Error)
		return result.Error
	}

	return nil
}

// CreateAndLoginUser создает пользователя и логинит его
func CreateAndLoginUser(t *testing.T, ts *TestServer, tx *gorm.DB, name, email, password string, role models.UserRole) (string, *models.User) {
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: password, // Сырой пароль
		Role:         role,
	}
	err := CreateUser(t, tx, user)
	assert.NoError(t, err, "Создание тестового пользователя не должно вызывать ошибку")

	// ✅ Логиним через API с сырым паролем
	loginBody := map[string]interface{}{
		"email":    email,
		"password": password, // Используем сырой пароль
	}

	res, bodyStr := ts.SendRequest(t, http.MethodPost, "/api/v1/auth/login", "", loginBody)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Логин должен быть успешным. Ответ: "+bodyStr)

	var loginResponse struct {
		Token string `json:"access_token"`
	}
	err = json.Unmarshal([]byte(bodyStr), &loginResponse)
	assert.NoError(t, err, "Не удалось распарсить JSON")
	assert.NotEmpty(t, loginResponse.Token, "Токен не должен быть пустым")

	log.Printf("✅ [Helper] Создан и залогинен пользователь %s (Role: %s)", email, role)

	// ✅ Восстанавливаем сырой пароль в объекте user (для удобства в тестах)
	user.PasswordHash = password

	return loginResponse.Token, user
}

// CreateAndLoginEmployer создает работодателя с уникальным email
func CreateAndLoginEmployer(t *testing.T, ts *TestServer, tx *gorm.DB) (string, *models.User, *models.EmployerProfile) {
	email := fmt.Sprintf("employer_%d@test.com", time.Now().UnixNano())
	token, user := CreateAndLoginUser(t, ts, tx, "Test Employer", email, "password123", models.UserRoleEmployer)

	profile := &models.EmployerProfile{
		UserID:      user.ID,
		CompanyName: "Test Company Inc.",
		City:        "Almaty",
		IsVerified:  true,
	}
	result := tx.Create(profile)
	assert.NoError(t, result.Error, "Не удалось создать профиль работодателя")

	log.Printf("✅ [Helper] Создан профиль работодателя для %s", email)
	return token, user, profile
}

// CreateAndLoginModel создает модель с уникальным email
func CreateAndLoginModel(t *testing.T, ts *TestServer, tx *gorm.DB) (string, *models.User, *models.ModelProfile) {
	email := fmt.Sprintf("model_%d@test.com", time.Now().UnixNano())
	token, user := CreateAndLoginUser(t, ts, tx, "Test Model", email, "password123", models.UserRoleModel)

	profile := &models.ModelProfile{
		UserID:   user.ID,
		Name:     "Test Model",
		Age:      25,
		Height:   175,
		Weight:   55,
		Gender:   "female",
		City:     "Almaty",
		IsPublic: true,
	}
	result := tx.Create(profile)
	assert.NoError(t, result.Error, "Не удалось создать профиль модели")

	log.Printf("✅ [Helper] Создан профиль модели для %s", email)
	return token, user, profile
}

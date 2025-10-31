package helpers

import (
	"encoding/json"
	"log"
	"mwork_backend/internal/models"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser в транзакции
func CreateUser(t *testing.T, db *gorm.DB, user *models.User) error {
	rawPassword := user.PasswordHash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Не удалось хешировать пароль для тестового пользователя: %v", err)
	}
	user.PasswordHash = string(hashedPassword)

	user.Status = models.UserStatusActive
	user.IsVerified = true

	result := db.Create(user)
	if result.Error != nil {
		t.Logf("ОШИБКА: не удалось создать пользователя %s: %v", user.Email, result.Error)
		return result.Error
	}

	user.PasswordHash = rawPassword
	return nil
}

// CreateAndLoginUser в транзакции
func CreateAndLoginUser(t *testing.T, ts *TestServer, tx *gorm.DB, name, email, password string, role models.UserRole) (string, *models.User) {
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: password,
		Role:         role,
	}
	err := CreateUser(t, tx, user)
	assert.NoError(t, err, "Создание тестового пользователя не должно вызывать ошибку")

	loginBody := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	res, bodyStr := ts.SendRequest(t, http.MethodPost, "/api/v1/auth/login", "", loginBody)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Логин тестового пользователя должен быть успешным. Ответ: "+bodyStr)

	var loginResponse struct {
		Token string `json:"access_token"`
	}
	err = json.Unmarshal([]byte(bodyStr), &loginResponse)
	assert.NoError(t, err, "Не удалось распарсить JSON ответа /login")
	assert.NotEmpty(t, loginResponse.Token, "Токен не должен быть пустым")

	log.Printf("✅ [Helper] Создан и залогинен пользователь %s (Role: %s)", email, role)
	return loginResponse.Token, user
}

// CreateAndLoginEmployer в транзакции
func CreateAndLoginEmployer(t *testing.T, ts *TestServer, tx *gorm.DB) (string, *models.User, *models.EmployerProfile) {
	email := "employer@test.com"
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

// CreateAndLoginModel в транзакции
func CreateAndLoginModel(t *testing.T, ts *TestServer, tx *gorm.DB) (string, *models.User, *models.ModelProfile) {
	email := "model@test.com"
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

package integration_test

import (
	"fmt"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TestAuthFlow - проверяет регистрацию и ОЖИДАЕМЫЙ провал логина
func TestAuthFlow(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// ✅ Уникальный email для этого теста
	email := fmt.Sprintf("authflow_%d@test.com", time.Now().UnixNano())

	registerBody := map[string]interface{}{
		"name":     "Тестовая Модель",
		"email":    email,
		"password": "super_password123",
		"role":     "model",
		"city":     "Almaty",
	}

	regRes, regBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/register", "", registerBody)

	assert.Equal(t, http.StatusCreated, regRes.StatusCode)
	assert.Contains(t, regBodyStr, "Registration successful")
	t.Logf("РЕГИСТРАЦИЯ: Успешно. Ответ: %s", regBodyStr)

	// Попытка логина без верификации
	loginBody := map[string]interface{}{
		"email":    email,
		"password": "super_password123",
	}
	logRes, logBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	assert.Equal(t, http.StatusForbidden, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "User not verified")
	t.Logf("ЛОГИН (НЕВЕРИФ.): Успешно провалился (403). Ответ: %s", logBodyStr)
}

// TestGetProfile_Success - проверяет получение профиля
func TestGetProfile_Success(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// ✅ Хелпер создаст уникальный email
	userToken, user, _ := helpers.CreateAndLoginModel(t, ts, tx)

	profRes, profBodyStr := ts.SendRequest(t, "GET", "/api/v1/profile", userToken, nil)

	assert.Equal(t, http.StatusOK, profRes.StatusCode)
	assert.Contains(t, profBodyStr, user.Email)
	assert.Contains(t, profBodyStr, user.Name)
	t.Logf("ПРОФИЛЬ: Успешно. Ответ: %s", profBodyStr)
}

// TestRegister_DuplicateEmail - проверяет защиту от дубликатов
func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// ✅ Уникальный email для этого теста
	email := fmt.Sprintf("duplicate_%d@test.com", time.Now().UnixNano())

	err := helpers.CreateUser(t, tx, &models.User{
		Name:         "User One",
		Email:        email,
		PasswordHash: "pass123",
		Role:         models.UserRoleModel,
	})
	assert.NoError(t, err)

	// Попытка зарегистрировать того же email
	duplicateBody := map[string]interface{}{
		"name":         "User Two",
		"email":        email, // ✅ Тот же email
		"password":     "password_is_long_enough_123",
		"role":         "employer",
		"city":         "Astana",
		"company_name": "Test Company",
	}
	regRes, regBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/register", "", duplicateBody)

	assert.Equal(t, http.StatusConflict, regRes.StatusCode)
	assert.Contains(t, regBodyStr, "Email already exists")
	t.Logf("ДУБЛИКАТ EMAIL: Успешно. Ответ: %s", regBodyStr)
}

// TestLogin_BadPassword - проверяет неверный пароль
func TestLogin_BadPassword(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// ✅ Уникальный email
	email := fmt.Sprintf("badpass_%d@test.com", time.Now().UnixNano())

	err := helpers.CreateUser(t, tx, &models.User{
		Name:         "Test User",
		Email:        email,
		PasswordHash: "correct-password", // Хелпер сам захеширует
		Role:         models.UserRoleModel,
	})
	assert.NoError(t, err)

	// Логин с неверным паролем
	loginBody := map[string]interface{}{
		"email":    email,
		"password": "WRONG-password",
	}
	logRes, logBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	assert.Equal(t, http.StatusUnauthorized, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "Invalid email or password")
	t.Logf("НЕВЕРНЫЙ ПАРОЛЬ: Успешно. Ответ: %s", logBodyStr)
}

// TestLogin_Success - проверяет успешный логин
func TestLogin_Success(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// ✅ Уникальный email
	email := fmt.Sprintf("success_%d@test.com", time.Now().UnixNano())
	password := "correct-password"

	// ✅ Вручную хешируем пароль для ясности
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Name:         "Test User",
		Email:        email,
		PasswordHash: string(hashedPassword), // ✅ Хеш
		Role:         models.UserRoleModel,
		IsVerified:   true, // ✅ Обязательно
		Status:       models.UserStatusActive,
	}

	result := tx.Create(user)
	assert.NoError(t, result.Error)

	// Логин с правильным паролем
	loginBody := map[string]interface{}{
		"email":    email,
		"password": password, // ✅ Сырой пароль
	}
	logRes, logBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	assert.Equal(t, http.StatusOK, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "access_token")
	t.Logf("УСПЕШНЫЙ ЛОГИН: Ответ: %s", logBodyStr)
}

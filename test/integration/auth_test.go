package integration_test

import (
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAuthFlow - проверяет регистрацию и ОЖИДАЕМЫЙ провал логина
func TestAuthFlow(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка (Arrange)
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Данные для регистрации
	registerBody := map[string]interface{}{
		"name":     "Тестовая Модель",
		"email":    "model@test.com",
		"password": "super_password123",
		"role":     "model",
		"city":     "Almaty",
	}

	// 2. Действие: Регистрация (Act)
	regRes, regBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/register", "", registerBody)

	// 3. Проверка: Регистрация (Assert)
	assert.Equal(t, http.StatusCreated, regRes.StatusCode)
	assert.Contains(t, regBodyStr, "Registration successful")
	t.Logf("РЕГИСТРАЦИЯ: Успешно. Ответ: %s", regBodyStr)

	// --- Шаг 2: Логин ---
	loginBody := map[string]interface{}{
		"email":    "model@test.com",
		"password": "super_password123",
	}
	logRes, logBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	// 3. Проверка: Логин (Assert)
	assert.Equal(t, http.StatusForbidden, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "User not verified")
	t.Logf("ЛОГИН (НЕВЕРИФ.): Успешно провалился (403). Ответ: %s", logBodyStr)
}

// TestGetProfile_Success - проверяет "золотой путь" с помощью хелпера
func TestGetProfile_Success(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	userToken, user, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Действие: Получение профиля (Act)
	profRes, profBodyStr := ts.SendRequest(t, "GET", "/api/v1/profile", userToken, nil)

	// 3. Проверка: Получение профиля (Assert)
	assert.Equal(t, http.StatusOK, profRes.StatusCode)
	assert.Contains(t, profBodyStr, user.Email)
	assert.Contains(t, profBodyStr, user.Name)
	t.Logf("ПРОФИЛЬ: Успешно. Ответ: %s", profBodyStr)
}

// TestRegister_DuplicateEmail - проверяет защиту от дубликатов
func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Используем хелпер, чтобы НАПРЯМУЮ создать юзера в транзакции
	err := helpers.CreateUser(t, tx, &models.User{
		Name:         "User One",
		Email:        "duplicate@test.com",
		PasswordHash: "pass123",
		Role:         models.UserRoleModel,
	})
	assert.NoError(t, err)

	// 2. Действие: Попытка регистрации с тем же email
	duplicateBody := map[string]interface{}{
		"name":         "User Two",
		"email":        "duplicate@test.com",
		"password":     "password_is_long_enough_123",
		"role":         "employer",
		"city":         "Astana",
		"company_name": "Test Company",
	}
	regRes, regBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/register", "", duplicateBody)

	// 3. Проверка
	assert.Equal(t, http.StatusConflict, regRes.StatusCode)
	assert.Contains(t, regBodyStr, "Email already exists")
	t.Logf("ДУБЛИКАКА EMAIL: Успешно. Ответ: %s", regBodyStr)
}

// TestLogin_BadPassword - проверяет неверный пароль
func TestLogin_BadPassword(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	err := helpers.CreateUser(t, tx, &models.User{
		Name:         "Test User",
		Email:        "user@test.com",
		PasswordHash: "correct-password",
		Role:         models.UserRoleModel,
	})
	assert.NoError(t, err)

	// 2. Действие: Логин с неверным паролем
	loginBody := map[string]interface{}{
		"email":    "user@test.com",
		"password": "WRONG-password",
	}
	logRes, logBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	// 3. Проверка
	assert.Equal(t, http.StatusUnauthorized, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "Invalid email or password")
	t.Logf("НЕВЕРНЫЙ ПАРОЛЬ: Успешно. Ответ: %s", logBodyStr)
}

// TestLogin_Success - проверяет успешный логин
func TestLogin_Success(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователя напрямую в транзакции
	user := &models.User{
		Name:         "Test User",
		Email:        "success@test.com",
		PasswordHash: "correct-password", // Сырой пароль
		Role:         models.UserRoleModel,
	}
	err := helpers.CreateUser(t, tx, user)
	assert.NoError(t, err)

	// 2. Действие: Логин с правильным паролем
	loginBody := map[string]interface{}{
		"email":    "success@test.com",
		"password": "correct-password", // Используем сырой пароль
	}
	logRes, logBodyStr := ts.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	// 3. Проверка
	assert.Equal(t, http.StatusOK, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "access_token")
	t.Logf("УСПЕШНЫЙ ЛОГИН: Ответ: %s", logBodyStr)
}

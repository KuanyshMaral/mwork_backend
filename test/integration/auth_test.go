// Имя пакета _test (с суффиксом) делает его "black-box" тестом.
// Он не имеет доступа к приватным функциям твоего API,
// а тестирует его "снаружи", как Postman.
package integration_test

import (
	"encoding/json"
	"mwork_backend/internal/models" // 👈 Добавили импорт
	"mwork_backend/test/helpers"    // 👈 ИМПОРТ НАШИХ ХЕЛПЕРОВ
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testServer *helpers.TestServer

// TestMain - это главный "хаб". Он запускается ОДИН РАЗ
// для всех тестов в этом файле.
func TestMain(m *testing.M) {
	// 1. Создаем сервер (он внутри создает БД, мигрирует и запускает Gin)
	testServer = helpers.NewTestServer(&testing.T{})

	// 2. Запускаем ВСЕ тесты (m.Run())
	code := m.Run()

	// 3. Останавливаем сервер и чистим за собой
	testServer.Close()
	os.Exit(code)
}

// TestAuthFlow - это наш E2E сценарий "золотого пути".
// Мы НЕ используем хелперы, потому что мы тестируем
// сами эндпоинты /register и /login.
func TestAuthFlow(t *testing.T) {
	// 1. Подготовка (Arrange)
	// Очищаем БД ПЕРЕД тестом для 100% изоляции
	testServer.ClearTables()

	// Данные для регистрации
	registerBody := map[string]interface{}{
		"name":     "Тестовая Модель",
		"email":    "model@test.com",
		"password": "super_password123",
		"role":     "model",
	}

	// 2. Действие: Регистрация (Act)
	regRes, regBodyStr := testServer.SendRequest(t, "POST", "/api/v1/auth/register", "", registerBody)

	// 3. Проверка: Регистрация (Assert)
	assert.Equal(t, http.StatusCreated, regRes.StatusCode)
	assert.Contains(t, regBodyStr, "model@test.com")
	t.Logf("РЕГИСТРАЦИЯ: Успешно. Ответ: %s", regBodyStr)

	// --- Шаг 2: Логин ---

	// 1. Подготовка (Arrange)
	loginBody := map[string]interface{}{
		"email":    "model@test.com",
		"password": "super_password123",
	}

	// 2. Действие: Логин (Act)
	logRes, logBodyStr := testServer.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	// 3. Проверка: Логин (Assert)
	assert.Equal(t, http.StatusOK, logRes.StatusCode)

	var loginResponse struct {
		Token string `json:"token"`
	}
	err := json.Unmarshal([]byte(logBodyStr), &loginResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResponse.Token, "Токен не должен быть пустым")
	t.Logf("ЛОГИН: Успешно. Получен токен.")

	userToken := loginResponse.Token

	// --- Шаг 3: Доступ к защищенному роуту ---

	// 2. Действие: Получение профиля (Act)
	profRes, profBodyStr := testServer.SendRequest(t, "GET", "/api/v1/profile", userToken, nil)

	// 3. Проверка: Получение профиля (Assert)
	assert.Equal(t, http.StatusOK, profRes.StatusCode)
	assert.Contains(t, profBodyStr, "model@test.com")
	assert.Contains(t, profBodyStr, "Тестовая Модель")
	t.Logf("ПРОФИЛЬ: Успешно. Ответ: %s", profBodyStr)
}

// TestRegister_DuplicateEmail - (ПЕРЕПИСАН)
// Здесь мы используем хелпер CreateUser, чтобы БЫСТРО
// создать юзера в БД и проверить защиту от дубликатов.
func TestRegister_DuplicateEmail(t *testing.T) {
	// 1. Подготовка
	testServer.ClearTables()

	// Используем хелпер, чтобы НАПРЯМУЮ создать юзера в БД
	err := helpers.CreateUser(t, testServer.DB, &models.User{
		Name:         "User One",
		Email:        "duplicate@test.com",
		PasswordHash: "pass123", // Хелпер сам хеширует
		Role:         models.UserRoleModel,
	})
	assert.NoError(t, err)

	// 2. Действие: Попытка регистрации с тем же email
	duplicateBody := map[string]interface{}{
		"name": "User Two", "email": "duplicate@test.com", "password": "pass456", "role": "employer",
	}
	regRes, regBodyStr := testServer.SendRequest(t, "POST", "/api/v1/auth/register", "", duplicateBody)

	// 3. Проверка
	assert.Equal(t, http.StatusBadRequest, regRes.StatusCode)
	// (в твоем логе было "email already in use", если нет - поменяй на свою ошибку)
	assert.Contains(t, regBodyStr, "email already in use")
	t.Logf("ДУБЛИКАТ EMAIL: Успешно. Ответ: %s", regBodyStr)
}

// TestLogin_BadPassword - (НОВЫЙ ТЕСТ)
// Проверяем, что нельзя залогиниться с неверным паролем
func TestLogin_BadPassword(t *testing.T) {
	// 1. Подготовка
	testServer.ClearTables()

	// Быстро создаем юзера в БД
	err := helpers.CreateUser(t, testServer.DB, &models.User{
		Name:         "Test User",
		Email:        "user@test.com",
		PasswordHash: "correct-password", // Хелпер хеширует
		Role:         models.UserRoleModel,
	})
	assert.NoError(t, err)

	// 2. Действие: Логин с неверным паролем
	loginBody := map[string]interface{}{
		"email":    "user@test.com",
		"password": "WRONG-password",
	}
	logRes, logBodyStr := testServer.SendRequest(t, "POST", "/api/v1/auth/login", "", loginBody)

	// 3. Проверка
	assert.Equal(t, http.StatusUnauthorized, logRes.StatusCode)
	assert.Contains(t, logBodyStr, "invalid credentials") // или "invalid email or password"
	t.Logf("НЕВЕРНЫЙ ПАРОЛЬ: Успешно. Ответ: %s", logBodyStr)
}

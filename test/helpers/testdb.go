package helpers

import (
	"encoding/json"
	"log"
	"mwork_backend/internal/models" // 👈 Убедись, что импорт моделей верный
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser - это низкоуровневый хелпер.
// Он создает пользователя НАПРЯМУЮ в БД, минуя API.
// Это быстрее и позволяет нам сразу сделать юзера активным.
func CreateUser(t *testing.T, db *gorm.DB, user *models.User) error {
	// 1. Хешируем пароль перед сохранением
	// Мы сохраняем оригинальный пароль (из user.PasswordHash), чтобы использовать его для логина
	rawPassword := user.PasswordHash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Не удалось хешировать пароль для тестового пользователя: %v", err)
	}
	user.PasswordHash = string(hashedPassword)

	// 2. ВАЖНО: Для тестов мы сразу делаем юзера активным и верифицированным,
	// чтобы не проходить флоу верификации по email.
	user.Status = models.UserStatusActive
	user.IsVerified = true

	// 3. Создаем пользователя
	result := db.Create(user)
	if result.Error != nil {
		t.Logf("ОШИБКА: не удалось создать пользователя %s: %v", user.Email, result.Error)
		return result.Error
	}

	// Возвращаем оригинальный пароль в поле, чтобы CreateAndLoginUser мог его использовать
	user.PasswordHash = rawPassword
	return nil
}

// CreateAndLoginUser - это высокоуровневый хелпер.
// Он создает пользователя И сразу логинится им через API,
// возвращая готовый accessToken.
// Это будет самый частый хелпер для 90% тестов.
func CreateAndLoginUser(t *testing.T, ts *TestServer, name, email, password string, role models.UserRole) (string, *models.User) {
	// 1. Создаем пользователя НАПРЯМУЮ в БД
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: password, // Временно храним "сырой" пароль здесь. CreateUser хеширует его.
		Role:         role,
	}
	err := CreateUser(t, ts.DB, user)
	assert.NoError(t, err, "Создание тестового пользователя не должно вызывать ошибку")

	// 2. Логинимся этим пользователем через API
	loginBody := map[string]interface{}{
		"email":    email,
		"password": password, // Используем "сырой" пароль для логина
	}

	res, bodyStr := ts.SendRequest(t, http.MethodPost, "/api/v1/auth/login", "", loginBody)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Логин тестового пользователя должен быть успешным")

	// 3. Парсим токен
	var loginResponse struct {
		Token string `json:"token"` // Убедись, что ключ "token" (или "access_token") верный
	}
	err = json.Unmarshal([]byte(bodyStr), &loginResponse)
	assert.NoError(t, err, "Не удалось распарсить JSON ответа /login")
	assert.NotEmpty(t, loginResponse.Token, "Токен не должен быть пустым")

	// 4. Возвращаем токен и созданного пользователя (на случай, если нужен его ID)
	log.Printf("✅ [Helper] Создан и залогинен пользователь %s (Role: %s)", email, role)
	return loginResponse.Token, user
}

// CreateAndLoginEmployer - хелпер-обертка для создания работодателя
// Сразу создает User + EmployerProfile
func CreateAndLoginEmployer(t *testing.T, ts *TestServer) (string, *models.User, *models.EmployerProfile) {
	email := "employer@test.com"
	// 1. Создаем юзера-работодателя
	token, user := CreateAndLoginUser(t, ts, "Test Employer", email, "password123", models.UserRoleEmployer)

	// 2. Создаем ему профиль НАПРЯМУЮ в БД
	profile := &models.EmployerProfile{
		UserID:      user.ID,
		CompanyName: "Test Company Inc.",
		City:        "Almaty",
		IsVerified:  true, // Сразу верифицирован
	}
	result := ts.DB.Create(profile)
	assert.NoError(t, result.Error, "Не удалось создать профиль работодателя")

	log.Printf("✅ [Helper] Создан профиль работодателя для %s", email)
	return token, user, profile
}

// CreateAndLoginModel - хелпер-обертка для создания модели
// Сразу создает User + ModelProfile
func CreateAndLoginModel(t *testing.T, ts *TestServer) (string, *models.User, *models.ModelProfile) {
	email := "model@test.com"
	// 1. Создаем юзера-модель
	token, user := CreateAndLoginUser(t, ts, "Test Model", email, "password123", models.UserRoleModel)

	// 2. Создаем ей профиль НАПРЯМУЮ в БД
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
	result := ts.DB.Create(profile)
	assert.NoError(t, result.Error, "Не удалось создать профиль модели")

	log.Printf("✅ [Helper] Создан профиль модели для %s", email)
	return token, user, profile
}

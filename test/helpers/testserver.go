package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"log"
	"mwork_backend/internal/config"
	"mwork_backend/internal/models"
	chatmodels "mwork_backend/internal/models/chat"
	"mwork_backend/internal/routes" // 👈 ИМПОРТ ТВОЕГО РОУТЕРА
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServer хранит экземпляры тестового сервера и БД
type TestServer struct {
	Server *httptest.Server
	DB     *gorm.DB
}

// NewTestServer создает и настраивает тестовый сервер и БД
func NewTestServer(t *testing.T) *TestServer {
	// 1. Загружаем конфиг
	cfg := config.GetConfig()
	dsn := cfg.Database.DSN

	// 2. ВАЖНО: Подменяем имя БД на тестовое, чтобы не убить рабочую
	// (например, "mwork" -> "mwork_test")
	// Убедись, что БД "mwork_test" СУЩЕСТВУЕТ в твоем Postgres
	testDSN := strings.Replace(dsn, "mwork", "mwork_test", 1)
	if testDSN == dsn {
		t.Fatalf("Не удалось заменить имя БД на тестовое. Проверь DSN в config.yaml")
	}

	// 3. Подключаемся к ТЕСТОВОЙ БД
	db, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("Не удалось подключиться к тестовой БД: %v", err)
	}

	// 4. (!!!) AutoMigrate - это ОК для тестов
	// Он создает чистую схему в mwork_test
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
		// ... и все остальные твои модели ...
		&chatmodels.Dialog{},
		&chatmodels.DialogParticipant{},
		&chatmodels.Message{},
		&chatmodels.MessageAttachment{},
		&chatmodels.MessageReaction{},
		&chatmodels.MessageReadReceipt{},
	)
	if err != nil {
		t.Fatalf("Не удалось выполнить AutoMigrate для тестовой БД: %v", err)
	}

	// 5. Настраиваем Gin-роутер
	// Здесь мы передаем *тестовую* БД в наш роутер
	router := routes.SetupRouter(db) // 👈 Убедись, что эта функция у тебя есть

	// 6. Запускаем тестовый сервер httptest
	server := httptest.NewServer(router)

	log.Println("✅ Тестовый сервер запущен, тестовая БД настроена.")

	return &TestServer{
		Server: server,
		DB:     db,
	}
}

// Close останавливает сервер и закрывает соединение с БД
func (ts *TestServer) Close() {
	ts.Server.Close()
	sqlDB, _ := ts.DB.DB()
	sqlDB.Close()
}

// ClearTables очищает все таблицы. Вызывается ПЕРЕД каждым тестом.
func (ts *TestServer) ClearTables() {
	log.Println("--- ОЧИСТКА ТАБЛИЦ ---")
	// TRUNCATE намного быстрее, чем DELETE
	// CASCADE удаляет все зависимые данные
	tables := []string{
		"users",
		"model_profiles",
		"employer_profiles",
		"castings",
		"casting_responses",
		"refresh_tokens",
		"user_subscriptions",
		// ... и т.д.
	}

	// Очищаем таблицы в обратном порядке из-за foreign keys
	// *Простой способ:* просто использовать CASCADE
	err := ts.DB.Exec("TRUNCATE TABLE users, model_profiles, employer_profiles, castings, casting_responses, refresh_tokens, user_subscriptions, subscription_plans, uploads, portfolio_items, reviews, notifications, payment_transactions, usage_tracking, chat.dialogs, chat.messages, chat.dialog_participants, chat.message_attachments, chat.message_reactions, chat.message_read_receipts RESTART IDENTITY CASCADE").Error
	if err != nil {
		log.Fatalf("Не удалось очистить таблицы: %v", err)
	}
}

// SendRequest - это универсальный помощник для отправки запросов
// Он возвращает *http.Response и тело ответа в виде строки
func (ts *TestServer) SendRequest(t *testing.T, method, path, token string, body interface{}) (*http.Response, string) {
	// 1. Формируем URL
	url := ts.Server.URL + path // httptest сам даст нам URL (напр. http://127.0.0.1:54321)

	// 2. Кодируем тело запроса (если оно есть)
	var reqBody io.Reader = nil
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Ошибка кодирования JSON для запроса: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// 3. Создаем запрос
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Ошибка создания HTTP-запроса: %v", err)
	}

	// 4. Устанавливаем заголовки
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 5. Отправляем запрос
	// Мы используем ts.Server.Client(), он не делает реальный сетевой вызов
	res, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("Ошибка отправки HTTP-запроса: %v", err)
	}

	// 6. Читаем тело ответа
	resBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Ошибка чтения тела ответа: %v", err)
	}
	defer res.Body.Close()

	return res, string(resBodyBytes)
}

package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mwork_backend/internal/app" // <-- 2. ИЗМЕНЕН ИМПОРТ (с routes на app)
	"mwork_backend/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestServer (без изменений)
type TestServer struct {
	Server *httptest.Server
	DB     *gorm.DB
}

// NewTestServer создает и настраивает тестовый сервер и БД
// 👇👇👇 ЭТА ФУНКЦИЯ ПОЛНОСТЬЮ ЗАМЕНЕНА 👇👇👇
func NewTestServer(t *testing.T) *TestServer {
	// 1. Загружаем конфиг.
	// Он автоматически берет DATABASE_URL (уже с 'mwork_test') из os.Getenv()
	config.LoadConfig()
	cfg := config.GetConfig()
	dsn := cfg.Database.DSN

	// 2. Подключаемся к ТЕСТОВОЙ БД
	//    Логика замены и проверки (которая вызывала deadlock) УДАЛЕНА.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// Мы добавили dsn в лог, чтобы видеть, куда он подключается
		t.Fatalf("Не удалось подключиться к тестовой БД (%s): %v", dsn, err)
	}

	// 3. AutoMigrate (Твой код миграций без изменений)
	/*err = db.AutoMigrate(
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
	}*/

	// 4. Настраиваем Gin-роутер
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Не удалось получить *sql.DB из GORM: %v", err)
	}
	router := app.SetupRouter(cfg, db, sqlDB) // 👈 ИСПРАВЛЕН ВЫЗОВ

	// 5. Запускаем тестовый сервер httptest
	server := httptest.NewServer(router)

	log.Printf("✅ Тестовый сервер запущен, тестовая БД (%s) настроена.", dsn)

	return &TestServer{
		Server: server,
		DB:     db,
	}
}

// Close (без изменений)
func (ts *TestServer) Close() {
	ts.Server.Close()
	sqlDB, _ := ts.DB.DB()
	sqlDB.Close()
}

// ClearTables очищает все таблицы.
func (ts *TestServer) ClearTables() {
	log.Println("--- ОЧИСТКА ТАБЛИЦ ---")

	// 6. ✅ ИСПРАВЛЕНО: Удалена неиспользуемая переменная 'tables'

	// Очищаем таблицы
	err := ts.DB.Exec("TRUNCATE TABLE users, model_profiles, employer_profiles, castings, casting_responses, refresh_tokens, user_subscriptions, subscription_plans, uploads, portfolio_items, reviews, notifications, payment_transactions, usage_tracking, chat.dialogs, chat.messages, chat.dialog_participants, chat.message_attachments, chat.message_reactions, chat.message_read_receipts RESTART IDENTITY CASCADE").Error
	if err != nil {
		log.Fatalf("Не удалось очистить таблицы: %v", err)
	}
}

// SendRequest (без изменений)
func (ts *TestServer) SendRequest(t *testing.T, method, path, token string, body interface{}) (*http.Response, string) {
	// ... (без изменений) ...
	url := ts.Server.URL + path

	var reqBody io.Reader = nil
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Ошибка кодирования JSON для запроса: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Ошибка создания HTTP-запроса: %v", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("Ошибка отправки HTTP-запроса: %v", err)
	}

	resBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Ошибка чтения тела ответа: %v", err)
	}
	defer res.Body.Close()

	return res, string(resBodyBytes)
}

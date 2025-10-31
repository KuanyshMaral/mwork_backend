package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mwork_backend/internal/app"
	"mwork_backend/internal/config"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestServer с поддержкой транзакций
type TestServer struct {
	Server      *httptest.Server
	DB          *gorm.DB   // Основное подключение (для миграций)
	serverMutex sync.Mutex // Защита от параллельного создания серверов
}

// NewTestServer создает тестовый сервер БЕЗ AutoMigrate
func NewTestServer(t *testing.T) *TestServer {
	config.LoadConfig()
	cfg := config.GetConfig()
	dsn := cfg.Database.DSN

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Не удалось подключиться к тестовой БД (%s): %v", dsn, err)
	}

	// УБИРАЕМ AutoMigrate - используем реальные миграции
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Не удалось получить *sql.DB из GORM: %v", err)
	}

	router := app.SetupRouter(cfg, db, sqlDB)
	server := httptest.NewServer(router)

	log.Printf("✅ Тестовый сервер запущен (транзакционный режим), БД: %s", dsn)

	return &TestServer{
		Server: server,
		DB:     db,
	}
}

// Close закрывает сервер
func (ts *TestServer) Close() {
	ts.Server.Close()
	sqlDB, _ := ts.DB.DB()
	sqlDB.Close()
}

// BeginTransaction начинает транзакцию для теста
func (ts *TestServer) BeginTransaction(t *testing.T) *gorm.DB {
	tx := ts.DB.Begin()
	if tx.Error != nil {
		t.Fatalf("Не удалось начать транзакцию: %v", tx.Error)
	}
	return tx
}

// RollbackTransaction откатывает транзакцию (вызывать в defer)
func (ts *TestServer) RollbackTransaction(t *testing.T, tx *gorm.DB) {
	if r := recover(); r != nil {
		tx.Rollback()
		t.Fatalf("Тест упал с panic: %v", r)
	}
	tx.Rollback()
}

// SendRequest остается без изменений
func (ts *TestServer) SendRequest(t *testing.T, method, path, token string, body interface{}) (*http.Response, string) {
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

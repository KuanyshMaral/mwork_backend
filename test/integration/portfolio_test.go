package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// createDummyFile — локальный хелпер для создания временного файла для тестов загрузки
func createDummyFile(t *testing.T, content string) (*os.File, func()) {
	t.Helper()
	file, err := os.CreateTemp("", "test_upload_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	file.Seek(0, 0) // Возвращаем указатель в начало
	cleanup := func() {
		file.Close()
		os.Remove(file.Name())
	}
	return file, cleanup
}

// CreateTestPortfolioItem — локальный хелпер для создания элемента портфолио через API
// (Использует multipart/form-data)
func CreateTestPortfolioItem(t *testing.T, ts *helpers.TestServer, modelToken string, title string) models.PortfolioItem {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// 1. Добавляем поля DTO (title, description) как form fields
	_ = writer.WriteField("title", title)
	_ = writer.WriteField("description", "Test description for "+title)

	// 2. Создаем и добавляем dummy-файл
	file, cleanup := createDummyFile(t, "dummy image data")
	defer cleanup()

	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	assert.NoError(t, err, "Failed to create form file")
	_, err = io.Copy(part, file)
	assert.NoError(t, err, "Failed to copy file data")

	// 3. Закрываем writer
	err = writer.Close()
	assert.NoError(t, err, "Failed to close multipart writer")

	// 4. Создаем и отправляем запрос
	url := ts.Server.URL + "/api/v1/portfolio"
	req, err := http.NewRequest(http.MethodPost, url, body)
	assert.NoError(t, err, "Failed to create multipart request")

	req.Header.Set("Authorization", "Bearer "+modelToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := ts.Server.Client().Do(req)
	assert.NoError(t, err, "Failed to send multipart request")

	resBodyBytes, _ := io.ReadAll(res.Body)
	res.Body.Close()

	assert.Equal(t, http.StatusCreated, res.StatusCode, "Failed to create portfolio item. Body: "+string(resBodyBytes))

	// 5. Парсим ответ
	var createdItem models.PortfolioItem
	err = json.Unmarshal(resBodyBytes, &createdItem)
	assert.NoError(t, err, "Failed to unmarshal created portfolio item")
	assert.NotEmpty(t, createdItem.ID, "Created item ID is empty")

	return createdItem
}

// TestPortfolioAndUploads — главный тест-сьют для эндпоинтов /portfolio и /uploads
func TestPortfolioAndUploads(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Setup
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем Модель (для тестов портфолио)
	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем Работодателя (для тестов 403 Forbidden)
	empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Создаем Админа (для тестов /admin/uploads)
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx,
		"Portfolio Admin", "portfolio-admin@test.com", "password123", models.UserRoleAdmin)

	// 2. Запускаем sub-тесты

	// --- Тесты /portfolio ---
	var item1, item2 models.PortfolioItem

	t.Run("POST /portfolio - Create Item", func(t *testing.T) {
		// 1. Успешное создание (тестируется через хелпер)
		item1 = CreateTestPortfolioItem(t, ts, modelToken, "Item One")
		item2 = CreateTestPortfolioItem(t, ts, modelToken, "Item Two")
		assert.NotEmpty(t, item1.ID)
		assert.NotEmpty(t, item2.ID)

		// 2. Ошибка: Попытка создания Работодателем (403 Forbidden)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("title", "Forbidden Item")
		file, clean := createDummyFile(t, "forbidden data")
		defer clean()
		part, _ := writer.CreateFormFile("file", "forbidden.txt")
		io.Copy(part, file)
		writer.Close()

		url := ts.Server.URL + "/api/v1/portfolio"
		req, _ := http.NewRequest(http.MethodPost, url, body)
		req.Header.Set("Authorization", "Bearer "+empToken) // Токен Работодателя
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		assert.NoError(t, err)
		res.Body.Close()
		// Ожидаем 403, т.к. сервис должен проверить, что ID юзера принадлежит Модели
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Employer should not be able to create portfolio items")

		// 3. Ошибка: Нет файла (400 Bad Request)
		// (Используем SendRequest, который отправляет JSON, что вызовет ошибку 'File is required')
		res, bodyStr := ts.SendRequest(t, http.MethodPost, "/api/v1/portfolio", modelToken, gin.H{"title": "no file"})
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Contains(t, bodyStr, "File is required", "Error message should state that file is required")
	})

	t.Run("GET /portfolio - Public Retrieval", func(t *testing.T) {
		// 1. GET /:itemId (Успешно)
		res, bodyStr := ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/"+item1.ID, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, item1.Title, "Response should contain item title")
		assert.Contains(t, bodyStr, item1.UploadID, "Response should contain upload ID")

		// 2. GET /:itemId (Не найдено)
		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/non-existent-uuid", "", nil)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)

		// 3. GET /model/:modelId (Успешно)
		res, bodyStr = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/model/"+modelProfile.ID, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"total":2`, "Should find 2 portfolio items for this model")
		assert.Contains(t, bodyStr, item1.ID)
		assert.Contains(t, bodyStr, item2.ID)

		// 4. GET /featured & /recent (Просто проверяем 200 OK)
		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/featured?limit=5", "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/recent?limit=5", "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("PUT /portfolio - Protected Management", func(t *testing.T) {
		// 1. PUT /:itemId (Update)
		updateBody := gin.H{"title": "Updated Title", "description": "Updated Desc"}
		res, bodyStr := ts.SendRequest(t, http.MethodPut, "/api/v1/portfolio/"+item1.ID, modelToken, updateBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Failed to update. Body: "+bodyStr)

		// 2. PUT /:itemId (Ошибка: не владелец)
		res, _ = ts.SendRequest(t, http.MethodPut, "/api/v1/portfolio/"+item1.ID, empToken, updateBody)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Employer should not update other's item")

		// 3. PUT /reorder
		// Меняем порядок item1 и item2
		reorderBody := gin.H{"ordered_ids": []string{item2.ID, item1.ID}}
		res, _ = ts.SendRequest(t, http.MethodPut, "/api/v1/portfolio/reorder", modelToken, reorderBody)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// 4. PUT /:itemId/visibility (Скрыть)
		visBody := gin.H{"is_public": false}
		res, _ = ts.SendRequest(t, http.MethodPut, "/api/v1/portfolio/"+item1.ID+"/visibility", modelToken, visBody)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("GET /portfolio/stats/:modelId", func(t *testing.T) {
		// 1. Успешно (с токеном)
		res, bodyStr := ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/stats/"+modelProfile.ID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Failed to get stats. Body: "+bodyStr)
		assert.Contains(t, bodyStr, "total_items", "Stats response missing keys")
		assert.Contains(t, bodyStr, "total_views", "Stats response missing keys")

		// 2. Ошибка: без токена
		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/stats/"+modelProfile.ID, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	// --- Тесты /uploads ---
	var generalUpload models.Upload

	t.Run("POST /uploads - General File Upload", func(t *testing.T) {
		// Тестируем generic-загрузчик (например, для аватара профиля)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("entityType", "model_profile")
		_ = writer.WriteField("entityId", modelProfile.ID)
		_ = writer.WriteField("usage", "avatar")

		file, clean := createDummyFile(t, "avatar data")
		defer clean()
		part, _ := writer.CreateFormFile("file", "avatar.jpg")
		io.Copy(part, file)
		writer.Close()

		url := ts.Server.URL + "/api/v1/uploads"
		req, _ := http.NewRequest(http.MethodPost, url, body)
		req.Header.Set("Authorization", "Bearer "+modelToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := ts.Server.Client().Do(req)
		assert.NoError(t, err)
		resBodyBytes, _ := io.ReadAll(res.Body)
		res.Body.Close()

		assert.Equal(t, http.StatusCreated, res.StatusCode, "Failed to upload generic file. Body: "+string(resBodyBytes))
		err = json.Unmarshal(resBodyBytes, &generalUpload)
		assert.NoError(t, err)
		assert.NotEmpty(t, generalUpload.ID, "Upload response has no ID")
	})

	t.Run("GET/DELETE /uploads - Protected Upload Management", func(t *testing.T) {
		// 1. GET /:uploadId
		res, bodyStr := ts.SendRequest(t, http.MethodGet, "/api/v1/uploads/"+generalUpload.ID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, generalUpload.ID)
		assert.Contains(t, bodyStr, "avatar") // Проверяем 'usage'

		// 2. GET /user/me
		res, bodyStr = ts.SendRequest(t, http.MethodGet, "/api/v1/uploads/user/me", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		// Должны найти 3 файла: 2 портфолио + 1 аватар
		assert.Contains(t, bodyStr, item1.UploadID)
		assert.Contains(t, bodyStr, item2.UploadID)
		assert.Contains(t, bodyStr, generalUpload.ID)

		// 3. GET /storage/usage
		res, bodyStr = ts.SendRequest(t, http.MethodGet, "/api/v1/uploads/storage/usage", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, "total_usage_bytes")
		assert.Contains(t, bodyStr, "total_files")

		// 4. DELETE /:uploadId
		res, _ = ts.SendRequest(t, http.MethodDelete, "/api/v1/uploads/"+generalUpload.ID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// 5. DELETE /:uploadId (Ошибка: не владелец)
		// (item1.UploadID принадлежит modelUser)
		res, _ = ts.SendRequest(t, http.MethodDelete, "/api/v1/uploads/"+item1.UploadID, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode)
	})

	// --- Тесты /admin/uploads ---
	t.Run("/admin/uploads - Admin Functions", func(t *testing.T) {
		// 1. POST /clean-orphaned (As Admin)
		res, _ := ts.SendRequest(t, http.MethodPost, "/api/v1/admin/uploads/clean-orphaned", adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// 2. POST /clean-orphaned (As Model - 403)
		res, _ = ts.SendRequest(t, http.MethodPost, "/api/v1/admin/uploads/clean-orphaned", modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "RoleMiddleware should block non-admins")

		// 3. GET /stats (As Admin)
		res, bodyStr := ts.SendRequest(t, http.MethodGet, "/api/v1/admin/uploads/stats", adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, "total_files", "Admin stats missing keys")
		assert.Contains(t, bodyStr, "total_storage_gb", "Admin stats missing keys")

		// 4. GET /stats (As Model - 403)
		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/admin/uploads/stats", modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "RoleMiddleware should block non-admins")
	})

	// --- Финальный DELETE (очистка) ---
	t.Run("DELETE /portfolio - Cleanup", func(t *testing.T) {
		// item1 был удален в 'PUT/DELETE /portfolio'
		// Удаляем item2
		res, _ := ts.SendRequest(t, http.MethodDelete, "/api/v1/portfolio/"+item2.ID, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}

// TestPortfolio_Isolated - отдельные изолированные тесты для лучшего параллелизма
func TestPortfolio_CreateAndDelete(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем модель
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем портфолио
	item := CreateTestPortfolioItem(t, ts, modelToken, "Isolated Test Item")
	assert.NotEmpty(t, item.ID)

	// Удаляем портфолио
	res, _ := ts.SendRequest(t, http.MethodDelete, "/api/v1/portfolio/"+item.ID, modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Проверяем, что удалено
	res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/"+item.ID, "", nil)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestPortfolio_Security(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем двух пользователей
	modelToken1, model1, _ := helpers.CreateAndLoginModel(t, ts, tx)
	modelToken2, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Модель 1 создает портфолио
	item := CreateTestPortfolioItem(t, ts, modelToken1, "Security Test Item")

	// Модель 2 пытается изменить портфолио модели 1
	updateBody := gin.H{"title": "Hacked Title"}
	res, _ := ts.SendRequest(t, http.MethodPut, "/api/v1/portfolio/"+item.ID, modelToken2, updateBody)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	// Модель 2 пытается удалить портфолио модели 1
	res, _ = ts.SendRequest(t, http.MethodDelete, "/api/v1/portfolio/"+item.ID, modelToken2, nil)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	// Модель 2 пытается получить приватную статистику модели 1
	res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/portfolio/stats/"+model1.ID, modelToken2, nil)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

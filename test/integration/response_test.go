package integration_test

import (
	"encoding/json"
	"fmt"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResponse_ModelFlow - проверяет E2E "золотой путь" для Модели
func TestResponse_ModelFlow(t *testing.T) {
	t.Parallel()

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	_, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ✅ Fixed: removed unused employerToken
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, employerUser.ID, "Casting for Models", "Almaty")

	// 2. Действие: Модель откликается на кастинг (POST)
	responseBody := map[string]interface{}{
		"message": "Я идеально подхожу для этой роли!",
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/responses/castings/"+casting.ID, modelToken, responseBody)

	// 3. Проверка: Создание
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Я идеально подхожу")
	t.Logf("ОТКЛИК (Model): Создание (201) - Успешно.")

	// 4. Действие: Модель получает свои отклики (GET /my)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/my", modelToken, nil)

	// 5. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Casting for Models")
	assert.Contains(t, bodyStr, `"total":1`)

	// Парсим ID отклика
	var getMyResp struct {
		Responses []models.CastingResponse `json:"responses"`
	}
	err := json.Unmarshal([]byte(bodyStr), &getMyResp)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(getMyResp.Responses), "Должен быть 1 отклик")
	responseID := getMyResp.Responses[0].ID
	t.Logf("ОТКЛИК (Model): GET /my (200) - Успешно, найден 1 отклик.")

	// 6. Действие: Модель читает свой отклик (GET /:responseId)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/"+responseID, modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, responseID)
	t.Logf("ОТКЛИК (Common): Модель читает свой отклик (200) - Успешно.")

	// 7. Действие: Модель удаляет свой отклик (DELETE)
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/responses/"+responseID, modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Response deleted successfully")
	t.Logf("ОТКЛИК (Model): DELETE /:id (200) - Успешно.")

	// 8. Действие: Модель проверяет, что отклик удален (GET /my)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/my", modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":0`)
	t.Logf("ОТКЛИК (Model): GET /my (пусто) (200) - Успешно.")
}

// TestResponse_EmployerFlow - проверяет E2E "золотой путь" для Работодателя
func TestResponse_EmployerFlow(t *testing.T) {
	t.Parallel()

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, employerUser.ID, "Casting for Models", "Almaty")

	// Симулируем отклик Модели через хелпер
	response := CreateTestResponse(t, tx, casting.ID, modelUser.ID, models.ResponseStatusPending)
	responseID := response.ID

	// 2. Действие: Работодатель получает список откликов (GET /list)
	listURL := fmt.Sprintf("/api/v1/responses/castings/%s/list", casting.ID)
	res, bodyStr := ts.SendRequest(t, "GET", listURL, employerToken, nil)

	// 3. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, modelUser.Name)
	assert.Contains(t, bodyStr, `"total":1`)
	t.Logf("ОТКЛИК (Employer): GET /list (200) - Успешно.")

	// 4. Действие: Работодатель одобряет отклик (PUT /status)
	statusURL := fmt.Sprintf("/api/v1/responses/%s/status", responseID)
	statusBody := map[string]interface{}{
		"status": models.ResponseStatusAccepted,
	}
	res, bodyStr = ts.SendRequest(t, "PUT", statusURL, employerToken, statusBody)

	// 5. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Response status updated successfully")
	t.Logf("ОТКЛИК (Employer): PUT /status (200) - Успешно.")

	// 6. Действие: Работодатель читает этот отклик (GET /:responseId)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/"+responseID, employerToken, nil)

	// 7. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"status":"accepted"`)
	t.Logf("ОТКЛИК (Common): Работодатель читает отклик (200) - Успешно.")
}

// TestResponse_Security - проверяет права доступа (401, 403, 404)
func TestResponse_Security(t *testing.T) {
	t.Parallel()

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	modelToken, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, employerUser.ID, "Casting", "Almaty")
	response := CreateTestResponse(t, tx, casting.ID, modelUser.ID, models.ResponseStatusPending)

	// 2. Действие: Работодатель пытается откликнуться (роут Модели)
	res, _ := ts.SendRequest(t, "POST", "/api/v1/responses/castings/"+casting.ID, employerToken, nil)
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Response): Работодатель не может откликнуться (403) - Успешно.")

	// 4. Действие: Модель пытается посмотреть список откликов (роут Работодателя)
	res, _ = ts.SendRequest(t, "GET", "/api/v1/responses/castings/"+casting.ID+"/list", modelToken, nil)
	// 5. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Response): Модель не может читать список (403) - Успешно.")

	// 6. Действие: Модель пытается обновить статус (роут Работодателя)
	statusURL := fmt.Sprintf("/api/v1/responses/%s/status", response.ID)
	res, _ = ts.SendRequest(t, "PUT", statusURL, modelToken, nil)
	// 7. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Response): Модель не может обновить статус (403) - Успешно.")

	// 8. Действие: Аноним пытается получить отклики
	res, _ = ts.SendRequest(t, "GET", "/api/v1/responses/my", "", nil)
	// 9. Проверка: (401 Unauthorized)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Response): Аноним не может читать /my (401) - Успешно.")
}

// TestResponse_StatusWorkflow - проверяет весь workflow статусов отклика
func TestResponse_StatusWorkflow(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx) // ✅ Fixed: removed unused modelUser
	casting := CreateTestCasting(t, tx, employerUser.ID, "Status Workflow Casting", "Almaty")

	// 1. Модель создает отклик (статус: pending)
	responseBody := map[string]interface{}{
		"message": "Хочу участвовать!",
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/responses/castings/"+casting.ID, modelToken, responseBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// Получаем ID созданного отклика
	var getMyResp struct {
		Responses []models.CastingResponse `json:"responses"`
	}
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/my", modelToken, nil)
	err := json.Unmarshal([]byte(bodyStr), &getMyResp)
	assert.NoError(t, err)
	responseID := getMyResp.Responses[0].ID

	// 2. Работодатель проверяет статус (должен быть pending)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/"+responseID, employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"status":"pending"`)

	// 3. Работодатель меняет статус на accepted
	statusBody := map[string]interface{}{"status": models.ResponseStatusAccepted}
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/responses/"+responseID+"/status", employerToken, statusBody)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// 4. Проверяем, что статус обновился
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/"+responseID, modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"status":"accepted"`)

	// 5. Работодатель меняет статус на rejected
	statusBody["status"] = models.ResponseStatusRejected
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/responses/"+responseID+"/status", employerToken, statusBody)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// 6. Проверяем финальный статус
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/"+responseID, employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"status":"rejected"`)
}

// TestResponse_DuplicatePrevention - проверяет защиту от дубликатов
func TestResponse_DuplicatePrevention(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	_, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, employerUser.ID, "Duplicate Test Casting", "Almaty")

	// 1. Первый отклик - успешно
	responseBody := map[string]interface{}{
		"message": "Первый отклик",
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/responses/castings/"+casting.ID, modelToken, responseBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// 2. Второй отклик от той же модели - должен быть конфликт
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/responses/castings/"+casting.ID, modelToken, responseBody)
	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Contains(t, bodyStr, "already responded", "Should prevent duplicate responses")

	// 3. Проверяем, что в базе только один отклик
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/responses/my", modelToken, nil)
	var getMyResp struct {
		Responses []models.CastingResponse `json:"responses"`
		Total     int                      `json:"total"`
	}
	err := json.Unmarshal([]byte(bodyStr), &getMyResp)
	assert.NoError(t, err)
	assert.Equal(t, 1, getMyResp.Total, "Should have only one response despite duplicate attempt")
}

// TestResponse_NotFound - проверяет обработку несуществующих ресурсов
func TestResponse_NotFound(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Пытаемся получить несуществующий отклик
	res, bodyStr := ts.SendRequest(t, "GET", "/api/v1/responses/non-existent-uuid", modelToken, nil)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, bodyStr, "not found", "Should return not found for non-existent response")

	// Пытаемся удалить несуществующий отклик
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/responses/non-existent-uuid", modelToken, nil)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// Пытаемся обновить статус несуществующего отклика
	statusBody := map[string]interface{}{"status": models.ResponseStatusAccepted}
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/responses/non-existent-uuid/status", modelToken, statusBody)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

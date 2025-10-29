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

// TestCastingWorkflow - E2E сценарий для кастингов
func TestCastingWorkflow(t *testing.T) {
	// 1. Подготовка (Arrange)
	testServer.ClearTables()

	// 1.1. Создаем Работодателя
	// Хелпер сразу создает User + EmployerProfile + логинится + возвращает токен
	employerToken, _, employerProfile := helpers.CreateAndLoginEmployer(t, testServer)

	// 1.2. Создаем Модель
	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, testServer)

	// 1.3. Готовим тело кастинга
	castingBody := map[string]interface{}{
		"title":       "Супер-кастинг для Рекламы",
		"description": "Ищем новые лица",
		"payment_min": 50000.0,
		"payment_max": 100000.0,
		"city":        "Almaty",
		"job_type":    "one_time",
	}

	// --- Шаг 1: Работодатель создает кастинг ---
	t.Log("ШАГ 1: Работодатель создает кастинг...")

	// 2. Действие (Act)
	resCreate, bodyCreate := testServer.SendRequest(t, "POST", "/api/v1/castings", employerToken, castingBody)

	// 3. Проверка (Assert)
	assert.Equal(t, http.StatusCreated, resCreate.StatusCode)

	// Парсим ответ, чтобы получить ID созданного кастинга
	var createdCasting struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	err := json.Unmarshal([]byte(bodyCreate), &createdCasting)
	assert.NoError(t, err)
	assert.Equal(t, "Супер-кастинг для Рекламы", createdCasting.Title)
	assert.NotEmpty(t, createdCasting.ID, "ID кастинга не должен быть пустым")

	// Проверяем, что кастинг реально появился в БД
	var dbCasting models.Casting
	err = testServer.DB.First(&dbCasting, "id = ?", createdCasting.ID).Error
	assert.NoError(t, err, "Кастинг должен быть в БД")
	assert.Equal(t, employerProfile.ID, dbCasting.EmployerID, "ID работодателя в кастинге должен быть верным")

	// --- Шаг 2: Модель откликается на кастинг ---
	t.Log("ШАГ 2: Модель откликается на кастинг...")

	// 1. Подготовка (Arrange)
	responseBody := map[string]interface{}{
		"message": "Очень заинтересована, готова прийти на пробы!",
	}
	responseURL := fmt.Sprintf("/api/v1/responses/castings/%s", createdCasting.ID)

	// 2. Действие (Act)
	resRespond, bodyRespond := testServer.SendRequest(t, "POST", responseURL, modelToken, responseBody)

	// 3. Проверка (Assert)
	assert.Equal(t, http.StatusCreated, resRespond.StatusCode)

	// Проверяем, что отклик появился в БД
	var dbResponse models.CastingResponse
	err = testServer.DB.First(&dbResponse, "casting_id = ? AND model_id = ?", dbCasting.ID, modelProfile.ID).Error
	assert.NoError(t, err, "Отклик должен быть в БД")
	assert.Equal(t, "Очень заинтересована, готова прийти на пробы!", *dbResponse.Message)

	// --- Шаг 3: Работодатель просматривает отклики ---
	t.Log("ШАГ 3: Работодатель просматривает отклики...")

	// 1. Подготовка (Arrange)
	responsesListURL := fmt.Sprintf("/api/v1/responses/castings/%s/list", createdCasting.ID)

	// 2. Действие (Act)
	resList, bodyList := testServer.SendRequest(t, "GET", responsesListURL, employerToken, nil)

	// 3. Проверка (Assert)
	assert.Equal(t, http.StatusOK, resList.StatusCode)
	// Проверяем, что в списке откликов есть наша модель
	assert.Contains(t, bodyList, modelProfile.Name)
	assert.Contains(t, bodyList, "Очень заинтересована, готова прийти на пробы!")
	t.Logf("ПОЛНЫЙ СЦЕНАРИЙ КАСТИНГА: Успешно пройден.")
}

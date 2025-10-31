package integration_test

import (
	"encoding/json"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCasting_FullFlow - проверяет E2E "золотой путь" для Работодателя
func TestCasting_FullFlow(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// 2. Действие: Создание кастинга (POST)
	castingBody := map[string]interface{}{
		"title":       "Test Casting",
		"city":        "Almaty",
		"description": "Нужны модели для съемки",
		"payment_min": 50000,
		"payment_max": 100000,
		"status":      "active", // Сразу публикуем
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/castings", employerToken, castingBody)

	// 3. Проверка: Создание
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Casting created successfully")
	t.Logf("КАСТИНГ: Создание (201) - Успешно. Ответ: %s", bodyStr)

	// 4. Действие: Получение своих кастингов (GET /my)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/castings/my", employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Test Casting")
	assert.Contains(t, bodyStr, "Almaty")

	// Парсим ID созданного кастинга
	var myCastings struct {
		Castings []models.Casting `json:"castings"`
	}
	err := json.Unmarshal([]byte(bodyStr), &myCastings)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(myCastings.Castings), "Должен быть 1 кастинг")
	createdCastingID := myCastings.Castings[0].ID
	t.Logf("КАСТИНГ: Получение /my (200) - Успешно. Найден ID: %s", createdCastingID)

	// 5. Действие: Обновление кастинга (PUT)
	updateBody := map[string]interface{}{
		"title": "Updated Title",
		"city":  "Astana", // Меняем город
	}
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/castings/"+createdCastingID, employerToken, updateBody)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Casting updated successfully")
	t.Logf("КАСТИНГ: Обновление (200) - Успешно.")

	// 6. Действие: Проверка обновления (GET /:castingId, публичный)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/castings/"+createdCastingID, "", nil) // 👈 без токена
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Updated Title")
	assert.Contains(t, bodyStr, "Astana")
	t.Logf("КАСТИНГ: Публичное чтение (200) - Успешно. Обновления применились.")

	// 7. Действие: Удаление кастинга (DELETE)
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/castings/"+createdCastingID, employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Casting deleted successfully")
	t.Logf("КАСТИНГ: Удаление (200) - Успешно.")

	// 8. Действие: Проверка удаления (GET /my)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/castings/my", employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"castings":[]`) // Ожидаем пустой массив
	t.Logf("КАСТИНГ: Проверка удаления (200) - Успешно. Массив пуст.")
}

// TestCasting_PublicRead - проверяет публичные роуты (поиск, по городу)
func TestCasting_PublicRead(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем работодателя и 2 кастинга
	_, user, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_ = CreateTestCasting(t, tx, user.ID, "Кастинг в Алматы", "Almaty")
	_ = CreateTestCasting(t, tx, user.ID, "Кастинг в Астане", "Astana")
	// Создаем модель (для роута /matching)
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Действие: Поиск по городу (GET /castings?city=...)
	res, bodyStr := ts.SendRequest(t, "GET", "/api/v1/castings?city=Almaty", "", nil)
	// 3. Проверка
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Кастинг в Алматы")
	assert.NotContains(t, bodyStr, "Кастинг в Астане")
	t.Logf("ПОИСК (Public): Поиск по городу (200) - Успешно.")

	// 2. Действие: Поиск по другому городу (GET /castings/city/...)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/castings/city/Astana", "", nil)
	// 3. Проверка
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Кастинг в Астане")
	assert.NotContains(t, bodyStr, "Кастинг в Алматы")
	t.Logf("ПОИСК (Public): GetByCity (200) - Успешно.")

	// 2. Действие: Поиск подходящих (GET /matching) (Роль: Модель)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/castings/matching", modelToken, nil)
	// 3. Проверка
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"castings":`) // Просто проверяем, что роут работает
	t.Logf("ПОИСК (Model): GetMatching (200) - Успешно.")
}

// TestCasting_Security - проверяет права доступа (401, 403, 404)
func TestCasting_Security(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Работодатель А создает кастинг
	_, userA, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	castingA := CreateTestCasting(t, tx, userA.ID, "Casting A", "Almaty")

	// Работодатель Б (используем базовый хелпер для уникальности)
	employerTokenB, _ := helpers.CreateAndLoginUser(t, ts, tx, "Employer B", "b@test.com", "pass123", models.UserRoleEmployer)

	// Модель
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Действие: Модель пытается создать кастинг (POST)
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/castings", modelToken, map[string]interface{}{"title": "Hack", "city": "Hack"})
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Contains(t, bodyStr, "FORBIDDEN")
	t.Logf("БЕЗОПАСНОСТЬ: Модель не может создать кастинг (403) - Успешно.")

	// 2. Действие: Аноним пытается создать кастинг (POST)
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/castings", "", map[string]interface{}{"title": "Hack", "city": "Hack"})
	// 3. Проверка: (401 Unauthorized)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	assert.Contains(t, bodyStr, "UNAUTHORIZED")
	t.Logf("БЕЗОПАСНОСТЬ: Аноним не может создать кастинг (401) - Успешно.")

	// 2. Действие: Работодатель Б пытается удалить кастинг Работодателя А (DELETE)
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/castings/"+castingA.ID, employerTokenB, nil)
	// 3. Проверка: (404 Not Found или 403 Forbidden)
	// (Т.к. сервис ищет кастинг по ID И ID работодателя, он его "не найдет")
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ: Работодатель Б не может удалить чужой кастинг (%d) - Успешно.", res.StatusCode)

	// 2. Действие: Работодатель Б пытается обновить кастинг Работодателя А (PUT)
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/castings/"+castingA.ID, employerTokenB, map[string]interface{}{"title": "Hack"})
	// 3. Проверка: (404 Not Found или 403 Forbidden)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ: Работодатель Б не может обновить чужой кастинг (%d) - Успешно.", res.StatusCode)
}

// TestCasting_Responses - проверяет функционал откликов на кастинги
func TestCasting_Responses(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем работодателя и кастинг
	employerToken, employer, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	casting := CreateTestCasting(t, tx, employer.ID, "Test Casting for Responses", "Almaty")

	// Создаем модель
	modelToken, model, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Действие: Модель откликается на кастинг
	responseBody := map[string]interface{}{
		"message": "Я хочу участвовать в этом кастинге!",
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/castings/"+casting.ID+"/responses", modelToken, responseBody)

	// 3. Проверка: Отклик создан
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Response submitted successfully")
	t.Logf("ОТКЛИК: Создание (201) - Успешно. Ответ: %s", bodyStr)

	// 4. Действие: Работодатель проверяет отклики на свой кастинг
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/castings/"+casting.ID+"/responses", employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, model.Name)
	assert.Contains(t, bodyStr, "Я хочу участвовать в этом кастинге!")
	t.Logf("ОТКЛИК: Получение работодателем (200) - Успешно.")
}

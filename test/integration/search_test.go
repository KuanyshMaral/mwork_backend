package integration_test

import (
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSearch - главный тест-сьют для всех эндпоинтов /search.
func TestSearch(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Setup
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// 2. Создаем пользователей
	// Админ для /admin/search
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx,
		"Admin Search",
		"admin-search@test.com",
		"password123",
		models.UserRoleAdmin,
	)

	// Модель для /search/history и как результат поиска
	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)

	// Работодатель для /search/history и как результат поиска
	empToken, empUser, empProfile := helpers.CreateAndLoginEmployer(t, ts, tx)

	// 3. Создаем данные для поиска
	casting1 := CreateTestCasting(t, tx, empUser.ID, "Casting for Actors", "Almaty")
	_ = CreateTestCasting(t, tx, empUser.ID, "Casting for Dancers", "Astana") // ✅ Fixed: removed unused casting2

	// --- 4. Запускаем sub-тесты ---

	t.Run("POST /search/castings - Public POST Search", func(t *testing.T) {
		endpoint := "/api/v1/search/castings"
		body := gin.H{
			"city":     "Almaty", // Должен найти casting1
			"page":     1,
			"pageSize": 10,
		}

		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, endpoint, "", body)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Public search should succeed. Body: "+bodyStr)
		// Проверяем структуру PaginatedResponse
		assert.Contains(t, bodyStr, `"total":1`, "Should find 1 casting in Almaty")
		assert.Contains(t, bodyStr, `"page":1`)
		assert.Contains(t, bodyStr, casting1.ID, "Response should contain Almaty casting ID")

		// Тест 2: Пустой результат
		body["city"] = "Mordor"
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, endpoint, "", body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"total":0`, "Should find 0 castings in Mordor")
		assert.Contains(t, bodyStr, `"data":[]`, "Data array should be empty")

		// Тест 3: Ошибка валидации (невалидный pageSize)
		body = gin.H{"page": 1, "pageSize": 999}
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, endpoint, "", body)
		// (Предполагаем, что сервис-слой или DTO имеет валидацию max=100)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Should fail validation for large pageSize. Body: "+bodyStr)
	})

	t.Run("GET /search/.../suggestions - Public GET Suggestions", func(t *testing.T) {
		endpoint := "/api/v1/search/castings/suggestions"

		// 1. Ошибка: Нет query
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Contains(t, bodyStr, "query parameter is required", "Error message should be correct")

		// 2. Успешно (Кастинги)
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint+"?query=Actor&limit=5", "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"suggestions":`, "Response should contain suggestions key")

		// 3. Успешно (Модели)
		endpointModels := "/api/v1/search/models/suggestions"
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpointModels+"?query=Model&limit=5", "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"suggestions":`)
	})

	t.Run("POST /search/models and /search/employers - Public POST Search", func(t *testing.T) {
		// 1. Поиск Моделей
		modelEndpoint := "/api/v1/search/models"
		body := gin.H{
			"city":     "Almaty", // Должен найти modelProfile
			"page":     1,
			"pageSize": 10,
		}
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, modelEndpoint, "", body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"total":1`, "Should find 1 model in Almaty")
		assert.Contains(t, bodyStr, modelProfile.ID, "Response should contain model ID")

		// 2. Поиск Работодателей
		empEndpoint := "/api/v1/search/employers"
		body = gin.H{
			"city":     "Almaty", // Должен найти empProfile
			"page":     1,
			"pageSize": 10,
		}
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, empEndpoint, "", body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"total":1`, "Should find 1 employer in Almaty")
		assert.Contains(t, bodyStr, empProfile.ID, "Response should contain employer ID")
	})

	t.Run("POST /search/unified - Public Unified Search", func(t *testing.T) {
		endpoint := "/api/v1/search/unified"
		// Ищем по "Almaty", должны найти 1 модель, 1 работодателя, 1 кастинг
		body := gin.H{
			"query":    "Almaty",
			"page":     1,
			"pageSize": 10,
		}

		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, endpoint, "", body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		// Проверяем, что в ответе есть все 3 типа
		assert.Contains(t, bodyStr, `"castings":`, "Unified search should return castings")
		assert.Contains(t, bodyStr, `"models":`, "Unified search should return models")
		assert.Contains(t, bodyStr, `"employers":`, "Unified search should return employers")
		// (Можно добавить более детальные проверки на total_found в каждом разделе)
	})

	t.Run("GET /search/history - Protected History", func(t *testing.T) {
		endpoint := "/api/v1/search/history"

		// 1. Ошибка: Без токена
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "History endpoint requires auth. Body: "+bodyStr)

		// 2. Успешно (Модель)
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"history":`, "Response should contain history key")

		// 3. Успешно (Работодатель)
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("DELETE /search/history - Protected History Clear", func(t *testing.T) {
		endpoint := "/api/v1/search/history"

		// (Предположим, что сервис записал историю поиска для modelToken)

		// 1. Успешно (Модель чистит свою историю)
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodDelete, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, "history cleared", "Success message mismatch")

		// 2. Проверяем, что история пуста
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, `"total":0`, "History total should be 0 after delete")

		// 3. Ошибка: Без токена
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodDelete, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("/admin/search - Admin Endpoints", func(t *testing.T) {
		analyticsEndpoint := "/api/v1/admin/search/analytics"
		reindexEndpoint := "/api/v1/admin/search/reindex"

		// 1. Ошибка: Без токена
		// ❗️ Добавлен 'tx'
		res, _ := ts.SendRequest(t, tx, http.MethodGet, analyticsEndpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

		// 2. Ошибка: Не та роль (Модель)
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, analyticsEndpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "RoleMiddleware should block non-admin. Body: "+bodyStr)

		// 3. Ошибка: Не та роль (Работодатель)
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodPost, reindexEndpoint, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "RoleMiddleware should block non-admin")

		// 4. Успешно: Админ GET Analytics
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, analyticsEndpoint, adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, "total_searches", "Admin analytics response missing keys") // (Предположение о DTO)

		// 5. Успешно: Админ POST Reindex
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, reindexEndpoint, adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, "reindexing started", "Admin reindex response mismatch")
	})
}

// TestSearch_Isolated - отдельные изолированные тесты для лучшего параллелизма
func TestSearch_CastingsIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем кастинги в разных городах
	_, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ✅ Fixed: removed unused empToken
	CreateTestCasting(t, tx, empUser.ID, "Almaty Casting", "Almaty")
	CreateTestCasting(t, tx, empUser.ID, "Astana Casting", "Astana")

	// Тестируем поиск по городу
	body := gin.H{
		"city":     "Almaty",
		"page":     1,
		"pageSize": 10,
	}
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, "/api/v1/search/castings", "", body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":1`, "Should find only 1 casting in Almaty")
	assert.Contains(t, bodyStr, "Almaty Casting")
	assert.NotContains(t, bodyStr, "Astana Casting")
}

func TestSearch_ModelsIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем модели в разных городах
	_, _, _ = helpers.CreateAndLoginModel(t, ts, tx) // ✅ Fixed: removed unused modelToken1

	// Создаем модель в другом городе
	userToken, _ := helpers.CreateAndLoginUser(t, ts, tx, // ✅ Fixed: 2 return values
		"Astana Model",
		"astana-model@test.com",
		"password123",
		models.UserRoleModel,
	)

	astanaBody := map[string]interface{}{
		"name":      "Astana Model",
		"age":       23,
		"height":    168,
		"weight":    54,
		"gender":    "female",
		"city":      "Astana", // Другой город
		"is_public": true,
	}
	// ❗️ Добавлен 'tx'
	ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", userToken, astanaBody)

	// Тестируем поиск моделей по городу
	body := gin.H{
		"city":     "Astana",
		"page":     1,
		"pageSize": 10,
	}
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, "/api/v1/search/models", "", body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":1`, "Should find only 1 model in Astana")
	assert.Contains(t, bodyStr, "Astana Model")
}

func TestSearch_SuggestionsIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем кастинг с уникальным названием
	_, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ✅ Fixed: removed unused empToken
	CreateTestCasting(t, tx, empUser.ID, "Unique Casting Name", "Almaty")

	// Тестируем suggestions
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/search/castings/suggestions?query=Unique&limit=5", "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Unique Casting Name", "Suggestions should contain matching casting")

	// Тестируем пустой запрос
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/search/castings/suggestions?query=NonExistent&limit=5", "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"suggestions":[]`, "Should return empty suggestions for non-existent query")
}

func TestSearch_HistorySecurity(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем двух пользователей
	modelToken1, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	modelToken2, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Каждый пользователь должен видеть только свою историю поиска
	// (предполагаем, что история сохраняется при поиске)

	// Пользователь 1 получает свою историю
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/search/history", modelToken1, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Пользователь 2 получает свою историю
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/search/history", modelToken2, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Пользователь 1 очищает свою историю
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodDelete, "/api/v1/search/history", modelToken1, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Проверяем, что история пользователя 1 пуста
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/search/history", modelToken1, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":0`, "User 1 history should be empty")

	// Проверяем, что история пользователя 2 не затронута
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/search/history", modelToken2, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	// История пользователя 2 должна остаться неизменной
}

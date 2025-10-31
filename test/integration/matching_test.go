package integration_test

import (
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestMatching - главный тест-сьют для всех эндпоинтов /matching и /admin/matching.
func TestMatching(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// 2. Создаем пользователей
	// Админ для /admin/matching
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx,
		"Admin Match",
		"admin-match@test.com",
		"password123",
		models.UserRoleAdmin,
	)

	// Модель 1 (для поиска и статистики)
	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)

	// Модель 2 (для поиска похожих)
	_, _, modelProfile2 := helpers.CreateAndLoginModel(t, ts, tx)

	// Работодатель (владелец кастингов)
	empToken, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// 3. Создаем Кастинги
	casting1 := CreateTestCasting(t, tx, empUser.ID, "Casting for Match Test", "Almaty")
	casting2 := CreateTestCasting(t, tx, empUser.ID, "Casting for Batch Test", "Astana")

	// --- 4. Запускаем sub-тесты ---

	t.Run("User Matching Routes (Protected)", func(t *testing.T) {
		// --- GET /castings/:castingId/models ---
		endpoint := "/api/v1/matching/castings/" + casting1.ID + "/models"

		// 1. Успешно (Работодатель ищет)
		res, bodyStr := ts.SendRequest(t, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Employer should find matches. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"matches":`, "Response should contain matches key")
		assert.Contains(t, bodyStr, modelProfile.ID, "Matches should include the created model")

		// 2. Успешно (Модель ищет, с параметрами)
		endpointWithParams := endpoint + "?limit=5&min_score=30"
		res, bodyStr = ts.SendRequest(t, http.MethodGet, endpointWithParams, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Model should find matches. Body: "+bodyStr)

		// 3. Ошибка: 401 (Без токена)
		res, _ = ts.SendRequest(t, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Endpoint should be protected")

		// --- POST /models/search ---
		searchBody := gin.H{
			"limit":     10,
			"min_score": 50,
			"criteria": gin.H{ // (Предполагаем структуру DTO)
				"city": "Almaty",
			},
		}
		res, bodyStr = ts.SendRequest(t, http.MethodPost, "/api/v1/matching/models/search", empToken, searchBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "POST search should succeed. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"matches":`, "Response should contain matches key")

		// --- GET /compatibility ---
		compEndpoint := "/api/v1/matching/compatibility?model_id=" + modelProfile.ID + "&casting_id=" + casting1.ID
		res, bodyStr = ts.SendRequest(t, http.MethodGet, compEndpoint, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Compatibility check should succeed. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"score":`, "Compatibility response should have score")

		// 1. Ошибка 400 (нет model_id)
		res, bodyStr = ts.SendRequest(t, http.MethodGet, "/api/v1/matching/compatibility?casting_id="+casting1.ID, modelToken, nil)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Should fail without model_id")
		assert.Contains(t, bodyStr, "required", "Error message should be correct")

		// --- GET /models/:modelId/similar ---
		similarEndpoint := "/api/v1/matching/models/" + modelProfile.ID + "/similar"
		res, bodyStr = ts.SendRequest(t, http.MethodGet, similarEndpoint, empToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Similar models search should succeed. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"similar_models":`)
		assert.Contains(t, bodyStr, modelProfile2.ID, "Similar models should find model 2")

		// --- GET /weights (Простая проверка) ---
		res, bodyStr = ts.SendRequest(t, http.MethodGet, "/api/v1/matching/weights", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		// (Предполагаем, что сервис возвращает JSON-объект с весами)
		assert.Contains(t, bodyStr, "{", "Weights should be a JSON object")

		// --- GET .../stats (Простая проверка) ---
		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/matching/castings/"+casting1.ID+"/stats", empToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Casting stats should be OK")

		res, _ = ts.SendRequest(t, http.MethodGet, "/api/v1/matching/models/"+modelProfile.ID+"/stats", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Model stats should be OK")
	})

	t.Run("Admin Matching Routes (Admin Only)", func(t *testing.T) {
		// --- PUT /weights ---
		weightsEndpoint := "/api/v1/admin/matching/weights"
		weightsBody := gin.H{
			"age_weight":    0.5,
			"city_weight":   1.5,
			"rating_weight": 0.0, // (Пример DTO)
		}

		// 1. Ошибка 403 (Модель)
		res, bodyStr := ts.SendRequest(t, http.MethodPut, weightsEndpoint, modelToken, weightsBody)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Model should be forbidden. Body: "+bodyStr)

		// 2. Ошибка 403 (Работодатель)
		res, _ = ts.SendRequest(t, http.MethodPut, weightsEndpoint, empToken, weightsBody)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Employer should be forbidden")

		// 3. Успешно (Админ)
		res, bodyStr = ts.SendRequest(t, http.MethodPut, weightsEndpoint, adminToken, weightsBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should update weights. Body: "+bodyStr)
		assert.Contains(t, bodyStr, "updated successfully", "Success message mismatch")

		// --- GET /stats/platform ---
		statsEndpoint := "/api/v1/admin/matching/stats/platform"
		res, _ = ts.SendRequest(t, http.MethodGet, statsEndpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Stats is admin-only")
		res, bodyStr = ts.SendRequest(t, http.MethodGet, statsEndpoint, adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should get stats. Body: "+bodyStr)
		assert.Contains(t, bodyStr, "avg_match_score", "Stats response missing keys") // (Предположение о DTO)

		// --- POST /recalculate ---
		recalcEndpoint := "/api/v1/admin/matching/recalculate"
		res, _ = ts.SendRequest(t, http.MethodPost, recalcEndpoint, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Recalculate is admin-only")
		res, bodyStr = ts.SendRequest(t, http.MethodPost, recalcEndpoint, adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should start recalculation. Body: "+bodyStr)

		// --- GET /logs ---
		logsEndpoint := "/api/v1/admin/matching/logs"
		res, _ = ts.SendRequest(t, http.MethodGet, logsEndpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Logs are admin-only")
		res, bodyStr = ts.SendRequest(t, http.MethodGet, logsEndpoint+"?page=1&pageSize=5", adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should get logs. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"logs":`, "Logs response missing keys")
		assert.Contains(t, bodyStr, `"page":1`, "Logs response pagination mismatch")

		// --- POST /batch ---
		batchEndpoint := "/api/v1/admin/matching/batch"
		batchBody := gin.H{"casting_ids": []string{casting1.ID, casting2.ID}}

		// 1. Ошибка 403 (Модель)
		res, _ = ts.SendRequest(t, http.MethodPost, batchEndpoint, modelToken, batchBody)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Batch match is admin-only")

		// 2. Ошибка 400 (Админ, неверное тело)
		res, bodyStr = ts.SendRequest(t, http.MethodPost, batchEndpoint, adminToken, gin.H{})
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Should fail validation. Body: "+bodyStr)

		// 3. Успешно (Админ)
		res, bodyStr = ts.SendRequest(t, http.MethodPost, batchEndpoint, adminToken, batchBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should run batch match. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"results":`, "Batch response missing keys")
		assert.Contains(t, bodyStr, casting1.ID, "Batch response missing casting 1")
		assert.Contains(t, bodyStr, casting2.ID, "Batch response missing casting 2")
	})
}

// TestMatching_Isolated - отдельные тесты для лучшей изоляции
func TestMatching_BasicSearch(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем минимальный набор данных
	empToken, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	modelToken, _, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, empUser.ID, "Simple Casting", "Almaty")

	// Тестируем базовый поиск
	endpoint := "/api/v1/matching/castings/" + casting.ID + "/models"

	// Работодатель ищет модели
	res, bodyStr := ts.SendRequest(t, http.MethodGet, endpoint, empToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, modelProfile.ID)

	// Модель ищет кастинги (через compatibility)
	compEndpoint := "/api/v1/matching/compatibility?model_id=" + modelProfile.ID + "&casting_id=" + casting.ID
	res, bodyStr = ts.SendRequest(t, http.MethodGet, compEndpoint, modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"score":`)
}

func TestMatching_AdminSecurity(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей разных ролей
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin2@test.com", "pass", models.UserRoleAdmin)
	empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Тестируем защиту админских эндпоинтов
	adminEndpoints := []string{
		"/api/v1/admin/matching/weights",
		"/api/v1/admin/matching/stats/platform",
		"/api/v1/admin/matching/recalculate",
		"/api/v1/admin/matching/logs",
		"/api/v1/admin/matching/batch",
	}

	for _, endpoint := range adminEndpoints {
		// Модель не может получить доступ
		res, _ := ts.SendRequest(t, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Model should be forbidden for: "+endpoint)

		// Работодатель не может получить доступ
		res, _ = ts.SendRequest(t, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Employer should be forbidden for: "+endpoint)

		// Админ может получить доступ (кроме POST/PUT которые требуют тело)
		if endpoint == "/api/v1/admin/matching/weights" {
			// Для PUT endpoints проверяем с телом
			weightsBody := gin.H{"age_weight": 0.5, "city_weight": 1.5}
			res, bodyStr := ts.SendRequest(t, http.MethodPut, endpoint, adminToken, weightsBody)
			assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should access PUT: "+endpoint+", Body: "+bodyStr)
		} else if endpoint == "/api/v1/admin/matching/batch" {
			// Для POST endpoints проверяем с телом
			batchBody := gin.H{"casting_ids": []string{}}
			res, bodyStr := ts.SendRequest(t, http.MethodPost, endpoint, adminToken, batchBody)
			// Может быть 400 (нет кастингов) но не 403
			assert.NotEqual(t, http.StatusForbidden, res.StatusCode, "Admin should not be forbidden for POST: "+endpoint+", Body: "+bodyStr)
		} else {
			// Для GET endpoints
			res, bodyStr := ts.SendRequest(t, http.MethodGet, endpoint, adminToken, nil)
			assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should access GET: "+endpoint+", Body: "+bodyStr)
		}
	}
}

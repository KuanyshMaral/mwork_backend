package integration_test

import (
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAnalytics - главный тест-сьют для всех эндпоинтов аналитики.
func TestAnalytics(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Setup
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// 2. Создаем пользователей, необходимых для тестов.
	// Нужен АДМИН для защищенных роутов.
	adminToken, adminUser := helpers.CreateAndLoginUser(t, ts, tx,
		"Admin User",
		"admin@mwork.test",
		"password123",
		models.UserRoleAdmin,
	)

	// Нужен Работодатель для тестов /castings/:employer_id/performance
	empToken, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ❗️ Захватываем empToken

	// Нужна Модель для генерации данных
	modelToken, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx) // ❗️ Захватываем modelToken

	// 3. Создаем тестовые данные для аналитики
	casting1 := CreateTestCasting(t, tx, empUser.ID, "Analytics Casting 1", "Almaty")
	CreateTestResponse(t, tx, casting1.ID, modelUser.ID, models.ResponseStatusAccepted)
	// Создадим еще один кастинг для проверки подсчетов
	CreateTestCasting(t, tx, empUser.ID, "Analytics Casting 2", "Nur-Sultan")

	// --- 4. Запускаем sub-тесты для каждого эндпоинта ---

	t.Run("GET /platform/overview - Secured Access", func(t *testing.T) {
		endpoint := "/api/v1/analytics/platform/overview"

		// ❗️ Тест 1: Доступ БЕЗ токена (должен быть 401)
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Endpoint should be protected by AuthMiddleware. Body: "+bodyStr)

		// ❗️ Тест 2: Доступ С токеном (должен быть 200)
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Endpoint should be accessible with token. Body: "+bodyStr)

		// Проверяем, что в ответе есть ожидаемые поля
		assert.Contains(t, bodyStr, "total_users", "Response should contain 'total_users'")
		assert.Contains(t, bodyStr, "total_castings", "Response should contain 'total_castings'")
	})

	t.Run("GET /users/acquisition - With Query Params", func(t *testing.T) {
		// Задаем кастомный диапазон дат
		dateFrom := time.Now().AddDate(0, -1, 0).Format("2006-01-02") // 1 месяц назад
		dateTo := time.Now().Format("2006-01-02")
		endpoint := "/api/v1/analytics/users/acquisition?dateFrom=" + dateFrom + "&dateTo=" + dateTo

		// ❗️ Добавляем tx и adminToken
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, adminToken, nil)

		assert.Equal(t, http.StatusOK, res.StatusCode, "Endpoint should support query params. Body: "+bodyStr)
		assert.Contains(t, bodyStr, "new_users_count", "Response should contain acquisition data")
	})

	t.Run("GET /castings/:employer_id/performance - Specific Employer", func(t *testing.T) {
		// Используем ID работодателя, созданного в setup
		endpoint := "/api/v1/analytics/castings/" + empUser.ID + "/performance"

		// ❗️ Тест 1: Доступ БЕЗ токена (должен быть 401)
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Endpoint should be protected. Body: "+bodyStr)

		// ❗️ Тест 2: Доступ С токеном (используем токен работодателя)
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Endpoint should be accessible with token. Body: "+bodyStr)

		// В ответе должны быть данные о 2 кастингах, которые мы создали
		assert.Contains(t, bodyStr, "total_castings", "Response should contain 'total_castings'")
		assert.Contains(t, bodyStr, "total_responses", "Response should contain 'total_responses'")
	})

	t.Run("GET /categories/popular - With Query Params", func(t *testing.T) {
		endpoint := "/api/v1/analytics/categories/popular?days=30&limit=5"

		// ❗️ Добавляем tx и modelToken (подойдет любой валидный токен)
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)

		assert.Equal(t, http.StatusOK, res.StatusCode, "Endpoint should be accessible. Body: "+bodyStr)
		// Ожидаем массив
		assert.True(t, (len(bodyStr) > 0 && bodyStr[0] == '['), "Response should be a JSON array")
	})

	// --- Тестирование защищенных (Admin) эндпоинтов ---

	t.Run("GET /admin/dashboard - Admin Access Control", func(t *testing.T) {
		endpoint := "/api/v1/analytics/admin/dashboard"

		// 1. Тест: Доступ БЕЗ токена (должен быть 401 Unauthorized)
		// ❗️ Добавляем tx
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Admin endpoint should require auth. Body: "+bodyStr)

		// 2. Тест: Доступ с токеном МОДЕЛИ (должен быть 403 Forbidden)
		modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
		// ❗️ Добавляем tx
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Admin endpoint should forbid non-admin roles. Body: "+bodyStr)

		// 3. Тест: Доступ с токеном РАБОТОДАТЕЛЯ (должен быть 403 Forbidden)
		empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
		// ❗️ Добавляем tx
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Admin endpoint should forbid non-admin roles. Body: "+bodyStr)

		// 4. Тест: Доступ с токеном АДМИНА (должен быть 200 OK)
		// ❗️ Добавляем tx
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, adminToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin endpoint should allow admin role. Body: "+bodyStr)
		// Проверяем, что админская панель вернула какие-то данные
		assert.Contains(t, bodyStr, "kpi_metrics", "Admin dashboard response should contain data. Body: "+bodyStr)
		assert.Contains(t, bodyStr, adminUser.ID, "Dashboard data should be related to the admin user")
	})

	t.Run("POST /reports/custom - Admin POST with Body and Validation", func(t *testing.T) {
		endpoint := "/api/v1/analytics/reports/custom"

		reportReq := map[string]interface{}{
			"report_name": "Test Custom Report",
			"metrics":     []string{"total_users", "total_castings"},
			"dimensions":  []string{"city"},
			"date_from":   "2025-01-01T00:00:00Z",
			"date_to":     time.Now().Format(time.RFC3339),
		}

		// 1. Тест: Доступ БЕЗ токена (401)
		// ❗️ Добавляем tx
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, endpoint, "", reportReq)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Admin POST endpoint should require auth. Body: "+bodyStr)

		// 2. Тест: Доступ с токеном РАБОТОДАТЕЛЯ (403)
		empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
		// ❗️ Добавляем tx
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, endpoint, empToken, reportReq)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Admin POST endpoint should forbid non-admin roles. Body: "+bodyStr)

		// 3. Тест: Ошибка валидации (400 Bad Request) - неполные данные
		invalidReq := map[string]interface{}{
			"report_name": "Missing metrics", // Неполные данные
			"date_from":   "2025-01-01T00:00:00Z",
			"date_to":     time.Now().Format(time.RFC3339),
		}
		// ❗️ Добавляем tx
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, endpoint, adminToken, invalidReq)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Endpoint should return 400 on validation error. Body: "+bodyStr)
		assert.Contains(t, bodyStr, "metrics", "Error message should mention 'metrics' field")

		// 4. Тест: Успешный запрос от Админа (200 OK)
		// ❗️ Добавляем tx
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, endpoint, adminToken, reportReq)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Admin POST endpoint should allow admin with valid body. Body: "+bodyStr)
		// (Предполагаем, что сервис возвращает URL или данные отчета)
		assert.Contains(t, bodyStr, "report_id", "Response should contain generated report data. Body: "+bodyStr)
	})
}

// ❗️ Переименован: TestAnalytics_PublicEndpointsIsolated -> TestAnalytics_SecuredEndpointsIsolated
func TestAnalytics_SecuredEndpointsIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем минимальный набор данных для тестирования
	// ❗️ Нужен токен для аутентификации
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	_, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	CreateTestCasting(t, tx, empUser.ID, "Isolated Casting", "Almaty")

	endpoints := []string{
		"/api/v1/analytics/platform/overview",
		"/api/v1/analytics/categories/popular?days=7&limit=3",
		"/api/v1/analytics/users/acquisition?dateFrom=2025-01-01&dateTo=2025-01-31",
	}

	for _, endpoint := range endpoints {
		// ❗️ Тест 1: Проверяем защиту (401)
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Endpoint should be protected: "+endpoint+", Body: "+bodyStr)

		// ❗️ Тест 2: Проверяем доступ (200)
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Endpoint should be accessible with token: "+endpoint+", Body: "+bodyStr)
	}
}

func TestAnalytics_EmployerPerformanceIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем работодателя с несколькими кастингами
	empToken, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ❗️ Захватываем empToken
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем несколько кастингов и откликов
	casting1 := CreateTestCasting(t, tx, empUser.ID, "Performance Casting 1", "Almaty")
	casting2 := CreateTestCasting(t, tx, empUser.ID, "Performance Casting 2", "Astana")
	CreateTestResponse(t, tx, casting1.ID, modelUser.ID, models.ResponseStatusAccepted)
	CreateTestResponse(t, tx, casting2.ID, modelUser.ID, models.ResponseStatusPending)

	// Тестируем эндпоинт производительности работодателя
	endpoint := "/api/v1/analytics/castings/" + empUser.ID + "/performance"

	// ❗️ Добавляем tx и empToken
	res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, empToken, nil)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total_castings":2`)
	assert.Contains(t, bodyStr, `"total_responses":2`)
}

func TestAnalytics_AdminSecurityIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей разных ролей
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin-analytics@test.com", "adminpass", models.UserRoleAdmin)
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Тестируем защиту админских эндпоинтов
	adminEndpoints := []string{
		"/api/v1/analytics/admin/dashboard",
		"/api/v1/analytics/reports/custom",
	}

	for _, endpoint := range adminEndpoints {
		// Модель не может получить доступ
		// ❗️ Добавляем tx
		res, _ := ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Model should be forbidden for: "+endpoint)

		// Работодатель не может получить доступ
		// ❗️ Добавляем tx
		res, _ = ts.SendRequest(t, tx, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Employer should be forbidden for: "+endpoint)

		// Админ может получить доступ
		if endpoint == "/api/v1/analytics/reports/custom" {
			// Для POST endpoints проверяем с телом
			reportReq := map[string]interface{}{
				"report_name": "Test Report",
				"metrics":     []string{"total_users"},
				"dimensions":  []string{"city"},
				"date_from":   "2025-01-01T00:00:00Z",
				"date_to":     time.Now().Format(time.RFC3339),
			}
			// ❗️ Добавляем tx
			res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, endpoint, adminToken, reportReq)
			// Может быть 200 или 400 (в зависимости от валидации), но не 403
			assert.NotEqual(t, http.StatusForbidden, res.StatusCode, "Admin should not be forbidden for POST: "+endpoint+", Body: "+bodyStr)
		} else {
			// Для GET endpoints
			// ❗️ Добавляем tx
			res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, adminToken, nil)
			assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should access GET: "+endpoint+", Body: "+bodyStr)
		}
	}
}

func TestAnalytics_DataConsistencyIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем тестовые данные
	empToken, empUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)    // ❗️ Захватываем empToken
	modelToken1, modelUser1, _ := helpers.CreateAndLoginModel(t, ts, tx) // ❗️ Захватываем modelToken1
	_, modelUser2, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем кастинги и отклики
	casting := CreateTestCasting(t, tx, empUser.ID, "Consistency Test Casting", "Almaty")
	CreateTestResponse(t, tx, casting.ID, modelUser1.ID, models.ResponseStatusAccepted)
	CreateTestResponse(t, tx, casting.ID, modelUser2.ID, models.ResponseStatusRejected)

	// Проверяем консистентность данных в разных эндпоинтах
	// ❗️ Добавляем tx и modelToken1
	overviewRes, overviewBody := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/analytics/platform/overview", modelToken1, nil)
	assert.Equal(t, http.StatusOK, overviewRes.StatusCode)

	// ❗️ Добавляем tx и empToken (логичнее использовать токен владельца)
	performanceRes, performanceBody := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/analytics/castings/"+empUser.ID+"/performance", empToken, nil)
	assert.Equal(t, http.StatusOK, performanceRes.StatusCode)

	// Оба эндпоинта должны показывать консистентные данные
	assert.Contains(t, overviewBody, `"total_castings":1`)
	assert.Contains(t, performanceBody, `"total_castings":1`)
	assert.Contains(t, performanceBody, `"total_responses":2`)
}

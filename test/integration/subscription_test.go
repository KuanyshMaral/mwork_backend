package integration_test

import (
	"encoding/json"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSubscription_PublicPlanListing - Проверяет, что анонимный пользователь может видеть планы
func TestSubscription_PublicPlanListing(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// 2. Действие: Анонимный юзер запрашивает планы
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/plans", "", nil)

	// 3. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"name":"Free"`, "Список планов должен содержать 'Free'")
	assert.Contains(t, bodyStr, `"total":1`)
	t.Logf("ПОДПИСКИ (Public): GET /plans - Успешно.")
}

// TestSubscription_AdminPlanManagement - Проверяет E2E флоу Админа по управлению планами
func TestSubscription_AdminPlanManagement(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin@sub.com", "adminpass", models.UserRoleAdmin) // ✅ Fixed: 2 return values

	// 2. Действие: Админ создает "Premium" план (POST)
	planBody := map[string]interface{}{
		"name":      "Premium Model Plan",
		"price":     5000,
		"currency":  "KZT",
		"duration":  "monthly",
		"features":  map[string]any{"support": true},
		"limits":    map[string]int{"publications": 50, "responses": 100},
		"is_active": true,
	}
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "POST", "/api/v1/admin/plans", adminToken, planBody)

	// 3. Проверка: Создание
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Plan created successfully")
	t.Logf("ПОДПИСКИ (Admin): POST /admin/plans (201) - Успешно.")

	// 4. Действие: Получаем ID созданного плана (через публичный роут)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/plans", "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var plansResponse struct {
		Plans []models.SubscriptionPlan `json:"plans"`
	}
	err := json.Unmarshal([]byte(bodyStr), &plansResponse)
	assert.NoError(t, err)
	// Ищем наш план (Free + Premium = 2)
	assert.Equal(t, 2, len(plansResponse.Plans), "Должно быть 2 плана (Free + Premium)")
	var premiumPlan models.SubscriptionPlan
	for _, p := range plansResponse.Plans {
		if p.Name == "Premium Model Plan" {
			premiumPlan = p
			break
		}
	}
	assert.NotEmpty(t, premiumPlan.ID, "Не удалось найти ID созданного Premium плана")
	planID := premiumPlan.ID
	t.Logf("ПОДПИСКИ (Admin): План найден, ID: %s", planID)

	// 5. Действие: Админ обновляет план (PUT)
	updateBody := map[string]interface{}{
		"price":     5500,
		"is_active": false,
	}
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "PUT", "/api/v1/admin/plans/"+planID, adminToken, updateBody)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Plan updated successfully")
	t.Logf("ПОДПИСКИ (Admin): PUT /admin/plans/:id (200) - Успешно.")

	// 6. Действие: Проверяем обновление (GET /:planId)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/plans/"+planID, "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"Price":5500`)
	assert.Contains(t, bodyStr, `"IsActive":false`)
	t.Logf("ПОДПИСКИ (Admin): Проверка обновления (200) - Успешно.")

	// 7. Действие: Админ удаляет план (DELETE)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "DELETE", "/api/v1/admin/plans/"+planID, adminToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Plan deleted successfully")
	t.Logf("ПОДПИСКИ (Admin): DELETE /admin/plans/:id (200) - Успешно.")

	// 8. Действие: Проверяем удаление (GET /:planId)
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "GET", "/api/v1/plans/"+planID, "", nil)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	t.Logf("ПОДПИСКИ (Admin): Проверка удаления (404) - Успешно.")
}

// TestSubscription_UserFlow - Проверяет, как Модель апгрейдит подписку
func TestSubscription_UserFlow(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)                                                          // ✅ Fixed: removed unused modelUser
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin@sub.com", "adminpass", models.UserRoleAdmin) // ✅ Fixed: 2 return values

	// 2. Действие (Модель): Проверяем "мою" подписку (должна быть "Free")
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/my", modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"Name":"Free"`)
	t.Logf("ПОДПИСКИ (User): GET /my (Free) - Успешно.")

	// 3. Подготовка (Админ): Создаем "Premium" план
	planBody := map[string]interface{}{
		"name":      "Premium",
		"price":     1000,
		"currency":  "KZT",
		"duration":  "monthly",
		"features":  map[string]any{},
		"limits":    map[string]int{"publications": 10},
		"is_active": true,
	}
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "POST", "/api/v1/admin/plans", adminToken, planBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// ...и получаем его ID
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/plans", "", nil)
	var plansResponse struct {
		Plans []models.SubscriptionPlan `json:"plans"`
	}
	json.Unmarshal([]byte(bodyStr), &plansResponse)
	premiumPlanID := ""
	for _, p := range plansResponse.Plans {
		if p.Name == "Premium" {
			premiumPlanID = p.ID
			break
		}
	}

	// 4. Действие (Модель): Апгрейд подписки (POST /subscribe)
	subBody := map[string]interface{}{"plan_id": premiumPlanID}
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "POST", "/api/v1/subscriptions/subscribe", modelToken, subBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, `"PlanID":"`+premiumPlanID+`"`)
	t.Logf("ПОДПИСКИ (User): POST /subscribe (201) - Успешно.")

	// 5. Действие (Модель): Проверяем "мою" подписку (должна быть "Premium")
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/my", modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"Name":"Premium"`)
	assert.Contains(t, bodyStr, `"status":"active"`)
	t.Logf("ПОДПИСКИ (User): GET /my (Premium) - Успешно.")

	// 6. Действие (Модель): Проверяем лимиты
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/check-limit?feature=publications", modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"can_use":true`)
	t.Logf("ПОДПИСКИ (User): GET /check-limit - Успешно.")
}

// TestSubscription_Security - Проверяет права доступа (401, 403)
func TestSubscription_Security(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Действие: Аноним пытается получить /my
	// ❗️ Добавлен 'tx'
	res, _ := ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/my", "", nil)
	// 3. Проверка: (401 Unauthorized)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Sub): Аноним не может читать /my (401) - Успешно.")

	// 2. Действие: Аноним пытается создать /subscribe
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "POST", "/api/v1/subscriptions/subscribe", "", nil)
	// 3. Проверка: (401 Unauthorized)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Sub): Аноним не может /subscribe (401) - Успешно.")

	// 2. Действие: Модель пытается создать План (роут Админа)
	planBody := map[string]interface{}{"name": "Hacked Plan", "price": 1}
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "POST", "/api/v1/admin/plans", modelToken, planBody)
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Sub): Модель не может /admin/plans (403) - Успешно.")

	// 2. Действие: Модель пытается получить статистику (роут Админа)
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "GET", "/api/v1/admin/subscriptions/stats/platform", modelToken, nil)
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Sub): Модель не может /admin/subscriptions/stats (403) - Успешно.")
}

// TestSubscription_Isolated - отдельные изолированные тесты для лучшего параллелизма
func TestSubscription_PlanCreationIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin-isolated@test.com", "adminpass", models.UserRoleAdmin) // ✅ Fixed: 2 return values

	// Создаем план
	planBody := map[string]interface{}{
		"name":      "Isolated Plan",
		"price":     2000,
		"currency":  "KZT",
		"duration":  "monthly",
		"features":  map[string]any{"feature1": true},
		"limits":    map[string]int{"limit1": 5},
		"is_active": true,
	}
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "POST", "/api/v1/admin/plans", adminToken, planBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// Проверяем, что план создался
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/plans", "", nil)
	assert.Contains(t, bodyStr, "Isolated Plan")
}

func TestSubscription_UserSubscriptionIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin-user@test.com", "adminpass", models.UserRoleAdmin) // ✅ Fixed: 2 return values

	// Создаем план для подписки
	planBody := map[string]interface{}{
		"name":      "User Test Plan",
		"price":     1500,
		"currency":  "KZT",
		"duration":  "monthly",
		"is_active": true,
	}
	// ❗️ Добавлен 'tx'
	res, _ := ts.SendRequest(t, tx, "POST", "/api/v1/admin/plans", adminToken, planBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// Получаем ID плана
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/plans", "", nil)
	var plansResponse struct {
		Plans []models.SubscriptionPlan `json:"plans"`
	}
	json.Unmarshal([]byte(bodyStr), &plansResponse)

	var testPlanID string
	for _, p := range plansResponse.Plans {
		if p.Name == "User Test Plan" {
			testPlanID = p.ID
			break
		}
	}

	// Пользователь подписывается на план
	subBody := map[string]interface{}{"plan_id": testPlanID}
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "POST", "/api/v1/subscriptions/subscribe", modelToken, subBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// Проверяем, что подписка активна
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/my", modelToken, nil)
	assert.Contains(t, bodyStr, "User Test Plan")
	assert.Contains(t, bodyStr, `"status":"active"`)
}

func TestSubscription_LimitCheckingIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Проверяем лимиты для Free плана
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/check-limit?feature=publications", modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"can_use":true`)

	// Проверяем несуществующий лимит
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/subscriptions/check-limit?feature=non_existent", modelToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	// Может быть true или false в зависимости от реализации
}

func TestSubscription_AdminSecurityIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей разных ролей
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin-sec@test.com", "adminpass", models.UserRoleAdmin) // ✅ Fixed: 2 return values
	modelToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	empToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Тестируем защиту админских эндпоинтов
	adminEndpoints := []string{
		"/api/v1/admin/plans",
		"/api/v1/admin/subscriptions/stats/platform",
		"/api/v1/admin/subscriptions/revenue",
	}

	for _, endpoint := range adminEndpoints {
		// Модель не может получить доступ
		// ❗️ Добавлен 'tx'
		res, _ := ts.SendRequest(t, tx, http.MethodGet, endpoint, modelToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Model should be forbidden for: "+endpoint)

		// Работодатель не может получить доступ
		// ❗️ Добавлен 'tx'
		res, _ = ts.SendRequest(t, tx, http.MethodGet, endpoint, empToken, nil)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Employer should be forbidden for: "+endpoint)

		// Админ может получить доступ
		if endpoint == "/api/v1/admin/plans" {
			// Для POST endpoints проверяем с телом
			planBody := map[string]interface{}{"name": "Test Plan", "price": 1000, "currency": "KZT"}
			// ❗️ Добавлен 'tx'
			res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, endpoint, adminToken, planBody)
			assert.Equal(t, http.StatusCreated, res.StatusCode, "Admin should access POST: "+endpoint+", Body: "+bodyStr)
		} else {
			// Для GET endpoints
			// ❗️ Добавлен 'tx'
			res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpoint, adminToken, nil)
			assert.Equal(t, http.StatusOK, res.StatusCode, "Admin should access GET: "+endpoint+", Body: "+bodyStr)
		}
	}
}

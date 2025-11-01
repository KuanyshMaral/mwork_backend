package integration_test

import (
	"encoding/json" // ✅ Added missing import
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProfile - главный тест-сьют для всех эндпоинтов /profiles.
func TestProfile(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Setup
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// 2. Создаем пользователей для тестов

	// --- Пользователи для тестов СОЗДАНИЯ ---
	// Эти юзеры имеют аккаунт, но еще не имеют профиля
	userForModelToken, _ := helpers.CreateAndLoginUser(t, ts, tx,
		"New Model",
		"new-model@test.com",
		"password123",
		models.UserRoleModel,
	)

	userForEmpToken, _ := helpers.CreateAndLoginUser(t, ts, tx,
		"New Employer",
		"new-emp@test.com",
		"password123",
		models.UserRoleEmployer,
	)

	// --- Пользователи с УЖЕ СОЗДАННЫМИ профилями (для GET, PUT, Search) ---
	// Хелперы CreateAndLoginModel/Employer автоматически создают им профили в БД
	modelToken, modelUser, modelProfile := helpers.CreateAndLoginModel(t, ts, tx)

	empToken, empUser, empProfile := helpers.CreateAndLoginEmployer(t, ts, tx)

	// --- 3. Запускаем sub-тесты ---

	// Группа тестов на СОЗДАНИЕ профилей
	t.Run("POST /profiles - Profile Creation", func(t *testing.T) {
		// --- Тест создания профиля Модели ---
		modelBody := map[string]interface{}{
			"name":      "Test Model Name",
			"age":       22,
			"height":    178,
			"weight":    58,
			"gender":    "female",
			"city":      "Test City",
			"is_public": true,
		}

		// 1. Успешное создание
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", userForModelToken, modelBody)
		assert.Equal(t, http.StatusCreated, res.StatusCode, "Should create model profile. Body: "+bodyStr)

		// 2. Ошибка: Попытка создать профиль ЕЩЕ РАЗ
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", userForModelToken, modelBody)
		assert.Equal(t, http.StatusConflict, res.StatusCode, "Should return 409 Conflict when profile already exists. Body: "+bodyStr)

		// 3. Ошибка: Невалидное тело (отсутствует 'name')
		invalidModelBody := map[string]interface{}{"age": 22, "city": "Test City"}
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", userForModelToken, invalidModelBody)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Should return 400 on validation error. Body: "+bodyStr)

		// 4. Ошибка: Без токена
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", "", modelBody)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Should return 401 without token. Body: "+bodyStr)

		// 5. Ошибка: Не та роль (Работодатель пытается создать профиль Модели)
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", userForEmpToken, modelBody)
		assert.Equal(t, http.StatusForbidden, res.StatusCode, "Should return 403 Forbidden for wrong role. Body: "+bodyStr)

		// --- Тест создания профиля Работодателя ---
		empBody := map[string]interface{}{
			"company_name": "Test Company LLC",
			"city":         "Test City",
		}

		// 1. Успешное создание
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/employer", userForEmpToken, empBody)
		assert.Equal(t, http.StatusCreated, res.StatusCode, "Should create employer profile. Body: "+bodyStr)

		// 2. Ошибка: Попытка создать ЕЩЕ РАЗ
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/employer", userForEmpToken, empBody)
		assert.Equal(t, http.StatusConflict, res.StatusCode, "Should return 409 Conflict. Body: "+bodyStr)
	})

	// Группа тестов на ПОЛУЧЕНИЕ профилей
	t.Run("GET /profiles/:userId - Public Profile Retrieval", func(t *testing.T) {
		// 1. Успешное получение профиля Модели (используем ID из setup)
		endpointModel := "/api/v1/profiles/" + modelUser.ID
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpointModel, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should get model profile. Body: "+bodyStr)
		// Проверяем, что в ответе есть данные профиля
		assert.Contains(t, bodyStr, modelProfile.Name, "Response should contain model name")
		assert.Contains(t, bodyStr, `"type":"model"`, "Response should specify type 'model'")

		// 2. Успешное получение профиля Работодателя
		endpointEmp := "/api/v1/profiles/" + empUser.ID
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpointEmp, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should get employer profile. Body: "+bodyStr)
		assert.Contains(t, bodyStr, empProfile.CompanyName, "Response should contain company name")
		assert.Contains(t, bodyStr, `"type":"employer"`, "Response should specify type 'employer'")

		// 3. Ошибка: Несуществующий ID
		endpointNotFound := "/api/v1/profiles/non-existent-uuid"
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpointNotFound, "", nil)
		assert.Equal(t, http.StatusNotFound, res.StatusCode, "Should return 404 for invalid ID. Body: "+bodyStr)
	})

	// Группа тестов на ОБНОВЛЕНИЕ профилей
	t.Run("PUT /profiles/me - Profile Updates", func(t *testing.T) {
		// --- 1. Тест PUT /me (Обновление) ---
		updateBody := map[string]interface{}{
			"name":        "Updated Model Name",
			"city":        "Astana",
			"description": "New description here",
			"languages":   []string{"kazakh", "russian"},
			"hourly_rate": 5000.0,
		}

		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodPut, "/api/v1/profiles/me", modelToken, updateBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should update model profile. Body: "+bodyStr)

		// Проверяем в транзакции, что данные сохранились
		var updatedProfile models.ModelProfile
		err := tx.First(&updatedProfile, "user_id = ?", modelUser.ID).Error
		assert.NoError(t, err, "Failed to find updated profile in DB")
		assert.Equal(t, "Updated Model Name", updatedProfile.Name)
		assert.Equal(t, "Astana", updatedProfile.City)
		assert.Equal(t, 5000.0, updatedProfile.HourlyRate)

		// Проверяем JSON поле
		var languages []string
		err = json.Unmarshal(updatedProfile.Languages, &languages)
		assert.NoError(t, err)
		assert.Equal(t, []string{"kazakh", "russian"}, languages)

		// --- 2. Тест PUT /me/visibility (Видимость) ---
		// Хелпер создает профиль как 'IsPublic: true'
		// Сначала выключаем
		visibilityBody := map[string]interface{}{"is_public": false}
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPut, "/api/v1/profiles/me/visibility", modelToken, visibilityBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should toggle visibility off. Body: "+bodyStr)

		// Проверяем в транзакции
		tx.First(&updatedProfile, "user_id = ?", modelUser.ID)
		assert.False(t, updatedProfile.IsPublic, "Profile IsPublic should be false in DB")

		// Включаем обратно
		visibilityBody["is_public"] = true
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPut, "/api/v1/profiles/me/visibility", modelToken, visibilityBody)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should toggle visibility on. Body: "+bodyStr)

		// Проверяем в транзакции
		tx.First(&updatedProfile, "user_id = ?", modelUser.ID)
		assert.True(t, updatedProfile.IsPublic, "Profile IsPublic should be true in DB")

		// 3. Ошибка: Попытка обновить видимость для Работодателя (у него нет 'is_public')
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodPut, "/api/v1/profiles/me/visibility", empToken, visibilityBody)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Should fail to toggle visibility for Employer. Body: "+bodyStr)
	})

	// Тест ПОИСКА
	t.Run("GET /profiles/.../search - Public Search", func(t *testing.T) {
		// Хелперы создают обоих юзеров в 'Almaty'

		// 1. Поиск Моделей
		endpointModelSearch := "/api/v1/profiles/models/search?city=Almaty&page=1&pageSize=5"
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, endpointModelSearch, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should successfully search models. Body: "+bodyStr)
		// Должны найти 'existing-model' и 'new-model' (которому мы создали профиль)
		assert.Contains(t, bodyStr, `"total":2`, "Should find 2 models in Almaty")
		assert.Contains(t, bodyStr, modelProfile.Name, "Search results should include existing model")
		assert.Contains(t, bodyStr, "Test Model Name", "Search results should include new model")

		// 2. Поиск Работодателей
		endpointEmpSearch := "/api/v1/profiles/employers/search?city=Almaty"
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpointEmpSearch, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should successfully search employers. Body: "+bodyStr)
		// Должны найти 'existing-emp'
		assert.Contains(t, bodyStr, `"total":1`, "Should find 1 employer in Almaty")
		assert.Contains(t, bodyStr, empProfile.CompanyName, "Search results should include existing employer")

		// 3. Поиск Моделей (пустой результат)
		endpointModelSearchEmpty := "/api/v1/profiles/models/search?city=Mordor"
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, endpointModelSearchEmpty, "", nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should return 200 for empty search. Body: "+bodyStr)
		assert.Contains(t, bodyStr, `"total":0`, "Should find 0 models in Mordor")
		assert.Contains(t, bodyStr, `"profiles":[]`, "Profiles array should be empty")
	})

	// Тест СТАТИСТИКИ
	t.Run("GET /me/stats - Get My Stats", func(t *testing.T) {
		// 1. Успешно для Модели
		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/profiles/me/stats", modelToken, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode, "Should get stats for model. Body: "+bodyStr)
		// Проверяем наличие ключей статистики (значения могут быть 0)
		assert.Contains(t, bodyStr, "total_views", "Stats should contain total_views")
		assert.Contains(t, bodyStr, "total_responses", "Stats should contain total_responses")

		// 2. Ошибка: Без токена
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/profiles/me/stats", "", nil)
		assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Should return 401 without token. Body: "+bodyStr)

		// 3. Ошибка: Для Работодателя (хэндлер явно возвращает 400)
		// ❗️ Добавлен 'tx'
		res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/profiles/me/stats", empToken, nil)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Should return 400 for employer role. Body: "+bodyStr)
		assert.Contains(t, bodyStr, "Stats not available", "Error message should be specific")
	})
}

// TestProfile_Isolated - отдельные изолированные тесты для лучшего параллелизма
func TestProfile_CreationIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователя без профиля
	userToken, user := helpers.CreateAndLoginUser(t, ts, tx, // ✅ Fixed: 2 return values
		"Isolated Model",
		"isolated-model@test.com",
		"password123",
		models.UserRoleModel,
	)

	// Создаем профиль модели
	modelBody := map[string]interface{}{
		"name":      "Isolated Model Profile",
		"age":       25,
		"height":    170,
		"weight":    55,
		"gender":    "female",
		"city":      "Isolated City",
		"is_public": true,
	}

	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodPost, "/api/v1/profiles/model", userToken, modelBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode, "Should create model profile in isolated test. Body: "+bodyStr)

	// Проверяем, что профиль создался в транзакции
	var profile models.ModelProfile
	err := tx.First(&profile, "user_id = ?", user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Isolated Model Profile", profile.Name)
	assert.Equal(t, "Isolated City", profile.City)
}

func TestProfile_SearchIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем несколько моделей в разных городах
	_, _, _ = helpers.CreateAndLoginModel(t, ts, tx) // По умолчанию создается в "Almaty"

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

	// Тестируем поиск по городу
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/profiles/models/search?city=Astana", "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":1`, "Should find only 1 model in Astana")
	assert.Contains(t, bodyStr, "Astana Model")

	// Поиск в другом городе
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodGet, "/api/v1/profiles/models/search?city=Almaty", "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":1`, "Should find only 1 model in Almaty")
}

func TestProfile_SecurityIsolated(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем двух пользователей
	_, model1, _ := helpers.CreateAndLoginModel(t, ts, tx) // ✅ Fixed: removed unused modelToken1
	modelToken2, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Модель 2 пытается получить приватную статистику модели 1
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, http.MethodGet, "/api/v1/profiles/me/stats", modelToken2, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Should get own stats")
	assert.NotContains(t, bodyStr, model1.ID, "Should not see other user's data")

	// Модель 2 пытается обновить профиль модели 1
	updateBody := map[string]interface{}{"name": "Hacked Name"}
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, http.MethodPut, "/api/v1/profiles/me", modelToken2, updateBody)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Should update own profile")

	// Проверяем, что профиль модели 1 не изменился
	var profile models.ModelProfile
	err := tx.First(&profile, "user_id = ?", model1.ID).Error
	assert.NoError(t, err)
	assert.NotEqual(t, "Hacked Name", profile.Name, "Other user's profile should not be modified")
}

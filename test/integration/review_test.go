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

// TestReview_CreateFlow_And_CanCreate - Проверяет сложную логику "Могу ли я оставить отзыв"
func TestReview_CreateFlow_And_CanCreate(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, employerUser.ID, "Test Casting", "Almaty")

	canCreateURL := fmt.Sprintf("/api/v1/reviews/can-create?model_id=%s&casting_id=%s", modelUser.ID, casting.ID)

	// 2. Действие: Проверяем "CanCreate" (ДО отклика)
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", canCreateURL, employerToken, nil)

	// 3. Проверка: (Должно быть false)
	assert.Equal(t, http.StatusOK, res.StatusCode) // Ожидаем 200, но с can_create: false
	assert.Contains(t, bodyStr, `"can_create":false`)
	t.Logf("ОТЗЫВ (CanCreate): 'false' до отклика - Успешно.")

	// 4. Подготовка: Симулируем "работу" - Модель откликается, Работодатель принимает
	_ = CreateTestResponse(t, tx, casting.ID, modelUser.ID, models.ResponseStatusAccepted) // ✅ Fixed: using Accepted instead of Approved

	// 5. Действие: Проверяем "CanCreate" (ПОСЛЕ отклика)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", canCreateURL, employerToken, nil)

	// 6. Проверка: (Должно быть true)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"can_create":true`)
	t.Logf("ОТЗЫВ (CanCreate): 'true' после отклика - Успешно.")

	// 7. Действие: Создаем отзыв (POST)
	reviewBody := map[string]interface{}{
		"model_id":    modelUser.ID,
		"casting_id":  casting.ID,
		"rating":      5,
		"review_text": "Отличная работа!",
	}
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "POST", "/api/v1/reviews", employerToken, reviewBody)

	// 8. Проверка:
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Отличная работа!")
	t.Logf("ОТЗЫВ (Create): Создание (201) - Успешно.")

	// 9. Действие: Проверяем "CanCreate" (ПОСЛЕ создания отзыва)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", canCreateURL, employerToken, nil)

	// 10. Проверка: (Должно быть false, т.к. уже оставил)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode) // Или 400
	assert.Contains(t, bodyStr, "already reviewed")        // Ожидаем причину
	t.Logf("ОТЗЫВ (CanCreate): 'false' после создания отзыва - Успешно.")
}

// TestReview_EmployerFlow_AndUpdate_Delete - Проверяет GET /my, PUT, DELETE
func TestReview_EmployerFlow_AndUpdate_Delete(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем отзыв напрямую в транзакции
	review := CreateTestReview(t, tx, employerUser.ID, modelUser.ID, nil, 5, "Initial review")

	// 2. Действие: Получаем "мои" отзывы
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/reviews/my", employerToken, nil)

	// 3. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Initial review")
	assert.Contains(t, bodyStr, `"total":1`)
	t.Logf("ОТЗЫВ (Employer): GET /my - Успешно.")

	// 4. Действие: Обновляем отзыв
	updateBody := map[string]interface{}{
		"rating":      4,
		"review_text": "Updated review text",
	}
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "PUT", "/api/v1/reviews/"+review.ID, employerToken, updateBody)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	t.Logf("ОТЗЫВ (Employer): PUT /:id - Успешно.")

	// 5. Действие: Удаляем отзыв
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "DELETE", "/api/v1/reviews/"+review.ID, employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	t.Logf("ОТЗЫВ (Employer): DELETE /:id - Успешно.")

	// 6. Действие: Проверяем, что отзывов не осталось
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/reviews/my", employerToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, `"total":0`)
	t.Logf("ОТЗЫВ (Employer): GET /my (пусто) - Успешно.")
}

// TestReview_PublicRead - Проверяет публичные роуты (GET /:id, GET /models/:id, ...)
func TestReview_PublicRead(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	_, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)
	review := CreateTestReview(t, tx, employerUser.ID, modelUser.ID, nil, 5, "Public review text")

	// 2. Действие: GET /:reviewId (анонимно)
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/reviews/"+review.ID, "", nil)
	// 3. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Public review text")
	t.Logf("ОТЗЫВ (Public): GET /:id - Успешно.")

	// 2. Действие: GET /models/:modelId (анонимно)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/reviews/models/"+modelUser.ID, "", nil)
	// 3. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Public review text")
	assert.Contains(t, bodyStr, `"total":1`)
	t.Logf("ОТЗЫВ (Public): GET /models/:id - Успешно.")

	// 2. Действие: GET /models/:modelId/stats (анонимно)
	// ❗️ Добавлен 'tx'
	res, bodyStr = ts.SendRequest(t, tx, "GET", "/api/v1/reviews/models/"+modelUser.ID+"/stats", "", nil)
	// 3. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var stats struct {
		AverageRating float64        `json:"average_rating"`
		TotalReviews  int            `json:"total_reviews"`
		RatingCount   map[string]int `json:"rating_count"`
	}
	err := json.Unmarshal([]byte(bodyStr), &stats)
	assert.NoError(t, err)
	assert.Equal(t, 5.0, stats.AverageRating)
	assert.Equal(t, 1, stats.TotalReviews)
	assert.Equal(t, 1, stats.RatingCount["5"])
	t.Logf("ОТЗЫВ (Public): GET /models/:id/stats - Успешно.")
}

// TestReview_Security - Проверяет права доступа (401, 403, 404)
func TestReview_Security(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	tokenA, userA, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	tokenB, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	tokenC, userC, _ := helpers.CreateAndLoginModel(t, ts, tx)                                                           // Модель
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin@test.com", "adminpass", models.UserRoleAdmin) // ✅ Fixed: 2 return values

	reviewAC := CreateTestReview(t, tx, userA.ID, userC.ID, nil, 5, "Review from A to C")

	// 2. Действие: Модель (tokenC) пытается создать отзыв
	// ❗️ Добавлен 'tx'
	res, _ := ts.SendRequest(t, tx, "POST", "/api/v1/reviews", tokenC, map[string]interface{}{"rating": 1})
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Review): Модель не может создать отзыв (403) - Успешно.")

	// 2. Действие: Аноним пытается создать отзыв
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "POST", "/api/v1/reviews", "", map[string]interface{}{"rating": 1})
	// 3. Проверка: (401 Unauthorized)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Review): Аноним не может создать отзыв (401) - Успешно.")

	// 2. Действие: Работодатель Б (tokenB) пытается удалить отзыв Работодателя А
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "DELETE", "/api/v1/reviews/"+reviewAC.ID, tokenB, nil)
	// 3. Проверка: (404 Not Found или 403 Forbidden)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Review): Работодатель Б не может удалить чужой отзыв (%d) - Успешно.", res.StatusCode)

	// 2. Действие: Обычный юзер (tokenA) пытается получить доступ к роутам админа
	// ❗️ Добавлен 'tx'
	res, _ = ts.SendRequest(t, tx, "GET", "/admin/reviews/recent", tokenA, nil)
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Review): Обычный юзер не может читать /admin/reviews (403) - Успешно.")

	// 2. Действие: Админ (adminToken) получает доступ к роутам админа
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/admin/reviews/recent", adminToken, nil)
	// 3. Проверка: (200 OK)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Review from A to C")
	t.Logf("БЕЗОПАСНОСТЬ (Review): Админ УСПЕШНО читает /admin/reviews (200) - Успешно.")
}

// TestReview_RatingValidation - Проверяет валидацию рейтинга
func TestReview_RatingValidation(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)
	casting := CreateTestCasting(t, tx, employerUser.ID, "Rating Test Casting", "Almaty")
	_ = CreateTestResponse(t, tx, casting.ID, modelUser.ID, models.ResponseStatusAccepted) // ✅ Fixed: using Accepted

	// Тестируем граничные значения рейтинга
	testCases := []struct {
		rating     int
		shouldPass bool
	}{
		{0, false}, // слишком низко
		{1, true},  // минимум
		{3, true},  // нормально
		{5, true},  // максимум
		{6, false}, // слишком высоко
	}

	for _, tc := range testCases {
		reviewBody := map[string]interface{}{
			"model_id":    modelUser.ID,
			"casting_id":  casting.ID,
			"rating":      tc.rating,
			"review_text": "Test review",
		}

		// ❗️ Добавлен 'tx'
		res, bodyStr := ts.SendRequest(t, tx, "POST", "/api/v1/reviews", employerToken, reviewBody)

		if tc.shouldPass {
			assert.Equal(t, http.StatusCreated, res.StatusCode, "Rating %d should be valid", tc.rating)
		} else {
			assert.Equal(t, http.StatusBadRequest, res.StatusCode, "Rating %d should be invalid. Body: %s", tc.rating, bodyStr)
		}
	}
}

// TestReview_ModelStats - Проверяет статистику модели
func TestReview_ModelStats(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	_, employerUser1, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ✅ Fixed: removed unused employerToken1
	_, employerUser2, _ := helpers.CreateAndLoginEmployer(t, ts, tx) // ✅ Fixed: removed unused employerToken2
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем несколько отзывов с разными рейтингами
	CreateTestReview(t, tx, employerUser1.ID, modelUser.ID, nil, 5, "Excellent")
	CreateTestReview(t, tx, employerUser2.ID, modelUser.ID, nil, 4, "Good")
	CreateTestReview(t, tx, employerUser1.ID, modelUser.ID, nil, 3, "Average")

	// Проверяем статистику
	// ❗️ Добавлен 'tx'
	res, bodyStr := ts.SendRequest(t, tx, "GET", "/api/v1/reviews/models/"+modelUser.ID+"/stats", "", nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var stats struct {
		AverageRating float64        `json:"average_rating"`
		TotalReviews  int            `json:"total_reviews"`
		RatingCount   map[string]int `json:"rating_count"`
	}
	err := json.Unmarshal([]byte(bodyStr), &stats)
	assert.NoError(t, err)

	assert.Equal(t, 3, stats.TotalReviews)
	assert.InDelta(t, 4.0, stats.AverageRating, 0.1) // (5+4+3)/3 = 4.0
	assert.Equal(t, 1, stats.RatingCount["5"])
	assert.Equal(t, 1, stats.RatingCount["4"])
	assert.Equal(t, 1, stats.RatingCount["3"])
}

// TestReview_UpdateOwnReview - Проверяет, что можно обновлять только свои отзывы
func TestReview_UpdateOwnReview(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	_, employerUser1, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	employerToken2, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	_, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Работодатель 1 создает отзыв
	review := CreateTestReview(t, tx, employerUser1.ID, modelUser.ID, nil, 5, "Original review")

	// Работодатель 2 пытается обновить чужой отзыв
	updateBody := map[string]interface{}{
		"rating":      1,
		"review_text": "Hacked review",
	}
	// ❗️ Добавлен 'tx'
	res, _ := ts.SendRequest(t, tx, "PUT", "/api/v1/reviews/"+review.ID, employerToken2, updateBody)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, res.StatusCode, "Should not be able to update other's review")

	// Проверяем, что отзыв не изменился
	var currentReview models.Review
	err := tx.First(&currentReview, "id = ?", review.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 5, currentReview.Rating)
	assert.Equal(t, "Original review", currentReview.ReviewText)
}

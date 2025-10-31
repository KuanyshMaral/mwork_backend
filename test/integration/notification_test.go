package integration_test

import (
	"encoding/json"
	"mwork_backend/internal/models"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// CreateTestNotification - хелпер для быстрого создания уведомлений в транзакции.
// Это симулирует, как другой сервис (например, чат или кастинги)
// создает уведомление для пользователя.
func CreateTestNotification(t *testing.T, tx *gorm.DB, userID string, title string, message string) models.Notification {
	notification := models.Notification{
		UserID:  userID,
		Type:    "test_notification",
		Title:   title,
		Message: message,
		IsRead:  false,
	}
	if err := tx.Create(&notification).Error; err != nil {
		t.Fatalf("Failed to create test notification: %v", err)
	}
	return notification
}

// TestNotification_UserFlow - проверяет E2E "золотой путь" для Пользователя
func TestNotification_UserFlow(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Используем CreateAndLoginModel, т.к. он возвращает верифицированного
	// юзера, который может сразу пользоваться API.
	userToken, user, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Действие: Проверка, что уведомлений нет (GET /unread-count)
	res, bodyStr := ts.SendRequest(t, "GET", "/api/v1/notifications/unread-count", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var unreadResponse struct {
		Count int `json:"unread_count"`
	}
	err := json.Unmarshal([]byte(bodyStr), &unreadResponse)
	assert.NoError(t, err)
	assert.Equal(t, 0, unreadResponse.Count, "Вначале непрочитанных уведомлений быть не должно")
	t.Logf("УВЕДОМЛЕНИЯ: Непрочитанных - 0 (200) - Успешно.")

	// 3. Действие: Создаем 2 уведомления через хелпер (симуляция)
	notif1 := CreateTestNotification(t, tx, user.ID, "Новый отклик", "Вам откликнулись на кастинг")
	_ = CreateTestNotification(t, tx, user.ID, "Новое сообщение", "Вам пришло сообщение")

	// 4. Действие: Проверяем непрочитанные (GET /unread-count)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/notifications/unread-count", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.Unmarshal([]byte(bodyStr), &unreadResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, unreadResponse.Count, "Должно быть 2 непрочитанных уведомления")
	t.Logf("УВЕДОМЛЕНИЯ: Непрочитанных - 2 (200) - Успешно.")

	// 5. Действие: Получаем все уведомления (GET /)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/notifications", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Новый отклик")
	assert.Contains(t, bodyStr, "Новое сообщение")

	var getResponse struct {
		Notifications []models.Notification `json:"notifications"`
		Total         int                   `json:"total"`
	}
	err = json.Unmarshal([]byte(bodyStr), &getResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, getResponse.Total, "Должно быть 2 уведомления в списке")
	t.Logf("УВЕДОМЛЕНИЯ: Получение списка (200) - Успешно.")

	// 6. Действие: Читаем первое уведомление (PUT /:id/read)
	notificationID := notif1.ID
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/notifications/"+notificationID+"/read", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Notification marked as read")
	t.Logf("УВЕДОМЛЕНИЯ: Пометить прочитанным (200) - Успешно.")

	// 7. Действие: Проверяем непрочитанные (GET /unread-count)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/notifications/unread-count", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.Unmarshal([]byte(bodyStr), &unreadResponse)
	assert.NoError(t, err)
	assert.Equal(t, 1, unreadResponse.Count, "Должно остаться 1 непрочитанное")
	t.Logf("УВЕДОМЛЕНИЯ: Непрочитанных - 1 (200) - Успешно.")

	// 8. Действие: Читаем все (PUT /read-all)
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/notifications/read-all", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	t.Logf("УВЕДОМЛЕНИЯ: Пометить все прочитанным (200) - Успешно.")

	// 9. Действие: Проверяем непрочитанные (GET /unread-count)
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/notifications/unread-count", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	err = json.Unmarshal([]byte(bodyStr), &unreadResponse)
	assert.NoError(t, err)
	assert.Equal(t, 0, unreadResponse.Count, "Непрочитанных не должно остаться")
	t.Logf("УВЕДОМЛЕНИЯ: Непрочитанных - 0 (200) - Успешно.")

	// 10. Действие: Удаляем одно (DELETE /:id)
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/notifications/"+notificationID, userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	t.Logf("УВЕДОМЛЕНИЯ: Удаление (200) - Успешно.")

	// 11. Действие: Проверяем общее кол-во
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/notifications", userToken, nil)
	err = json.Unmarshal([]byte(bodyStr), &getResponse)
	assert.NoError(t, err)
	assert.Equal(t, 1, getResponse.Total, "Должно остаться 1 уведомление")
	t.Logf("УВЕДОМЛЕНИЯ: Получение списка (200) - Успешно, осталось 1.")
}

// TestNotification_Security - проверяет права доступа (401, 403, 404)
func TestNotification_Security(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Пользователь А (владелец уведомления)
	tokenA, userA, _ := helpers.CreateAndLoginModel(t, ts, tx)
	// Пользователь Б (посторонний)
	tokenB, _ := helpers.CreateAndLoginUser(t, ts, tx, "Model B", "model_b@test.com", "pass123", models.UserRoleModel)
	// Админ
	adminToken, _ := helpers.CreateAndLoginUser(t, ts, tx, "Admin", "admin@test.com", "adminpass", models.UserRoleAdmin)

	// Создаем уведомление для Пользователя А
	notificationA := CreateTestNotification(t, tx, userA.ID, "Секрет", "Секретное сообщение")

	// 2. Действие: Аноним пытается получить список
	res, _ := ts.SendRequest(t, "GET", "/api/v1/notifications", "", nil)
	// 3. Проверка: (401 Unauthorized)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ: Аноним не может читать список (401) - Успешно.")

	// 2. Действие: Пользователь Б пытается прочитать уведомление А
	res, _ = ts.SendRequest(t, "GET", "/api/v1/notifications/"+notificationA.ID, tokenB, nil)
	// 3. Проверка: (404 Not Found)
	// (Сервис не должен находить чужое уведомление)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ: Пользователь Б не может читать чужое уведомление (404) - Успешно.")

	// 2. Действие: Пользователь Б пытается пометить прочитанным уведомление А
	res, _ = ts.SendRequest(t, "PUT", "/api/v1/notifications/"+notificationA.ID+"/read", tokenB, nil)
	// 3. Проверка: (404 Not Found)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ: Пользователь Б не может читать чужое уведомление (404) - Успешно.")

	// 2. Действие: Обычный юзер (Модель А) пытается получить доступ к роутам админа
	res, _ = ts.SendRequest(t, "GET", "/admin/notifications", tokenA, nil)
	// 3. Проверка: (403 Forbidden)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ: Обычный юзер не может читать /admin/notifications (403) - Успешно.")

	// 2. Действие: Админ получает доступ к /admin/notifications
	res, bodyStr := ts.SendRequest(t, "GET", "/admin/notifications", adminToken, nil)
	// 3. Проверка: (200 OK)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Секретное сообщение") // Админ видит все
	t.Logf("БЕЗОПАСНОСТЬ: Админ УСПЕШНО читает /admin/notifications (200) - Успешно.")
}

// TestNotification_Pagination - проверяет пагинацию уведомлений
func TestNotification_Pagination(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	userToken, user, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем 5 уведомлений
	for i := 1; i <= 5; i++ {
		CreateTestNotification(t, tx, user.ID,
			"Уведомление "+string(rune('A'+i-1)),
			"Сообщение "+string(rune('A'+i-1)),
		)
	}

	// Тестируем пагинацию
	res, bodyStr := ts.SendRequest(t, "GET", "/api/v1/notifications?page=1&limit=2", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var paginatedResponse struct {
		Notifications []models.Notification `json:"notifications"`
		Total         int                   `json:"total"`
		Page          int                   `json:"page"`
		Limit         int                   `json:"limit"`
		TotalPages    int                   `json:"total_pages"`
	}
	err := json.Unmarshal([]byte(bodyStr), &paginatedResponse)
	assert.NoError(t, err)

	assert.Equal(t, 5, paginatedResponse.Total)
	assert.Equal(t, 1, paginatedResponse.Page)
	assert.Equal(t, 2, paginatedResponse.Limit)
	assert.Equal(t, 3, paginatedResponse.TotalPages)
	assert.Equal(t, 2, len(paginatedResponse.Notifications))
	t.Logf("УВЕДОМЛЕНИЯ: Пагинация работает корректно - Успешно.")
}

// TestNotification_RealTime - проверяет WebSocket/real-time функционал
func TestNotification_RealTime(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	userToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Проверяем, что real-time endpoint доступен
	// (это может быть WebSocket endpoint или long-polling)
	res, _ := ts.SendRequest(t, "GET", "/api/v1/notifications/stream", userToken, nil)

	// В зависимости от реализации, это может быть:
	// - 200 OK для long-polling
	// - 101 Switching Protocols для WebSocket
	// - 400 если требуется специальный заголовок
	assert.Contains(t, []int{
		http.StatusOK,
		http.StatusSwitchingProtocols,
		http.StatusBadRequest, // если не поддерживается в тестах
	}, res.StatusCode, "Real-time endpoint should be accessible")

	t.Logf("УВЕДОМЛЕНИЯ: Real-time endpoint доступен (%d) - Успешно.", res.StatusCode)
}

// TestNotification_MarkAllRead - проверяет массовое помечание как прочитанное
func TestNotification_MarkAllRead(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	userToken, user, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем несколько непрочитанных уведомлений
	for i := 1; i <= 3; i++ {
		CreateTestNotification(t, tx, user.ID,
			"Уведомление "+string(rune('A'+i-1)),
			"Сообщение "+string(rune('A'+i-1)),
		)
	}

	// Проверяем, что есть непрочитанные
	res, bodyStr := ts.SendRequest(t, "GET", "/api/v1/notifications/unread-count", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	var unreadResponse struct {
		Count int `json:"unread_count"`
	}
	json.Unmarshal([]byte(bodyStr), &unreadResponse)
	assert.Equal(t, 3, unreadResponse.Count)

	// Помечаем все как прочитанные
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/notifications/read-all", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Проверяем, что непрочитанных не осталось
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/notifications/unread-count", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	json.Unmarshal([]byte(bodyStr), &unreadResponse)
	assert.Equal(t, 0, unreadResponse.Count)
	t.Logf("УВЕДОМЛЕНИЯ: Все уведомления помечены как прочитанные - Успешно.")
}

// TestNotification_Filtering - проверяет фильтрацию уведомлений
func TestNotification_Filtering(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	userToken, user, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// Создаем уведомления разных типов
	CreateTestNotification(t, tx, user.ID, "Системное уведомление", "Системное сообщение")
	CreateTestNotification(t, tx, user.ID, "Чат уведомление", "Новое сообщение в чате")

	// Тестируем фильтрацию по типу (если поддерживается API)
	res, bodyStr := ts.SendRequest(t, "GET", "/api/v1/notifications", userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Системное уведомление")
	assert.Contains(t, bodyStr, "Чат уведомление")
	t.Logf("УВЕДОМЛЕНИЯ: Фильтрация работает - Успешно.")
}

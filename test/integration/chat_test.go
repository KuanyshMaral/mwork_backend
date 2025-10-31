package integration_test

import (
	"encoding/json"
	"mwork_backend/internal/models"
	chatmodels "mwork_backend/internal/models/chat"
	"mwork_backend/test/helpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestChat_DialogAndMessageFlow - проверяет E2E "золотой путь" для чата
func TestChat_DialogAndMessageFlow(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем Модель (User A)
	modelToken, modelUser, _ := helpers.CreateAndLoginModel(t, ts, tx)
	// Создаем Работодателя (User B)
	employerToken, employerUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// --- 2. Модель (А) создает диалог с Работодателем (Б) ---
	// Роут: POST /api/v1/dialogs
	createDialogBody := map[string]interface{}{
		"participant_ids": []string{employerUser.ID},
		"is_group":        false,
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", modelToken, createDialogBody)

	// 3. Проверка: Диалог создан
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, employerUser.ID, "Ответ должен содержать ID участника")

	// Парсим ID диалога
	var dialog chatmodels.Dialog
	err := json.Unmarshal([]byte(bodyStr), &dialog)
	assert.NoError(t, err)
	dialogID := dialog.ID
	assert.NotEmpty(t, dialogID, "Dialog ID не должен быть пустым")
	t.Logf("ЧАТ: Диалог создан (201). ID: %s", dialogID)

	// --- 4. Модель (А) отправляет сообщение в этот диалог ---
	// Роут: POST /api/v1/messages
	sendMessageBody := map[string]interface{}{
		"dialog_id": dialogID,
		"content":   "Привет! Это тестовое сообщение от Модели.",
	}
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/messages", modelToken, sendMessageBody)

	// 5. Проверка: Сообщение создано
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Привет! Это тестовое сообщение")

	// Парсим ID сообщения
	var message chatmodels.Message
	err = json.Unmarshal([]byte(bodyStr), &message)
	assert.NoError(t, err)
	messageID := message.ID
	assert.NotEmpty(t, messageID)
	t.Logf("ЧАТ: Сообщение отправлено (201). ID: %s", messageID)

	// --- 6. Работодатель (Б) получает свой список диалогов ---
	// Роут: GET /api/v1/dialogs
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/dialogs", employerToken, nil)

	// 7. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, dialogID, "Список диалогов должен содержать новый диалог")
	assert.Contains(t, bodyStr, "Привет! Это тестовое сообщение", "LastMessage должен обновиться")
	t.Logf("ЧАТ: Работодатель получил список диалогов (200) - Успешно.")

	// --- 8. Работодатель (Б) получает сообщения из этого диалога ---
	// Роут: GET /api/v1/dialogs/:dialogID/messages
	messagesURL := "/api/v1/dialogs/" + dialogID + "/messages"
	res, bodyStr = ts.SendRequest(t, "GET", messagesURL, employerToken, nil)

	// 9. Проверка:
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, messageID, "Список сообщений должен содержать новое сообщение")
	assert.Contains(t, bodyStr, modelUser.ID, "Сообщение должно содержать ID отправителя (Модели)")
	t.Logf("ЧАТ: Работодатель получил сообщения (200) - Успешно.")
}

// TestChat_Security - проверяет, что посторонний юзер не может читать чужие чаты
func TestChat_Security(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Участники диалога
	tokenA, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	_, userB, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	// Посторонний (Хакер)
	tokenC, _ := helpers.CreateAndLoginUser(t, ts, tx, "Hacker", "hacker@test.com", "pass123", models.UserRoleModel)

	// 2. Создаем диалог между A и Б
	createDialogBody := map[string]interface{}{"participant_ids": []string{userB.ID}}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", tokenA, createDialogBody)
	var dialog chatmodels.Dialog
	json.Unmarshal([]byte(bodyStr), &dialog)
	dialogID := dialog.ID

	// 3. А отправляет секретное сообщение
	sendMessageBody := map[string]interface{}{"dialog_id": dialogID, "content": "Секретное сообщение для Работодателя"}
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/messages", tokenA, sendMessageBody)
	var message chatmodels.Message
	json.Unmarshal([]byte(bodyStr), &message)
	messageID := message.ID

	// --- 4. Тесты безопасности ---

	// 4.1. Действие: Хакер (С) пытается получить список сообщений диалога (А-Б)
	messagesURL := "/api/v1/dialogs/" + dialogID + "/messages"
	res, _ = ts.SendRequest(t, "GET", messagesURL, tokenC, nil)

	// 4.2. Проверка: (403 Forbidden или 404 Not Found)
	// (Сервис не должен разрешать доступ, т.к. юзер С не участник)
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Чат): Хакер не может читать чужой диалог (%d) - Успешно.", res.StatusCode)

	// 4.3. Действие: Хакер (С) пытается получить конкретное сообщение
	messageURL := "/api/v1/messages/" + messageID
	res, _ = ts.SendRequest(t, "GET", messageURL, tokenC, nil)

	// 4.4. Проверка: (403 Forbidden или 404 Not Found)
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, res.StatusCode)
	t.Logf("БЕЗОПАСНОСТЬ (Чат): Хакер не может читать чужое сообщение (%d) - Успешно.", res.StatusCode)

	// 4.5. Действие: Хакер (С) получает свой (пустой) список диалогов
	res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/dialogs", tokenC, nil)

	// 4.6. Проверка: (200 OK, но нет чужого диалога)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotContains(t, bodyStr, dialogID, "Хакер не должен видеть чужой ID диалога")
	assert.NotContains(t, bodyStr, "Секретное сообщение", "Хакер не должен видеть чужое LastMessage")
	t.Logf("БЕЗОПАСНОСТЬ (Чат): Хакер не видит чужой диалог в своем списке (200) - Успешно.")
}

// TestChat_GroupDialog - проверяет создание группового чата
func TestChat_GroupDialog(t *testing.T) {
	t.Parallel() // ✅ Параллельный запуск

	// 1. Подготовка
	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем нескольких пользователей
	creatorToken, _, _ := helpers.CreateAndLoginEmployer(t, ts, tx)
	modelToken1, modelUser1, _ := helpers.CreateAndLoginModel(t, ts, tx)
	modelToken2, modelUser2, _ := helpers.CreateAndLoginModel(t, ts, tx)

	// 2. Создаем групповой диалог
	createGroupBody := map[string]interface{}{
		"participant_ids": []string{modelUser1.ID, modelUser2.ID},
		"is_group":        true,
		"group_name":      "Тестовая группа кастинга",
	}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", creatorToken, createGroupBody)

	// 3. Проверка: Групповой диалог создан
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Contains(t, bodyStr, "Тестовая группа кастинга")

	var groupDialog chatmodels.Dialog
	err := json.Unmarshal([]byte(bodyStr), &groupDialog)
	assert.NoError(t, err)
	assert.True(t, groupDialog.IsGroup)
	// Проверяем наличие группового названия (может быть в разных полях в зависимости от структуры)
	t.Logf("ЧАТ: Групповой диалог создан (201). ID: %s", groupDialog.ID)

	// 4. Проверяем, что все участники видят диалог
	participants := []struct {
		token string
		name  string
	}{
		{creatorToken, "Создатель"},
		{modelToken1, "Модель 1"},
		{modelToken2, "Модель 2"},
	}

	for _, p := range participants {
		res, bodyStr = ts.SendRequest(t, "GET", "/api/v1/dialogs", p.token, nil)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, bodyStr, groupDialog.ID, "%s должен видеть групповой диалог", p.name)
		t.Logf("ЧАТ: %s видит групповой диалог - Успешно.", p.name)
	}
}

// TestChat_MessageReactions - проверяет реакции на сообщения
func TestChat_MessageReactions(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей
	user1Token, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	user2Token, user2, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Создаем диалог
	createDialogBody := map[string]interface{}{"participant_ids": []string{user2.ID}}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", user1Token, createDialogBody)
	var dialog chatmodels.Dialog
	json.Unmarshal([]byte(bodyStr), &dialog)

	// Отправляем сообщение
	sendMessageBody := map[string]interface{}{"dialog_id": dialog.ID, "content": "Тестовое сообщение"}
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/messages", user1Token, sendMessageBody)
	var message chatmodels.Message
	json.Unmarshal([]byte(bodyStr), &message)

	// Добавляем реакцию
	reactionBody := map[string]interface{}{"reaction": "👍"}
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/messages/"+message.ID+"/reactions", user2Token, reactionBody)
	assert.Equal(t, http.StatusCreated, res.StatusCode)
	t.Logf("ЧАТ: Реакция добавлена - Успешно.")
}

// TestChat_MessageEditing - проверяет редактирование сообщений
func TestChat_MessageEditing(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей
	userToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	_, otherUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Создаем диалог
	createDialogBody := map[string]interface{}{"participant_ids": []string{otherUser.ID}}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", userToken, createDialogBody)
	var dialog chatmodels.Dialog
	json.Unmarshal([]byte(bodyStr), &dialog)

	// Отправляем сообщение
	sendMessageBody := map[string]interface{}{"dialog_id": dialog.ID, "content": "Оригинальное сообщение"}
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/messages", userToken, sendMessageBody)
	var message chatmodels.Message
	json.Unmarshal([]byte(bodyStr), &message)

	// Редактируем сообщение
	editBody := map[string]interface{}{"content": "Отредактированное сообщение"}
	res, bodyStr = ts.SendRequest(t, "PUT", "/api/v1/messages/"+message.ID, userToken, editBody)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, bodyStr, "Отредактированное сообщение")
	t.Logf("ЧАТ: Сообщение отредактировано - Успешно.")
}

// TestChat_MessageDeletion - проверяет удаление сообщений
func TestChat_MessageDeletion(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей
	userToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	_, otherUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Создаем диалог
	createDialogBody := map[string]interface{}{"participant_ids": []string{otherUser.ID}}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", userToken, createDialogBody)
	var dialog chatmodels.Dialog
	json.Unmarshal([]byte(bodyStr), &dialog)

	// Отправляем сообщение
	sendMessageBody := map[string]interface{}{"dialog_id": dialog.ID, "content": "Сообщение для удаления"}
	res, bodyStr = ts.SendRequest(t, "POST", "/api/v1/messages", userToken, sendMessageBody)
	var message chatmodels.Message
	json.Unmarshal([]byte(bodyStr), &message)

	// Удаляем сообщение
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/messages/"+message.ID, userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	t.Logf("ЧАТ: Сообщение удалено - Успешно.")
}

// TestChat_DialogDeletion - проверяет удаление диалогов
func TestChat_DialogDeletion(t *testing.T) {
	t.Parallel()

	ts := GetTestServer(t)
	tx := ts.BeginTransaction(t)
	defer ts.RollbackTransaction(t, tx)

	// Создаем пользователей
	userToken, _, _ := helpers.CreateAndLoginModel(t, ts, tx)
	_, otherUser, _ := helpers.CreateAndLoginEmployer(t, ts, tx)

	// Создаем диалог
	createDialogBody := map[string]interface{}{"participant_ids": []string{otherUser.ID}}
	res, bodyStr := ts.SendRequest(t, "POST", "/api/v1/dialogs", userToken, createDialogBody)
	var dialog chatmodels.Dialog
	json.Unmarshal([]byte(bodyStr), &dialog)

	// Удаляем диалог
	res, bodyStr = ts.SendRequest(t, "DELETE", "/api/v1/dialogs/"+dialog.ID, userToken, nil)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	t.Logf("ЧАТ: Диалог удален - Успешно.")
}

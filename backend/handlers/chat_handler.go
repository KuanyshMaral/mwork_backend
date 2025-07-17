package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	_ "mwork_front_fn/backend/dto"
	_ "mwork_front_fn/backend/models"
	_ "mwork_front_fn/backend/models/chat"
	serviceChat "mwork_front_fn/backend/services/chat"
)

// ChatHandler — единый handler для всех функций модуля чатов
type ChatHandler struct {
	Chat         *serviceChat.ChatService
	Attachments  *serviceChat.AttachmentService
	Reactions    *serviceChat.ReactionService
	ReadReceipts *serviceChat.ReadReceiptService
}

// NewChatHandler инициализирует handler
func NewChatHandler(
	chat *serviceChat.ChatService,
	attachments *serviceChat.AttachmentService,
	reactions *serviceChat.ReactionService,
	readReceipts *serviceChat.ReadReceiptService,
) *ChatHandler {
	// Встраиваем зависимость во внутренний сервис
	chat.Attachments = attachments

	return &ChatHandler{
		Chat:         chat,
		Attachments:  attachments,
		Reactions:    reactions,
		ReadReceipts: readReceipts,
	}
}

// CreateDialog godoc
// @Summary Создать новый диалог (личный или групповой)
// @Tags Chat
// @Accept json
// @Produce json
// @Param input body dto.CreateDialogInput true "Информация о диалоге"
// @Success 201 {object} chat.Dialog
// @Failure 400,500 {object} models.ErrorResponse
// @Router /chat/dialogs [post]
func (h *ChatHandler) CreateDialog(c *gin.Context) {
	var input serviceChat.CreateDialogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dialog, err := h.Chat.CreateDialog(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dialog)
}

// SendMessage godoc
// @Summary Отправить сообщение в диалог
// @Tags Chat
// @Accept json
// @Produce json
// @Param input body dto.SendMessageInput true "Тело сообщения"
// @Success 201 {object} chat.Message
// @Failure 400,500 {object} models.ErrorResponse
// @Router /chat/messages/send [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	var input serviceChat.SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.Chat.SendMessage(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

// GetMessages godoc
// @Summary Получить сообщения из диалога
// @Tags Chat
// @Produce json
// @Param id path string true "ID диалога"
// @Param user_id query string true "ID пользователя"
// @Success 200 {array} chat.Message
// @Failure 403,500 {object} models.ErrorResponse
// @Router /chat/dialogs/{id}/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	dialogID := c.Param("id")
	userID := c.Query("user_id") // временно, потом из JWT

	messages, err := h.Chat.GetMessages(userID, dialogID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// MarkAllAsRead godoc
// @Summary Отметить все сообщения как прочитанные
// @Tags Chat
// @Param id path string true "ID диалога"
// @Param user_id query string true "ID пользователя"
// @Success 204
// @Failure 500 {object} models.ErrorResponse
// @Router /chat/dialogs/{id}/read [post]
func (h *ChatHandler) MarkAllAsRead(c *gin.Context) {
	dialogID := c.Param("id")
	userID := c.Query("user_id") // временно

	err := h.Chat.MarkAllAsRead(userID, dialogID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDialogFiles godoc
// @Summary Получить вложения в диалоге
// @Tags Chat
// @Produce json
// @Param id path string true "ID диалога"
// @Param type query string false "Фильтр по типу (image, video, file)"
// @Success 200 {array} chat.MessageAttachment
// @Failure 500 {object} models.ErrorResponse
// @Router /chat/dialogs/{id}/files [get]
func (h *ChatHandler) GetDialogFiles(c *gin.Context) {
	dialogID := c.Param("id")
	filter := c.Query("type") // "image", "file", "video" и т.д.

	var filterType *string
	if filter != "" {
		filterType = &filter
	}

	files, err := h.Attachments.GetByDialogID(dialogID, filterType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, files)
}

type toggleReactionInput struct {
	UserID    string `json:"user_id" binding:"required"`
	MessageID string `json:"message_id" binding:"required"`
	Emoji     string `json:"emoji" binding:"required"`
}

// ToggleReaction godoc
// @Summary Поставить или убрать реакцию на сообщение
// @Tags Chat
// @Accept json
// @Produce json
// @Param input body toggleReactionInput true "Реакция"
// @Success 204
// @Failure 400,500 {object} models.ErrorResponse
// @Router /chat/reactions/toggle [post]
func (h *ChatHandler) ToggleReaction(c *gin.Context) {
	var input toggleReactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.Reactions.Toggle(input.UserID, input.MessageID, input.Emoji)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// LeaveDialog godoc
// @Summary Выйти из диалога
// @Tags Chat
// @Param id path string true "ID диалога"
// @Param user_id query string true "ID пользователя"
// @Success 204
// @Failure 500 {object} models.ErrorResponse
// @Router /chat/dialogs/{id}/leave [post]
func (h *ChatHandler) LeaveDialog(c *gin.Context) {
	dialogID := c.Param("id")
	userID := c.Query("user_id")

	err := h.Chat.LeaveDialog(userID, dialogID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetUnreadCount godoc
// @Summary Получить количество непрочитанных сообщений в диалоге
// @Tags Chat
// @Produce json
// @Param id path string true "ID диалога"
// @Param user_id query string true "ID пользователя"
// @Success 200 {object} map[string]int64
// @Failure 500 {object} models.ErrorResponse
// @Router /chat/dialogs/{id}/unread [get]
func (h *ChatHandler) GetUnreadCount(c *gin.Context) {
	dialogID := c.Param("id")
	userID := c.Query("user_id")

	count, err := h.ReadReceipts.GetUnreadCount(userID, dialogID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

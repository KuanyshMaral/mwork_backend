package old_shit

import (
	"github.com/gin-gonic/gin"
	serviceChat "mwork_backend/internal/services/chat"
	"net/http"
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

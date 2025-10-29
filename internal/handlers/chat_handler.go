package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
	// "github.com/gorilla/mux" // No longer needed
)

type ChatHandler struct {
	chatService services.ChatService
}

func NewChatHandler(chatService services.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// RegisterRoutes регистрирует все маршруты для чата, используя GIN
func (h *ChatHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Мы регистрируем маршруты непосредственно в группе /api/v1 (которая передается как 'router')
	// Mux: router.HandleFunc("/dialogs", h.CreateDialog).Methods(http.MethodPost)
	// Gin: router.POST("/dialogs", h.CreateDialog)

	// Dialog routes
	router.POST("/dialogs", h.CreateDialog)
	router.POST("/dialogs/casting", h.CreateCastingDialog)
	router.GET("/dialogs", h.GetUserDialogs)
	router.GET("/dialogs/:dialogID", h.GetDialog)
	router.PUT("/dialogs/:dialogID", h.UpdateDialog)
	router.DELETE("/dialogs/:dialogID", h.DeleteDialog)
	router.POST("/dialogs/:dialogID/leave", h.LeaveDialog)
	router.GET("/dialogs/between/:user1ID/:user2ID", h.GetDialogBetweenUsers)

	// Participant routes
	router.POST("/dialogs/:dialogID/participants", h.AddParticipants)
	router.DELETE("/dialogs/:dialogID/participants/:userID", h.RemoveParticipant)
	router.PUT("/dialogs/:dialogID/participants/:userID/role", h.UpdateParticipantRole)
	router.POST("/dialogs/:dialogID/mute", h.MuteDialog)
	router.POST("/dialogs/:dialogID/last-seen", h.UpdateLastSeen)
	router.POST("/dialogs/:dialogID/typing", h.SetTyping)

	// Message routes
	router.POST("/messages", h.SendMessage)
	router.POST("/messages/attachments", h.SendMessageWithAttachments)
	router.GET("/messages/:messageID", h.GetMessage)
	router.PUT("/messages/:messageID", h.UpdateMessage)
	router.DELETE("/messages/:messageID", h.DeleteMessage)
	router.POST("/messages/forward", h.ForwardMessage)
	router.GET("/dialogs/:dialogID/messages", h.GetMessages)
	router.GET("/dialogs/:dialogID/search", h.SearchMessages)

	// Attachment routes
	router.POST("/attachments/upload", h.UploadAttachment)
	router.GET("/messages/:messageID/attachments", h.GetMessageAttachments)
	router.GET("/dialogs/:dialogID/attachments", h.GetDialogAttachments)
	router.DELETE("/attachments/:attachmentID", h.DeleteAttachment)

	// Reaction routes
	router.POST("/messages/:messageID/reactions", h.AddReaction)
	router.DELETE("/messages/:messageID/reactions", h.RemoveReaction)
	router.GET("/messages/:messageID/reactions", h.GetMessageReactions)

	// Read receipts routes
	router.POST("/dialogs/:dialogID/read", h.MarkMessagesAsRead)
	router.GET("/dialogs/:dialogID/unread-count", h.GetUnreadCount)
	router.GET("/messages/:messageID/read-receipts", h.GetReadReceipts)

	// Combined routes
	router.GET("/dialogs/:dialogID/with-messages", h.GetDialogWithMessages)

	// Admin routes (TODO: Защитите эти маршруты с помощью Admin middleware)
	router.GET("/admin/dialogs", h.GetAllDialogs)
	router.GET("/admin/stats", h.GetChatStats)
	router.POST("/admin/clean", h.CleanOldMessages)
	router.DELETE("/admin/users/:userID/messages", h.DeleteUserMessages)
}

// Dialog handlers

func (h *ChatHandler) CreateDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.CreateDialogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	dialog, err := h.chatService.CreateDialog(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dialog)
}

func (h *ChatHandler) CreateCastingDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		CastingID  string `json:"casting_id"`
		EmployerID string `json:"employer_id"`
		ModelID    string `json:"model_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	dialog, err := h.chatService.CreateCastingDialog(req.CastingID, req.EmployerID, req.ModelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dialog)
}

func (h *ChatHandler) GetDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	dialog, err := h.chatService.GetDialog(dialogID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dialog)
}

func (h *ChatHandler) GetUserDialogs(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogs, err := h.chatService.GetUserDialogs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dialogs)
}

func (h *ChatHandler) GetDialogBetweenUsers(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user1ID := c.Param("user1ID")
	user2ID := c.Param("user2ID")

	dialog, err := h.chatService.GetDialogBetweenUsers(user1ID, user2ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dialog)
}

func (h *ChatHandler) UpdateDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	var req dto.UpdateDialogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.UpdateDialog(userID, dialogID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dialog updated successfully"})
}

func (h *ChatHandler) DeleteDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	if err := h.chatService.DeleteDialog(userID, dialogID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dialog deleted successfully"})
}

func (h *ChatHandler) LeaveDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	if err := h.chatService.LeaveDialog(userID, dialogID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left dialog successfully"})
}

// Participant handlers

func (h *ChatHandler) AddParticipants(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	var req struct {
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.AddParticipants(userID, dialogID, req.ParticipantIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participants added successfully"})
}

func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")
	targetUserID := c.Param("userID")

	if err := h.chatService.RemoveParticipant(userID, dialogID, targetUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant removed successfully"})
}

func (h *ChatHandler) UpdateParticipantRole(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")
	targetUserID := c.Param("userID")

	var req struct {
		Role string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.UpdateParticipantRole(userID, dialogID, targetUserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant role updated successfully"})
}

func (h *ChatHandler) MuteDialog(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	var req struct {
		Muted bool `json:"muted"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.MuteDialog(userID, dialogID, req.Muted); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dialog mute status updated"})
}

func (h *ChatHandler) UpdateLastSeen(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	if err := h.chatService.UpdateLastSeen(userID, dialogID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "last seen updated"})
}

func (h *ChatHandler) SetTyping(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	var req struct {
		Typing bool `json:"typing"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.SetTyping(userID, dialogID, req.Typing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "typing status updated"})
}

// Message handlers

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	message, err := h.chatService.SendMessage(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) SendMessageWithAttachments(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 50 MB лимит
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	// Parse message data
	var req dto.SendMessageRequest
	messageData := c.Request.FormValue("message")
	if err := json.Unmarshal([]byte(messageData), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message data"})
		return
	}

	// Get files
	files := c.Request.MultipartForm.File["files"]

	message, err := h.chatService.SendMessageWithAttachments(userID, &req, files)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")
	criteria := parseMessageCriteria(c)

	messages, err := h.chatService.GetMessages(dialogID, userID, criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *ChatHandler) GetMessage(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	message, err := h.chatService.GetMessage(messageID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message)
}

func (h *ChatHandler) UpdateMessage(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	var req dto.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.UpdateMessage(userID, messageID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message updated successfully"})
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	if err := h.chatService.DeleteMessage(userID, messageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted successfully"})
}

func (h *ChatHandler) ForwardMessage(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.ForwardMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	message, err := h.chatService.ForwardMessage(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) SearchMessages(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")
	query := c.Query("q")

	messages, err := h.chatService.SearchMessages(userID, dialogID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// Attachment handlers

func (h *ChatHandler) UploadAttachment(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Используем Gin-метод FormFile
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}

	attachment, err := h.chatService.UploadAttachment(userID, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, attachment)
}

func (h *ChatHandler) GetMessageAttachments(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	attachments, err := h.chatService.GetMessageAttachments(messageID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, attachments)
}

func (h *ChatHandler) GetDialogAttachments(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	attachments, err := h.chatService.GetDialogAttachments(dialogID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, attachments)
}

func (h *ChatHandler) DeleteAttachment(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	attachmentID := c.Param("attachmentID")

	if err := h.chatService.DeleteAttachment(userID, attachmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "attachment deleted successfully"})
}

// Reaction handlers

func (h *ChatHandler) AddReaction(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	var req struct {
		Emoji string `json:"emoji"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.AddReaction(userID, messageID, req.Emoji); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "reaction added successfully"})
}

func (h *ChatHandler) RemoveReaction(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	if err := h.chatService.RemoveReaction(userID, messageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reaction removed successfully"})
}

func (h *ChatHandler) GetMessageReactions(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	reactions, err := h.chatService.GetMessageReactions(messageID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reactions)
}

// Read receipts handlers

func (h *ChatHandler) MarkMessagesAsRead(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	if err := h.chatService.MarkMessagesAsRead(userID, dialogID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "messages marked as read"})
}

func (h *ChatHandler) GetUnreadCount(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	count, err := h.chatService.GetUnreadCount(dialogID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

func (h *ChatHandler) GetReadReceipts(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID := c.Param("messageID")

	receipts, err := h.chatService.GetReadReceipts(messageID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, receipts)
}

// Combined handlers

func (h *ChatHandler) GetDialogWithMessages(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dialogID := c.Param("dialogID")

	criteria := parseMessageCriteria(c)

	result, err := h.chatService.GetDialogWithMessages(dialogID, userID, criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Admin handlers

func (h *ChatHandler) GetAllDialogs(c *gin.Context) {
	criteria := parseDialogCriteria(c)

	dialogs, err := h.chatService.GetAllDialogs(criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dialogs)
}

func (h *ChatHandler) GetChatStats(c *gin.Context) {
	stats, err := h.chatService.GetChatStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ChatHandler) CleanOldMessages(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.chatService.CleanOldMessages(req.Days); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "old messages cleaned successfully"})
}

func (h *ChatHandler) DeleteUserMessages(c *gin.Context) {
	adminID := getUserIDFromContext(c)
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID := c.Param("userID")

	if err := h.chatService.DeleteUserMessages(adminID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user messages deleted successfully"})
}

// Helper functions

func getUserIDFromContext(c *gin.Context) string {
	// Получаем userID из контекста Gin (обычно устанавливается middleware аутентификации)
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

// Вспомогательная функция для парсинга query-параметров (как в AnalyticsHandler)
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func parseMessageCriteria(c *gin.Context) dto.MessageCriteria {
	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	return dto.MessageCriteria{
		Limit:  limit,
		Offset: offset,
	}
}

func parseDialogCriteria(c *gin.Context) dto.DialogCriteria {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)

	return dto.DialogCriteria{
		Page:     page,
		PageSize: pageSize,
	}
}

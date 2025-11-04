package handlers

import (
	"encoding/json"
	"net/http"
	// "strconv" // <-- No longer needed

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/validator"
	"mwork_backend/pkg/apperrors"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	*BaseHandler
	chatService services.ChatService
}

func NewChatHandler(base *BaseHandler, chatService services.ChatService) *ChatHandler {
	return &ChatHandler{
		BaseHandler: base,
		chatService: chatService,
	}
}

// RegisterRoutes - ОЧИЩЕНО
func (h *ChatHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.Use(middleware.AuthMiddleware())

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
	router.DELETE("/dialogs/:dialogID/participants/:userId", h.RemoveParticipant)
	router.PUT("/dialogs/:dialogID/participants/:userId/role", h.UpdateParticipantRole)
	router.POST("/dialogs/:dialogID/mute", h.MuteDialog)
	router.POST("/dialogs/:dialogID/last-seen", h.UpdateLastSeen)
	router.POST("/dialogs/:dialogID/typing", h.SetTyping)

	// Message routes
	router.POST("/messages", h.SendMessage)
	router.POST("/messages/attachments", h.SendMessageWithAttachments) // <-- ОСТАВЛЕН
	router.GET("/messages/:messageID", h.GetMessage)
	router.PUT("/messages/:messageID", h.UpdateMessage)
	router.DELETE("/messages/:messageID", h.DeleteMessage)
	router.POST("/messages/forward", h.ForwardMessage)
	router.GET("/dialogs/:dialogID/messages", h.GetMessages)
	router.GET("/dialogs/:dialogID/search", h.SearchMessages)

	// ▼▼▼ УДАЛЕНО: Маршруты Attachment ▼▼▼
	// router.POST("/attachments/upload", h.UploadAttachment)
	// router.GET("/messages/:messageID/attachments", h.GetMessageAttachments)
	// router.GET("/dialogs/:dialogID/attachments", h.GetDialogAttachments)
	// router.DELETE("/attachments/:attachmentID", h.DeleteAttachment)
	// ▲▲▲ УДАЛЕНО ▲▲▲

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

	// Admin routes
	router.GET("/admin/dialogs", h.GetAllDialogs)
	router.GET("/admin/stats", h.GetChatStats)
	router.POST("/admin/clean", h.CleanOldMessages)
	router.DELETE("/admin/users/:userId/messages", h.DeleteUserMessages)
}

// --- Dialog handlers (Обновлены с Context) ---

func (h *ChatHandler) CreateDialog(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateDialogRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	dialog, err := h.chatService.CreateDialog(c.Request.Context(), h.GetDB(c), userID, &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dialog)
}

func (h *ChatHandler) CreateCastingDialog(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	var req struct {
		CastingID  string `json:"casting_id"`
		EmployerID string `json:"employer_id"`
		ModelID    string `json:"model_id"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	dialog, err := h.chatService.CreateCastingDialog(c.Request.Context(), h.GetDB(c), req.CastingID, req.EmployerID, req.ModelID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dialog)
}

func (h *ChatHandler) GetDialog(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	// ✅ DB + Context
	dialog, err := h.chatService.GetDialog(c.Request.Context(), h.GetDB(c), dialogID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dialog)
}

func (h *ChatHandler) GetUserDialogs(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB + Context
	dialogs, err := h.chatService.GetUserDialogs(c.Request.Context(), h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dialogs)
}

func (h *ChatHandler) GetDialogBetweenUsers(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	user1ID := c.Param("user1ID")
	user2ID := c.Param("user2ID")

	// ✅ DB + Context
	dialog, err := h.chatService.GetDialogBetweenUsers(c.Request.Context(), h.GetDB(c), user1ID, user2ID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dialog)
}

func (h *ChatHandler) UpdateDialog(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	var req dto.UpdateDialogRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.UpdateDialog(c.Request.Context(), h.GetDB(c), userID, dialogID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dialog updated successfully"})
}

func (h *ChatHandler) DeleteDialog(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	// ✅ DB + Context
	if err := h.chatService.DeleteDialog(c.Request.Context(), h.GetDB(c), userID, dialogID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dialog deleted successfully"})
}

func (h *ChatHandler) LeaveDialog(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	// ✅ DB + Context
	if err := h.chatService.LeaveDialog(c.Request.Context(), h.GetDB(c), userID, dialogID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left dialog successfully"})
}

// --- Participant handlers (Обновлены с Context) ---

func (h *ChatHandler) AddParticipants(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	var req struct {
		ParticipantIDs []string `json:"participant_ids"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.AddParticipants(c.Request.Context(), h.GetDB(c), userID, dialogID, req.ParticipantIDs); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participants added successfully"})
}

func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")
	targetUserID := c.Param("userID")

	// ✅ DB + Context
	if err := h.chatService.RemoveParticipant(c.Request.Context(), h.GetDB(c), userID, dialogID, targetUserID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant removed successfully"})
}

func (h *ChatHandler) UpdateParticipantRole(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")
	targetUserID := c.Param("userID")

	var req struct {
		Role string `json:"role"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.UpdateParticipantRole(c.Request.Context(), h.GetDB(c), userID, dialogID, targetUserID, req.Role); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant role updated successfully"})
}

func (h *ChatHandler) MuteDialog(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	var req struct {
		Muted bool `json:"muted"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.MuteDialog(c.Request.Context(), h.GetDB(c), userID, dialogID, req.Muted); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dialog mute status updated"})
}

func (h *ChatHandler) UpdateLastSeen(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	// ✅ DB + Context
	if err := h.chatService.UpdateLastSeen(c.Request.Context(), h.GetDB(c), userID, dialogID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "last seen updated"})
}

func (h *ChatHandler) SetTyping(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	var req struct {
		Typing bool `json:"typing"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.SetTyping(c.Request.Context(), h.GetDB(c), userID, dialogID, req.Typing); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "typing status updated"})
}

// --- Message handlers (Обновлены с Context) ---

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.SendMessageRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	message, err := h.chatService.SendMessage(c.Request.Context(), h.GetDB(c), userID, &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) SendMessageWithAttachments(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		h.HandleServiceError(c, apperrors.NewBadRequestError("failed to parse multipart form: "+err.Error()))
		return
	}

	var req dto.SendMessageRequest
	messageData := c.Request.FormValue("message")
	if err := json.Unmarshal([]byte(messageData), &req); err != nil {
		h.HandleServiceError(c, apperrors.NewBadRequestError("invalid message data: "+err.Error()))
		return
	}

	if err := h.validator.Validate(&req); err != nil {
		if vErr, ok := err.(*validator.ValidationError); ok {
			apperrors.HandleError(c, apperrors.ValidationError(vErr.Errors))
		} else {
			apperrors.HandleError(c, apperrors.InternalError(err))
		}
		return
	}

	files := c.Request.MultipartForm.File["files"]

	// ✅ DB + Context
	message, err := h.chatService.SendMessageWithAttachments(c.Request.Context(), h.GetDB(c), userID, &req, files)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")
	criteria := parseMessageCriteria(c)

	// ✅ DB + Context
	messages, err := h.chatService.GetMessages(c.Request.Context(), h.GetDB(c), dialogID, userID, criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *ChatHandler) GetMessage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	// ✅ DB + Context
	message, err := h.chatService.GetMessage(c.Request.Context(), h.GetDB(c), messageID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, message)
}

func (h *ChatHandler) UpdateMessage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	var req dto.UpdateMessageRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.UpdateMessage(c.Request.Context(), h.GetDB(c), userID, messageID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message updated successfully"})
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	// ✅ DB + Context
	if err := h.chatService.DeleteMessage(c.Request.Context(), h.GetDB(c), userID, messageID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted successfully"})
}

func (h *ChatHandler) ForwardMessage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.ForwardMessageRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	message, err := h.chatService.ForwardMessage(c.Request.Context(), h.GetDB(c), userID, &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *ChatHandler) SearchMessages(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")
	query := c.Query("q")

	// ✅ DB + Context
	messages, err := h.chatService.SearchMessages(c.Request.Context(), h.GetDB(c), userID, dialogID, query)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, messages)
}

// --- ▼▼▼ УДАЛЕНО: Attachment handlers ▼▼▼ ---
//
// func (h *ChatHandler) UploadAttachment(c *gin.Context) { ... }
// func (h *ChatHandler) GetMessageAttachments(c *gin.Context) { ... }
// func (h *ChatHandler) GetDialogAttachments(c *gin.Context) { ... }
// func (h *ChatHandler) DeleteAttachment(c *gin.Context) { ... }
//
// --- ▲▲▲ УДАЛЕНО ▲▲▲ ---

// --- Reaction handlers (Обновлены с Context) ---

func (h *ChatHandler) AddReaction(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	var req struct {
		Emoji string `json:"emoji"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.AddReaction(c.Request.Context(), h.GetDB(c), userID, messageID, req.Emoji); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "reaction added successfully"})
}

func (h *ChatHandler) RemoveReaction(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	// ✅ DB + Context
	if err := h.chatService.RemoveReaction(c.Request.Context(), h.GetDB(c), userID, messageID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reaction removed successfully"})
}

func (h *ChatHandler) GetMessageReactions(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	// ✅ DB + Context
	reactions, err := h.chatService.GetMessageReactions(c.Request.Context(), h.GetDB(c), messageID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, reactions)
}

// --- Read receipts handlers (Обновлены с Context) ---

func (h *ChatHandler) MarkMessagesAsRead(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	// ✅ DB + Context
	if err := h.chatService.MarkMessagesAsRead(c.Request.Context(), h.GetDB(c), userID, dialogID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "messages marked as read"})
}

func (h *ChatHandler) GetUnreadCount(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	// ✅ DB + Context
	count, err := h.chatService.GetUnreadCount(c.Request.Context(), h.GetDB(c), dialogID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

func (h *ChatHandler) GetReadReceipts(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	// ✅ DB + Context
	receipts, err := h.chatService.GetReadReceipts(c.Request.Context(), h.GetDB(c), messageID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, receipts)
}

// --- Combined handlers (Обновлены с Context) ---

func (h *ChatHandler) GetDialogWithMessages(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	criteria := parseMessageCriteria(c)

	// ✅ DB + Context
	result, err := h.chatService.GetDialogWithMessages(c.Request.Context(), h.GetDB(c), dialogID, userID, criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// --- Admin handlers (Обновлены с Context) ---

func (h *ChatHandler) GetAllDialogs(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	criteria := parseDialogCriteria(c)

	// ✅ DB + Context
	dialogs, err := h.chatService.GetAllDialogs(c.Request.Context(), h.GetDB(c), criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dialogs)
}

func (h *ChatHandler) GetChatStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB + Context
	stats, err := h.chatService.GetChatStats(c.Request.Context(), h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ChatHandler) CleanOldMessages(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	var req struct {
		Days int `json:"days"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.chatService.CleanOldMessages(c.Request.Context(), h.GetDB(c), req.Days); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "old messages cleaned successfully"})
}

func (h *ChatHandler) DeleteUserMessages(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	userID := c.Param("userID")
	dialogID := c.Query("dialog_id")
	if dialogID == "" {
		apperrors.HandleError(c, apperrors.NewBadRequestError("dialog_id query parameter is required"))
		return
	}

	// ✅ DB + Context
	if err := h.chatService.DeleteUserMessages(c.Request.Context(), h.GetDB(c), adminID, dialogID, userID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user messages deleted successfully"})
}

// --- Helper functions ---

// (Предполагаем, что 'ParseQueryInt' и 'ParsePagination' находятся в BaseHandler)

func parseMessageCriteria(c *gin.Context) dto.MessageCriteria {
	limit := ParseQueryInt(c, "limit", 50)
	offset := ParseQueryInt(c, "offset", 0)

	return dto.MessageCriteria{
		Limit:  limit,
		Offset: offset,
	}
}

func parseDialogCriteria(c *gin.Context) dto.DialogCriteria {
	page, pageSize := ParsePagination(c)

	return dto.DialogCriteria{
		Page:     page,
		PageSize: pageSize,
	}
}

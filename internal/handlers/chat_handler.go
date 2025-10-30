package handlers

import (
	"encoding/json"
	"net/http"
	// "strconv" // <-- No longer needed

	"mwork_backend/internal/appErrors" // <-- Added import
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/validator" // <-- Added import (for manual validation)

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	*BaseHandler // <-- 1. Embed BaseHandler
	chatService  services.ChatService
}

// 2. Update the constructor
func NewChatHandler(base *BaseHandler, chatService services.ChatService) *ChatHandler {
	return &ChatHandler{
		BaseHandler: base, // <-- 3. Assign it
		chatService: chatService,
	}
}

// RegisterRoutes (no changes, but middleware imports are now required)
func (h *ChatHandler) RegisterRoutes(router *gin.RouterGroup) {
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

	// Admin routes (now protected by GetAndAuthorizeUserID)
	router.GET("/admin/dialogs", h.GetAllDialogs)
	router.GET("/admin/stats", h.GetChatStats)
	router.POST("/admin/clean", h.CleanOldMessages)
	router.DELETE("/admin/users/:userID/messages", h.DeleteUserMessages)
}

// --- Dialog handlers ---

func (h *ChatHandler) CreateDialog(c *gin.Context) {
	// 4. Use GetAndAuthorizeUserID
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateDialogRequest
	// 5. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	dialog, err := h.chatService.CreateDialog(userID, &req)
	if err != nil {
		// 6. Use HandleServiceError
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

	dialog, err := h.chatService.CreateCastingDialog(req.CastingID, req.EmployerID, req.ModelID)
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

	dialog, err := h.chatService.GetDialog(dialogID, userID)
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

	dialogs, err := h.chatService.GetUserDialogs(userID)
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

	dialog, err := h.chatService.GetDialogBetweenUsers(user1ID, user2ID)
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

	if err := h.chatService.UpdateDialog(userID, dialogID, &req); err != nil {
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

	if err := h.chatService.DeleteDialog(userID, dialogID); err != nil {
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

	if err := h.chatService.LeaveDialog(userID, dialogID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left dialog successfully"})
}

// --- Participant handlers ---

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

	if err := h.chatService.AddParticipants(userID, dialogID, req.ParticipantIDs); err != nil {
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

	if err := h.chatService.RemoveParticipant(userID, dialogID, targetUserID); err != nil {
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

	if err := h.chatService.UpdateParticipantRole(userID, dialogID, targetUserID, req.Role); err != nil {
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

	if err := h.chatService.MuteDialog(userID, dialogID, req.Muted); err != nil {
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

	if err := h.chatService.UpdateLastSeen(userID, dialogID); err != nil {
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

	if err := h.chatService.SetTyping(userID, dialogID, req.Typing); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "typing status updated"})
}

// --- Message handlers ---

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.SendMessageRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	message, err := h.chatService.SendMessage(userID, &req)
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

	// 50 MB лимит
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		h.HandleServiceError(c, appErrors.NewBadRequestError("failed to parse multipart form: "+err.Error()))
		return
	}

	// Parse message data
	var req dto.SendMessageRequest
	messageData := c.Request.FormValue("message")
	if err := json.Unmarshal([]byte(messageData), &req); err != nil {
		h.HandleServiceError(c, appErrors.NewBadRequestError("invalid message data: "+err.Error()))
		return
	}

	// 7. Manually call validator (since BindAndValidate_JSON won't work on a string field)
	if err := h.validator.Validate(&req); err != nil {
		if vErr, ok := err.(*validator.ValidationError); ok {
			appErrors.HandleError(c, appErrors.ValidationError(vErr.Errors))
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	// Get files
	files := c.Request.MultipartForm.File["files"]

	message, err := h.chatService.SendMessageWithAttachments(userID, &req, files)
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

	// 8. Use refactored helper
	criteria := parseMessageCriteria(c)

	messages, err := h.chatService.GetMessages(dialogID, userID, criteria)
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

	message, err := h.chatService.GetMessage(messageID, userID)
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

	if err := h.chatService.UpdateMessage(userID, messageID, &req); err != nil {
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

	if err := h.chatService.DeleteMessage(userID, messageID); err != nil {
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

	message, err := h.chatService.ForwardMessage(userID, &req)
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

	messages, err := h.chatService.SearchMessages(userID, dialogID, query)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, messages)
}

// --- Attachment handlers ---

func (h *ChatHandler) UploadAttachment(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// Используем Gin-метод FormFile
	fileHeader, err := c.FormFile("file")
	if err != nil {
		appErrors.HandleError(c, appErrors.NewBadRequestError("no file provided"))
		return
	}

	attachment, err := h.chatService.UploadAttachment(userID, fileHeader)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, attachment)
}

func (h *ChatHandler) GetMessageAttachments(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	messageID := c.Param("messageID")

	attachments, err := h.chatService.GetMessageAttachments(messageID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, attachments)
}

func (h *ChatHandler) GetDialogAttachments(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	attachments, err := h.chatService.GetDialogAttachments(dialogID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, attachments)
}

func (h *ChatHandler) DeleteAttachment(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	attachmentID := c.Param("attachmentID")

	if err := h.chatService.DeleteAttachment(userID, attachmentID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "attachment deleted successfully"})
}

// --- Reaction handlers ---

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

	if err := h.chatService.AddReaction(userID, messageID, req.Emoji); err != nil {
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

	if err := h.chatService.RemoveReaction(userID, messageID); err != nil {
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

	reactions, err := h.chatService.GetMessageReactions(messageID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, reactions)
}

// --- Read receipts handlers ---

func (h *ChatHandler) MarkMessagesAsRead(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	if err := h.chatService.MarkMessagesAsRead(userID, dialogID); err != nil {
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

	count, err := h.chatService.GetUnreadCount(dialogID, userID)
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

	receipts, err := h.chatService.GetReadReceipts(messageID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, receipts)
}

// --- Combined handlers ---

func (h *ChatHandler) GetDialogWithMessages(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	dialogID := c.Param("dialogID")

	criteria := parseMessageCriteria(c)

	result, err := h.chatService.GetDialogWithMessages(dialogID, userID, criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// --- Admin handlers ---

func (h *ChatHandler) GetAllDialogs(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	criteria := parseDialogCriteria(c)

	dialogs, err := h.chatService.GetAllDialogs(criteria)
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

	stats, err := h.chatService.GetChatStats()
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

	if err := h.chatService.CleanOldMessages(req.Days); err != nil {
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

	if err := h.chatService.DeleteUserMessages(adminID, userID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user messages deleted successfully"})
}

// --- Helper functions ---

// 9. Removed getUserIDFromContext and parseIntQuery

// 10. Updated helpers to use BaseHandler functions
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

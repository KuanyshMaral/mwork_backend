package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gorilla/mux"
)

type ChatHandler struct {
	chatService services.ChatService
}

func NewChatHandler(chatService services.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// RegisterRoutes регистрирует все маршруты для чата
func (h *ChatHandler) RegisterRoutes(router *mux.Router) {
	// Dialog routes
	router.HandleFunc("/dialogs", h.CreateDialog).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/casting", h.CreateCastingDialog).Methods(http.MethodPost)
	router.HandleFunc("/dialogs", h.GetUserDialogs).Methods(http.MethodGet)
	router.HandleFunc("/dialogs/{dialogID}", h.GetDialog).Methods(http.MethodGet)
	router.HandleFunc("/dialogs/{dialogID}", h.UpdateDialog).Methods(http.MethodPut)
	router.HandleFunc("/dialogs/{dialogID}", h.DeleteDialog).Methods(http.MethodDelete)
	router.HandleFunc("/dialogs/{dialogID}/leave", h.LeaveDialog).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/between/{user1ID}/{user2ID}", h.GetDialogBetweenUsers).Methods(http.MethodGet)

	// Participant routes
	router.HandleFunc("/dialogs/{dialogID}/participants", h.AddParticipants).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/{dialogID}/participants/{userID}", h.RemoveParticipant).Methods(http.MethodDelete)
	router.HandleFunc("/dialogs/{dialogID}/participants/{userID}/role", h.UpdateParticipantRole).Methods(http.MethodPut)
	router.HandleFunc("/dialogs/{dialogID}/mute", h.MuteDialog).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/{dialogID}/last-seen", h.UpdateLastSeen).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/{dialogID}/typing", h.SetTyping).Methods(http.MethodPost)

	// Message routes
	router.HandleFunc("/messages", h.SendMessage).Methods(http.MethodPost)
	router.HandleFunc("/messages/attachments", h.SendMessageWithAttachments).Methods(http.MethodPost)
	router.HandleFunc("/messages/{messageID}", h.GetMessage).Methods(http.MethodGet)
	router.HandleFunc("/messages/{messageID}", h.UpdateMessage).Methods(http.MethodPut)
	router.HandleFunc("/messages/{messageID}", h.DeleteMessage).Methods(http.MethodDelete)
	router.HandleFunc("/messages/forward", h.ForwardMessage).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/{dialogID}/messages", h.GetMessages).Methods(http.MethodGet)
	router.HandleFunc("/dialogs/{dialogID}/search", h.SearchMessages).Methods(http.MethodGet)

	// Attachment routes
	router.HandleFunc("/attachments/upload", h.UploadAttachment).Methods(http.MethodPost)
	router.HandleFunc("/messages/{messageID}/attachments", h.GetMessageAttachments).Methods(http.MethodGet)
	router.HandleFunc("/dialogs/{dialogID}/attachments", h.GetDialogAttachments).Methods(http.MethodGet)
	router.HandleFunc("/attachments/{attachmentID}", h.DeleteAttachment).Methods(http.MethodDelete)

	// Reaction routes
	router.HandleFunc("/messages/{messageID}/reactions", h.AddReaction).Methods(http.MethodPost)
	router.HandleFunc("/messages/{messageID}/reactions", h.RemoveReaction).Methods(http.MethodDelete)
	router.HandleFunc("/messages/{messageID}/reactions", h.GetMessageReactions).Methods(http.MethodGet)

	// Read receipts routes
	router.HandleFunc("/dialogs/{dialogID}/read", h.MarkMessagesAsRead).Methods(http.MethodPost)
	router.HandleFunc("/dialogs/{dialogID}/unread-count", h.GetUnreadCount).Methods(http.MethodGet)
	router.HandleFunc("/messages/{messageID}/read-receipts", h.GetReadReceipts).Methods(http.MethodGet)

	// Combined routes
	router.HandleFunc("/dialogs/{dialogID}/with-messages", h.GetDialogWithMessages).Methods(http.MethodGet)

	// Admin routes
	router.HandleFunc("/admin/dialogs", h.GetAllDialogs).Methods(http.MethodGet)
	router.HandleFunc("/admin/stats", h.GetChatStats).Methods(http.MethodGet)
	router.HandleFunc("/admin/clean", h.CleanOldMessages).Methods(http.MethodPost)
	router.HandleFunc("/admin/users/{userID}/messages", h.DeleteUserMessages).Methods(http.MethodDelete)
}

// Dialog handlers

func (h *ChatHandler) CreateDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateDialogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dialog, err := h.chatService.CreateDialog(userID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, dialog)
}

func (h *ChatHandler) CreateCastingDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		CastingID  string `json:"casting_id"`
		EmployerID string `json:"employer_id"`
		ModelID    string `json:"model_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dialog, err := h.chatService.CreateCastingDialog(req.CastingID, req.EmployerID, req.ModelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, dialog)
}

func (h *ChatHandler) GetDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	dialog, err := h.chatService.GetDialog(dialogID, userID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dialog)
}

func (h *ChatHandler) GetUserDialogs(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dialogs, err := h.chatService.GetUserDialogs(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dialogs)
}

func (h *ChatHandler) GetDialogBetweenUsers(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	user1ID := vars["user1ID"]
	user2ID := vars["user2ID"]

	dialog, err := h.chatService.GetDialogBetweenUsers(user1ID, user2ID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dialog)
}

func (h *ChatHandler) UpdateDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	var req dto.UpdateDialogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.UpdateDialog(userID, dialogID, &req); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "dialog updated successfully"})
}

func (h *ChatHandler) DeleteDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	if err := h.chatService.DeleteDialog(userID, dialogID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "dialog deleted successfully"})
}

func (h *ChatHandler) LeaveDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	if err := h.chatService.LeaveDialog(userID, dialogID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "left dialog successfully"})
}

// Participant handlers

func (h *ChatHandler) AddParticipants(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	var req struct {
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.AddParticipants(userID, dialogID, req.ParticipantIDs); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "participants added successfully"})
}

func (h *ChatHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]
	targetUserID := vars["userID"]

	if err := h.chatService.RemoveParticipant(userID, dialogID, targetUserID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "participant removed successfully"})
}

func (h *ChatHandler) UpdateParticipantRole(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]
	targetUserID := vars["userID"]

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.UpdateParticipantRole(userID, dialogID, targetUserID, req.Role); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "participant role updated successfully"})
}

func (h *ChatHandler) MuteDialog(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	var req struct {
		Muted bool `json:"muted"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.MuteDialog(userID, dialogID, req.Muted); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "dialog mute status updated"})
}

func (h *ChatHandler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	if err := h.chatService.UpdateLastSeen(userID, dialogID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "last seen updated"})
}

func (h *ChatHandler) SetTyping(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	var req struct {
		Typing bool `json:"typing"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.SetTyping(userID, dialogID, req.Typing); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "typing status updated"})
}

// Message handlers

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	message, err := h.chatService.SendMessage(userID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, message)
}

func (h *ChatHandler) SendMessageWithAttachments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Parse message data
	var req dto.SendMessageRequest
	if err := json.Unmarshal([]byte(r.FormValue("message")), &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid message data")
		return
	}

	// Get files
	files := r.MultipartForm.File["files"]

	message, err := h.chatService.SendMessageWithAttachments(userID, &req, files)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, message)
}

func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	criteria := parseMessageCriteria(r)

	messages, err := h.chatService.GetMessages(dialogID, userID, criteria)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	message, err := h.chatService.GetMessage(messageID, userID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, message)
}

func (h *ChatHandler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	var req dto.UpdateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.UpdateMessage(userID, messageID, &req); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "message updated successfully"})
}

func (h *ChatHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	if err := h.chatService.DeleteMessage(userID, messageID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "message deleted successfully"})
}

func (h *ChatHandler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	message, err := h.chatService.ForwardMessage(userID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, message)
}

func (h *ChatHandler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]
	query := r.URL.Query().Get("q")

	messages, err := h.chatService.SearchMessages(userID, dialogID, query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, messages)
}

// Attachment handlers

func (h *ChatHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "no file provided")
		return
	}
	defer file.Close()

	fileHeader := r.MultipartForm.File["file"][0]

	attachment, err := h.chatService.UploadAttachment(userID, fileHeader)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, attachment)
}

func (h *ChatHandler) GetMessageAttachments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	attachments, err := h.chatService.GetMessageAttachments(messageID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, attachments)
}

func (h *ChatHandler) GetDialogAttachments(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	attachments, err := h.chatService.GetDialogAttachments(dialogID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, attachments)
}

func (h *ChatHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	attachmentID := vars["attachmentID"]

	if err := h.chatService.DeleteAttachment(userID, attachmentID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "attachment deleted successfully"})
}

// Reaction handlers

func (h *ChatHandler) AddReaction(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	var req struct {
		Emoji string `json:"emoji"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.AddReaction(userID, messageID, req.Emoji); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"message": "reaction added successfully"})
}

func (h *ChatHandler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	if err := h.chatService.RemoveReaction(userID, messageID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "reaction removed successfully"})
}

func (h *ChatHandler) GetMessageReactions(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	reactions, err := h.chatService.GetMessageReactions(messageID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, reactions)
}

// Read receipts handlers

func (h *ChatHandler) MarkMessagesAsRead(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	if err := h.chatService.MarkMessagesAsRead(userID, dialogID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "messages marked as read"})
}

func (h *ChatHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	count, err := h.chatService.GetUnreadCount(dialogID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]int64{"unread_count": count})
}

func (h *ChatHandler) GetReadReceipts(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	receipts, err := h.chatService.GetReadReceipts(messageID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, receipts)
}

// Combined handlers

func (h *ChatHandler) GetDialogWithMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	dialogID := vars["dialogID"]

	criteria := parseMessageCriteria(r)

	result, err := h.chatService.GetDialogWithMessages(dialogID, userID, criteria)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Admin handlers

func (h *ChatHandler) GetAllDialogs(w http.ResponseWriter, r *http.Request) {
	criteria := parseDialogCriteria(r)

	dialogs, err := h.chatService.GetAllDialogs(criteria)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dialogs)
}

func (h *ChatHandler) GetChatStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.chatService.GetChatStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func (h *ChatHandler) CleanOldMessages(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Days int `json:"days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatService.CleanOldMessages(req.Days); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "old messages cleaned successfully"})
}

func (h *ChatHandler) DeleteUserMessages(w http.ResponseWriter, r *http.Request) {
	adminID := getUserIDFromContext(r)
	if adminID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	userID := vars["userID"]

	if err := h.chatService.DeleteUserMessages(adminID, userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "user messages deleted successfully"})
}

// Helper functions

func getUserIDFromContext(r *http.Request) string {
	// Получаем userID из контекста (обычно устанавливается middleware аутентификации)
	if userID := r.Context().Value("userID"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

func parseMessageCriteria(r *http.Request) dto.MessageCriteria {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 50 // default limit
	}

	offset, _ := strconv.Atoi(query.Get("offset"))

	return dto.MessageCriteria{
		Limit:  limit,
		Offset: offset,
	}
}

func parseDialogCriteria(r *http.Request) dto.DialogCriteria {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page == 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	if pageSize == 0 {
		pageSize = 20
	}

	return dto.DialogCriteria{
		Page:     page,
		PageSize: pageSize,
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

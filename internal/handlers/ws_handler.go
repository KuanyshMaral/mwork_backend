package handlers

import (
	serviceschat "mwork_backend/internal/services/chat"
	"mwork_backend/ws"
	"net/http"
)

type WSHandler struct {
	Manager            *ws.WebSocketManager
	ChatService        *serviceschat.ChatService
	AttachmentService  *serviceschat.AttachmentService
	ReactionService    *serviceschat.ReactionService
	ReadReceiptService *serviceschat.ReadReceiptService
}

func NewWSHandler(manager *ws.WebSocketManager, chat *serviceschat.ChatService, attach *serviceschat.AttachmentService, react *serviceschat.ReactionService, read *serviceschat.ReadReceiptService) *WSHandler {
	return &WSHandler{
		Manager:            manager,
		ChatService:        chat,
		AttachmentService:  attach,
		ReactionService:    react,
		ReadReceiptService: read,
	}
}

func (h *WSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// userID должен быть извлечён из контекста (через JWT middleware)
	userID := r.Context().Value("user_id").(string)

	ws.ServeWS(
		h.Manager,
		w,
		r,
		userID,
		h.ChatService,
		h.AttachmentService,
		h.ReactionService,
		h.ReadReceiptService,
	)
}

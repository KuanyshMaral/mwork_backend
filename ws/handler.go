package ws

import (
	"net/http"

	chat "mwork_backend/internal/services/chat"
)

// WebSocketHandler — структура для инициализации обработчика с зависимостями
type WebSocketHandler struct {
	Manager            *WebSocketManager
	ChatService        *chat.ChatService
	AttachmentService  *chat.AttachmentService
	ReactionService    *chat.ReactionService
	ReadReceiptService *chat.ReadReceiptService
}

// NewWebSocketHandler — инициализация хендлера
func NewWebSocketHandler(
	manager *WebSocketManager,
	chatService *chat.ChatService,
	attachmentService *chat.AttachmentService,
	reactionService *chat.ReactionService,
	readReceiptService *chat.ReadReceiptService,
) *WebSocketHandler {
	return &WebSocketHandler{
		Manager:            manager,
		ChatService:        chatService,
		AttachmentService:  attachmentService,
		ReactionService:    reactionService,
		ReadReceiptService: readReceiptService,
	}
}

// HandleWebSocketConnection — основной HTTP endpoint
func (h *WebSocketHandler) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	// Пример: получить userID из middleware/jwt
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Обработка WebSocket соединения
	ServeWS(
		h.Manager,
		w,
		r,
		userID.(string),
		h.ChatService,
		h.AttachmentService,
		h.ReactionService,
		h.ReadReceiptService,
	)
}

package ws

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшн добавьте проверку origin
	},
}

type WebSocketHandler struct {
	Manager *WebSocketManager
}

func NewWebSocketHandler(manager *WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{
		Manager: manager,
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ChatService теперь передаётся через Manager
	ServeWS(h.Manager, w, r, userID)
}

// ServeWS теперь принимает интерфейс
func ServeWS(
	manager *WebSocketManager,
	w http.ResponseWriter,
	r *http.Request,
	userID string,
) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		ID:      userID,
		Conn:    conn,
		Send:    make(chan any, 256), // Буферизованный канал
		Ctx:     r.Context(),
		Manager: manager,
	}

	manager.register <- client

	go client.readPump()
	go client.writePump()
}

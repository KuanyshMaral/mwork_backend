package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшн добавьте проверку origin
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Убрана зависимость от BaseHandler
type WebSocketHandler struct {
	Manager *WebSocketManager
}

func NewWebSocketHandler(manager *WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{
		Manager: manager,
	}
}

func (h *WebSocketHandler) ServeWS(c *gin.Context) {

	// Получаем userID из middleware (как в BaseHandler)
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized. Missing userID in context."})
		return
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized. Invalid userID format in context."})
		return
	}

	// Обновляем соединение
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	log.Printf("WebSocket-клиент %s подключен\n", userID)

	// Создаем клиента
	client := &Client{
		ID:      userID, // <-- Используем userID из middleware
		Conn:    conn,
		Send:    make(chan any, 256),
		Ctx:     c.Request.Context(),
		Manager: h.Manager,
	}

	// Регистрируем клиента
	h.Manager.register <- client

	// Запускаем read/write
	go client.readPump()
	go client.writePump()
}

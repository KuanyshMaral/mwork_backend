package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin" // <--- ИМПОРТИРУЕМ GIN
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшн добавьте проверку origin
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketHandler struct {
	Manager *WebSocketManager
}

func NewWebSocketHandler(manager *WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{
		Manager: manager,
	}
}

//
// УДАЛЕНО: func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request)
// УДАЛЕНО: func ServeWS(...)
//

// ИСПРАВЛЕНИЕ: Мы создаем метод ServeWS(c *gin.Context), который ожидает app.go
func (h *WebSocketHandler) ServeWS(c *gin.Context) {

	// 1. Получаем userID из Gin-контекста
	// В production, лучше получать его из auth middleware:
	// userID, exists := c.Get("userID")
	// if !exists {
	//    c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	//	  return
	// }
	//
	// Временное решение: получаем из query-параметра
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized. 'user_id' query parameter is required."})
		return
	}

	// 2. Обновляем соединение, используя c.Writer и c.Request из Gin
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	log.Printf("WebSocket-клиент %s подключен\n", userID)

	// 3. Создаем клиента, используя h.Manager из структуры
	client := &Client{
		ID:      userID,
		Conn:    conn,
		Send:    make(chan any, 256), // Буферизованный канал
		Ctx:     c.Request.Context(), // Используем контекст из Gin
		Manager: h.Manager,           // Используем Manager из *h
	}

	// 4. Регистрируем клиента в менеджере
	h.Manager.register <- client

	// 5. Запускаем read/write
	go client.readPump()
	go client.writePump()
}

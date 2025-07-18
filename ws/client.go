package ws

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	chat "mwork_backend/internal/services/chat"
	"net/http"
)

type IncomingWSMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan any
	Ctx  context.Context

	Manager            *WebSocketManager
	ChatService        *chat.ChatService
	AttachmentService  *chat.AttachmentService
	ReactionService    *chat.ReactionService
	ReadReceiptService *chat.ReadReceiptService
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // продакшн: проверка origin
	},
}

func ServeWS(
	manager *WebSocketManager,
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	chatSvc *chat.ChatService,
	attachSvc *chat.AttachmentService,
	reactionSvc *chat.ReactionService,
	readReceiptSvc *chat.ReadReceiptService,
) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		ID:                 userID,
		Conn:               conn,
		Send:               make(chan any),
		Ctx:                context.Background(),
		Manager:            manager,
		ChatService:        chatSvc,
		AttachmentService:  attachSvc,
		ReactionService:    reactionSvc,
		ReadReceiptService: readReceiptSvc,
	}

	manager.register <- client

	go client.readPump()
	go client.writePump()
}

func (c *Client) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break
		}

		var msg IncomingWSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Println("Failed to parse message:", err)
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) writePump() {
	for msg := range c.Send {
		if err := c.Conn.WriteJSON(msg); err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
	}
}

// Централизованный обработчик
func (c *Client) handleMessage(msg IncomingWSMessage) {
	switch msg.Action {

	case "send_message":
		var input chat.SendMessageInput
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			log.Println("Invalid send_message payload:", err)
			return
		}
		createdMsg, err := c.ChatService.SendMessage(input)
		if err != nil {
			log.Println("Failed to send message:", err)
			return
		}
		c.Send <- createdMsg

	case "add_reaction":
		var payload struct {
			UserID    string `json:"user_id"`
			MessageID string `json:"message_id"`
			Emoji     string `json:"emoji"`
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Println("Invalid add_reaction payload:", err)
			return
		}
		if err := c.ReactionService.Add(payload.UserID, payload.MessageID, payload.Emoji); err != nil {
			log.Println("Failed to add reaction:", err)
		}

	case "mark_as_read":
		var payload struct {
			UserID    string `json:"user_id"`
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Println("Invalid mark_as_read payload:", err)
			return
		}
		if err := c.ReadReceiptService.MarkAsRead(payload.UserID, payload.MessageID); err != nil {
			log.Println("Failed to mark as read:", err)
		}

	default:
		log.Println("Unhandled action:", msg.Action)
	}
}

package ws

import (
	"context"
	"encoding/json"
	"log"
	"mwork_backend/internal/services/dto"

	"github.com/gorilla/websocket"
)

type IncomingWSMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type Client struct {
	ID      string
	Conn    *websocket.Conn
	Send    chan any
	Ctx     context.Context
	Manager *WebSocketManager
}

func (c *Client) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket close error: %v", err)
			}
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
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// Канал закрыт
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (c *Client) handleMessage(msg IncomingWSMessage) {
	// Получаем DB сессию из менеджера, используя контекст этого клиента
	db := c.Manager.getDB(c.Ctx)

	switch msg.Action {
	case "send_message":
		var input struct {
			DialogID  string  `json:"dialog_id"`
			Type      string  `json:"type"`
			Content   string  `json:"content"`
			ReplyToID *string `json:"reply_to_id"`
		}
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			log.Println("Invalid send_message payload:", err)
			c.Send <- map[string]string{"error": "invalid_payload"}
			return
		}

		req := &dto.SendMessageRequest{
			DialogID:  input.DialogID,
			Type:      input.Type,
			Content:   input.Content,
			ReplyToID: input.ReplyToID,
		}

		// Управляем транзакцией вручную
		tx := db.Begin()
		if tx.Error != nil {
			log.Println("Failed to start transaction:", tx.Error)
			c.Send <- map[string]string{"error": "internal_error"}
			return
		}

		// Передаем транзакцию (tx) в сервис
		createdMsg, err := c.Manager.chatService.SendMessage(tx, c.ID, req)
		if err != nil {
			tx.Rollback() // <-- Откат
			log.Println("Failed to send message:", err)
			c.Send <- map[string]string{"error": "failed_to_send"}
			return
		}

		if err := tx.Commit().Error; err != nil { // <-- Коммит
			log.Println("Failed to commit transaction:", err)
			c.Send <- map[string]string{"error": "internal_error"}
			return
		}

		// Передаем context для широковещания
		c.Manager.BroadcastToDialog(c.Ctx, input.DialogID, map[string]interface{}{
			"action": "new_message",
			"data":   createdMsg,
		})

	case "typing_start":
		var input struct {
			DialogID string `json:"dialog_id"`
		}
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			log.Println("Invalid typing_start payload:", err)
			return
		}

		// Передаем 'db' (транзакция не обязательна для typing)
		if err := c.Manager.chatService.SetTyping(db, c.ID, input.DialogID, true); err != nil {
			log.Println("Failed to set typing:", err)
			return
		}

		// Передаем context
		c.Manager.BroadcastToDialog(c.Ctx, input.DialogID, map[string]interface{}{
			"action": "user_typing",
			"data": map[string]interface{}{
				"user_id":   c.ID,
				"dialog_id": input.DialogID,
				"typing":    true,
			},
		})

	case "typing_stop":
		var input struct {
			DialogID string `json:"dialog_id"`
		}
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			log.Println("Invalid typing_stop payload:", err)
			return
		}

		// Передаем 'db'
		if err := c.Manager.chatService.SetTyping(db, c.ID, input.DialogID, false); err != nil {
			log.Println("Failed to stop typing:", err)
			return
		}

	case "mark_as_read":
		var input struct {
			DialogID string `json:"dialog_id"`
		}
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			log.Println("Invalid mark_as_read payload:", err)
			return
		}

		// Используем транзакцию, так как обновляем много сообщений
		tx := db.Begin()
		if tx.Error != nil {
			log.Println("Failed to start transaction:", tx.Error)
			return
		}

		// Передаем 'tx'
		if err := c.Manager.chatService.MarkMessagesAsRead(tx, c.ID, input.DialogID); err != nil {
			tx.Rollback()
			log.Println("Failed to mark as read:", err)
			return
		}

		if err := tx.Commit().Error; err != nil {
			log.Println("Failed to commit transaction:", err)
		}

	default:
		log.Println("Unhandled action:", msg.Action)
		c.Send <- map[string]string{"error": "unknown_action"}
	}
}

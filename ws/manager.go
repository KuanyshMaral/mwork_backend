package ws

import (
	"log"
	"mwork_backend/internal/services/chat"
	"sync"
)

type WebSocketManager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan any
	mu         sync.RWMutex

	// chat services
	chatService        *chat.ChatService
	attachmentService  *chat.AttachmentService
	reactionService    *chat.ReactionService
	readReceiptService *chat.ReadReceiptService
}

func NewWebSocketManager(
	chat *chat.ChatService,
	attachments *chat.AttachmentService,
	reactions *chat.ReactionService,
	readReceipts *chat.ReadReceiptService,
) *WebSocketManager {
	return &WebSocketManager{
		clients:            make(map[string]*Client),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		broadcast:          make(chan any),
		chatService:        chat,
		attachmentService:  attachments,
		reactionService:    reactions,
		readReceiptService: readReceipts,
	}
}

func (manager *WebSocketManager) Run() {
	for {
		select {
		case client := <-manager.register:
			manager.mu.Lock()
			manager.clients[client.ID] = client
			manager.mu.Unlock()
			log.Printf("Client registered: %s", client.ID)

		case client := <-manager.unregister:
			manager.mu.Lock()
			if _, ok := manager.clients[client.ID]; ok {
				close(client.Send)
				delete(manager.clients, client.ID)
				log.Printf("Client unregistered: %s", client.ID)
			}
			manager.mu.Unlock()

		case message := <-manager.broadcast:
			manager.mu.RLock()
			for _, client := range manager.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(manager.clients, client.ID)
				}
			}
			manager.mu.RUnlock()
		}
	}
}

// BroadcastToClient позволяет отправить сообщение одному клиенту по ID
func (manager *WebSocketManager) BroadcastToClient(clientID string, message any) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if client, ok := manager.clients[clientID]; ok {
		client.Send <- message
	}
}

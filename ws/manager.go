package ws

import (
	"context" // <-- Добавлен
	"log"
	"mwork_backend/internal/services"
	"sync"

	"gorm.io/gorm" // <-- Добавлен
)

type WebSocketManager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan any
	mu         sync.RWMutex

	chatService services.ChatService
	dbPool      *gorm.DB // <-- Храним главный пул DB
}

// Принимаем dbPool
func NewWebSocketManager(chatService services.ChatService, dbPool *gorm.DB) *WebSocketManager {
	return &WebSocketManager{
		clients:     make(map[string]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan any),
		chatService: chatService,
		dbPool:      dbPool, // <-- Сохраняем пул
	}
}

// getDB создает новую DB-сессию из пула для конкретной операции
func (manager *WebSocketManager) getDB(ctx context.Context) *gorm.DB {
	return manager.dbPool.WithContext(ctx)
}

func (manager *WebSocketManager) Run() {
	for {
		select {
		case client := <-manager.register:
			manager.mu.Lock()
			manager.clients[client.ID] = client
			manager.mu.Unlock()
			log.Printf("Client registered: %s, total: %d", client.ID, len(manager.clients))

		case client := <-manager.unregister:
			manager.mu.Lock()
			if _, ok := manager.clients[client.ID]; ok {
				close(client.Send)
				delete(manager.clients, client.ID)
				log.Printf("Client unregistered: %s, total: %d", client.ID, len(manager.clients))
			}
			manager.mu.Unlock()

		case message := <-manager.broadcast:
			manager.broadcastMessage(message)
		}
	}
}

// BroadcastToDialog отправляет сообщение всем участникам диалога
// (Принимает context)
func (manager *WebSocketManager) BroadcastToDialog(ctx context.Context, dialogID string, message any) {
	// Получаем участников диалога из ChatService
	participants, err := manager.getDialogParticipants(ctx, dialogID) // Передаем context
	if err != nil {
		log.Printf("Failed to get dialog participants: %v", err)
		return
	}

	manager.mu.RLock()
	defer manager.mu.RUnlock()

	for _, participantID := range participants {
		if client, ok := manager.clients[participantID]; ok {
			select {
			case client.Send <- message:
				// Сообщение отправлено
			default:
				// Канал заполнен, клиент отключается
				go func(clientID string) {
					manager.unregister <- manager.clients[clientID]
				}(participantID)
			}
		}
	}
}

// getDialogParticipants - вспомогательный метод (Принимает context)
func (manager *WebSocketManager) getDialogParticipants(ctx context.Context, dialogID string) ([]string, error) {
	// Создаем DB сессию для этого запроса
	db := manager.getDB(ctx)

	// Передаем 'db' в сервис
	dialog, err := manager.chatService.GetDialog(db, dialogID, "")
	if err != nil {
		return nil, err
	}

	// Извлекаем ID участников из ответа
	var participantIDs []string
	for _, participant := range dialog.Participants {
		participantIDs = append(participantIDs, participant.UserID)
	}

	return participantIDs, nil
}

func (manager *WebSocketManager) broadcastMessage(message any) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	for clientID, client := range manager.clients {
		select {
		case client.Send <- message:
			// Сообщение отправлено
		default:
			// Канал заполнен, клиент отключается
			go func() {
				manager.unregister <- client
			}()
			log.Printf("Client %s disconnected due to full send channel", clientID)
		}
	}
}

// BroadcastToClient отправляет сообщение конкретному клиенту
func (manager *WebSocketManager) BroadcastToClient(clientID string, message any) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if client, ok := manager.clients[clientID]; ok {
		select {
		case client.Send <- message:
		default:
			go func() {
				manager.unregister <- client
			}()
		}
	}
}

// GetClientCount возвращает количество подключенных клиентов
func (manager *WebSocketManager) GetClientCount() int {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return len(manager.clients)
}

// IsClientConnected проверяет, подключен ли клиент
func (manager *WebSocketManager) IsClientConnected(clientID string) bool {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	_, exists := manager.clients[clientID]
	return exists
}

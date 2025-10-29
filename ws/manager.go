package ws

import (
	"log"
	"mwork_backend/internal/services"
	"sync"
)

type WebSocketManager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan any
	mu         sync.RWMutex

	chatService services.ChatService // Используем интерфейс
}

func NewWebSocketManager(chatService services.ChatService) *WebSocketManager {
	return &WebSocketManager{
		clients:     make(map[string]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan any),
		chatService: chatService,
	}
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
func (manager *WebSocketManager) BroadcastToDialog(dialogID string, message any) {
	// Получаем участников диалога из ChatService
	// ВАЖНО: Вам нужно добавить метод GetDialogParticipants в ваш ChatService
	// или использовать существующие методы для получения участников
	participants, err := manager.getDialogParticipants(dialogID)
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

// getDialogParticipants - вспомогательный метод для получения участников диалога
func (manager *WebSocketManager) getDialogParticipants(dialogID string) ([]string, error) {
	// Получаем информацию о диалоге
	dialog, err := manager.chatService.GetDialog(dialogID, "") // userID может быть пустым для административных целей
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

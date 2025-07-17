package chat

import (
	"time"

	modelChat "mwork_front_fn/internal/models/chat"
	repoChat "mwork_front_fn/internal/repositories/chat"
)

type ReadReceiptService struct {
	Repo    *repoChat.MessageReadReceiptRepository
	MsgRepo *repoChat.MessageRepository
}

func NewReadReceiptService(
	repo *repoChat.MessageReadReceiptRepository,
	msgRepo *repoChat.MessageRepository,
) *ReadReceiptService {
	return &ReadReceiptService{
		Repo:    repo,
		MsgRepo: msgRepo,
	}
}

// MarkAsRead помечает конкретное сообщение как прочитанное
func (s *ReadReceiptService) MarkAsRead(userID, messageID string) error {
	exists, err := s.Repo.Exists(userID, messageID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	receipt := &modelChat.MessageReadReceipt{
		UserID:    userID,
		MessageID: messageID,
		ReadAt:    time.Now(),
	}
	return s.Repo.Create(receipt)
}

// MarkAllAsRead помечает все сообщения в чате как прочитанные
func (s *ReadReceiptService) MarkAllAsRead(userID, dialogID string) error {
	messages, err := s.MsgRepo.GetByDialog(dialogID)
	if err != nil {
		return err
	}

	var receipts []modelChat.MessageReadReceipt
	now := time.Now()

	for _, msg := range messages {
		exists, err := s.Repo.Exists(userID, msg.ID)
		if err != nil {
			return err
		}
		if !exists {
			receipts = append(receipts, modelChat.MessageReadReceipt{
				UserID:    userID,
				MessageID: msg.ID,
				ReadAt:    now,
			})
		}
	}

	if len(receipts) > 0 {
		return s.Repo.CreateMany(receipts)
	}
	return nil
}

// GetByMessageID возвращает список кто прочитал сообщение
func (s *ReadReceiptService) GetByMessageID(messageID string) ([]modelChat.MessageReadReceipt, error) {
	return s.Repo.GetByMessageID(messageID)
}

// GetByUserAndDialog возвращает все прочтённые сообщения пользователем
func (s *ReadReceiptService) GetByUserAndDialog(userID, dialogID string) ([]modelChat.MessageReadReceipt, error) {
	return s.Repo.GetByUserAndDialog(userID, dialogID)
}

// GetUnreadCount возвращает количество непрочитанных сообщений
func (s *ReadReceiptService) GetUnreadCount(userID, dialogID string) (int64, error) {
	return s.Repo.GetUnreadCountByDialog(userID, dialogID)
}

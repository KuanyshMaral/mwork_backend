package chat

import (
	"time"

	"github.com/google/uuid"
	modelChat "mwork_front_fn/internal/models/chat"
	repoChat "mwork_front_fn/internal/repositories/chat"
)

type AttachmentService struct {
	Repo *repoChat.MessageAttachmentRepository
}

func NewAttachmentService(repo *repoChat.MessageAttachmentRepository) *AttachmentService {
	return &AttachmentService{Repo: repo}
}

type AttachmentInput struct {
	URL      string
	Type     string // image, video, file, etc.
	MimeType string
	Size     int64
}

// AddToMessage добавляет вложения к сообщению
func (s *AttachmentService) AddToMessage(messageID, uploaderID string, inputs []AttachmentInput) ([]modelChat.MessageAttachment, error) {
	attachments := make([]modelChat.MessageAttachment, 0, len(inputs))

	for _, in := range inputs {
		attachments = append(attachments, modelChat.MessageAttachment{
			ID:         uuid.New().String(),
			MessageID:  messageID,
			URL:        in.URL,
			FileType:   in.Type,
			MimeType:   in.MimeType,
			Size:       in.Size,
			UploaderID: uploaderID,
			CreatedAt:  time.Now(),
		})
	}

	if err := s.Repo.CreateMany(attachments); err != nil {
		return nil, err
	}

	return attachments, nil
}

// GetByMessageID возвращает все вложения сообщения
func (s *AttachmentService) GetByMessageID(messageID string) ([]modelChat.MessageAttachment, error) {
	return s.Repo.GetByMessageID(messageID)
}

// DeleteByID удаляет конкретное вложение
func (s *AttachmentService) DeleteByID(id string) error {
	return s.Repo.DeleteByID(id)
}

// DeleteByMessageID удаляет все вложения сообщения
func (s *AttachmentService) DeleteByMessageID(messageID string) error {
	return s.Repo.DeleteByMessageID(messageID)
}

// GetByDialogID возвращает все вложения из диалога (с опциональной фильтрацией)
func (s *AttachmentService) GetByDialogID(dialogID string, filterType *string) ([]modelChat.MessageAttachment, error) {
	return s.Repo.GetByDialogID(dialogID, filterType)
}

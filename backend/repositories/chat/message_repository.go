package chat

import (
	"gorm.io/gorm"
	"mwork_front_fn/backend/models/chat"
)

type MessageRepository struct {
	DB *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{DB: db}
}

// Create сохраняет новое сообщение
func (r *MessageRepository) Create(message *chat.Message) error {
	return r.DB.Create(message).Error
}

// GetByDialog возвращает все сообщения по диалогу
func (r *MessageRepository) GetByDialog(dialogID string) ([]chat.Message, error) {
	var messages []chat.Message
	err := r.DB.Where("dialog_id = ?", dialogID).Order("created_at ASC").
		Preload("Reactions").Preload("ReplyTo").Preload("ForwardFrom").Preload("Attachments").
		Find(&messages).Error
	return messages, err
}

// GetByID возвращает одно сообщение по ID
func (r *MessageRepository) GetByID(id string) (*chat.Message, error) {
	var msg chat.Message
	err := r.DB.Preload("Reactions").Preload("ReplyTo").Preload("ForwardFrom").Preload("Attachments").
		First(&msg, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// DeleteByID удаляет сообщение по ID
func (r *MessageRepository) DeleteByID(id string) error {
	return r.DB.Where("id = ?", id).Delete(&chat.Message{}).Error
}

// UpdateStatus обновляет статус сообщения (read, delivered и т.п.)
func (r *MessageRepository) UpdateStatus(id, status string) error {
	return r.DB.Model(&chat.Message{}).Where("id = ?", id).Update("status", status).Error
}

// FindLatestInDialog возвращает последнее сообщение в диалоге
func (r *MessageRepository) FindLatestInDialog(dialogID string) (*chat.Message, error) {
	var msg chat.Message
	err := r.DB.Where("dialog_id = ?", dialogID).
		Order("created_at DESC").Limit(1).
		Preload("Reactions").Preload("Attachments").
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

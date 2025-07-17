package chat

import (
	"gorm.io/gorm"
	"mwork_front_fn/backend/models/chat"
)

type MessageAttachmentRepository struct {
	DB *gorm.DB
}

func NewMessageAttachmentRepository(db *gorm.DB) *MessageAttachmentRepository {
	return &MessageAttachmentRepository{DB: db}
}

// Create сохраняет одно вложение
func (r *MessageAttachmentRepository) Create(attachment *chat.MessageAttachment) error {
	return r.DB.Create(attachment).Error
}

// CreateMany сохраняет несколько вложений за раз (например, при multi-upload)
func (r *MessageAttachmentRepository) CreateMany(attachments []chat.MessageAttachment) error {
	return r.DB.Create(&attachments).Error
}

// GetByMessageID возвращает вложения по сообщению
func (r *MessageAttachmentRepository) GetByMessageID(messageID string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	err := r.DB.Where("message_id = ?", messageID).Find(&attachments).Error
	return attachments, err
}

// GetByID получает одно вложение по ID
func (r *MessageAttachmentRepository) GetByID(id string) (*chat.MessageAttachment, error) {
	var attachment chat.MessageAttachment
	err := r.DB.First(&attachment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// DeleteByMessageID удаляет все вложения, связанные с сообщением
func (r *MessageAttachmentRepository) DeleteByMessageID(messageID string) error {
	return r.DB.Where("message_id = ?", messageID).Delete(&chat.MessageAttachment{}).Error
}

// DeleteByID удаляет одно вложение по его ID
func (r *MessageAttachmentRepository) DeleteByID(id string) error {
	return r.DB.Where("id = ?", id).Delete(&chat.MessageAttachment{}).Error
}

// GetAllByUploader возвращает все вложения пользователя
func (r *MessageAttachmentRepository) GetAllByUploader(uploaderID string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	err := r.DB.Where("uploader_id = ?", uploaderID).Order("created_at DESC").Find(&attachments).Error
	return attachments, err
}

// Exists проверяет, существует ли вложение
func (r *MessageAttachmentRepository) Exists(id string) (bool, error) {
	var count int64
	err := r.DB.Model(&chat.MessageAttachment{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// GetByDialogID возвращает все вложения в рамках диалога (с фильтром по типу)
func (r *MessageAttachmentRepository) GetByDialogID(dialogID string, filterType *string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	query := r.DB.Joins("JOIN messages m ON m.id = message_attachments.message_id").
		Where("m.dialog_id = ?", dialogID).
		Order("message_attachments.created_at DESC")

	if filterType != nil {
		query = query.Where("message_attachments.type = ?", *filterType)
	}

	err := query.Find(&attachments).Error
	return attachments, err
}

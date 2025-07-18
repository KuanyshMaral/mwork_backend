package chat

import (
	"mwork_backend/internal/models/chat"

	"gorm.io/gorm"
)

type MessageReadReceiptRepository struct {
	DB *gorm.DB
}

func NewMessageReadReceiptRepository(db *gorm.DB) *MessageReadReceiptRepository {
	return &MessageReadReceiptRepository{DB: db}
}

// Create добавляет новую запись о прочтении
func (r *MessageReadReceiptRepository) Create(receipt *chat.MessageReadReceipt) error {
	return r.DB.Create(receipt).Error
}

// CreateMany — массовая вставка (например, при sync с мобилки)
func (r *MessageReadReceiptRepository) CreateMany(receipts []chat.MessageReadReceipt) error {
	return r.DB.Create(&receipts).Error
}

// GetByMessageID возвращает всех, кто прочитал сообщение
func (r *MessageReadReceiptRepository) GetByMessageID(messageID string) ([]chat.MessageReadReceipt, error) {
	var receipts []chat.MessageReadReceipt
	err := r.DB.Where("message_id = ?", messageID).Find(&receipts).Error
	return receipts, err
}

// GetByUserAndDialog возвращает все прочтённые сообщения пользователем в чате
func (r *MessageReadReceiptRepository) GetByUserAndDialog(userID, dialogID string) ([]chat.MessageReadReceipt, error) {
	var receipts []chat.MessageReadReceipt
	err := r.DB.
		Joins("JOIN messages m ON m.id = message_read_receipts.message_id").
		Where("m.dialog_id = ? AND message_read_receipts.user_id = ?", dialogID, userID).
		Find(&receipts).Error
	return receipts, err
}

// Exists проверяет, читал ли пользователь сообщение
func (r *MessageReadReceiptRepository) Exists(userID, messageID string) (bool, error) {
	var count int64
	err := r.DB.Model(&chat.MessageReadReceipt{}).
		Where("user_id = ? AND message_id = ?", userID, messageID).
		Count(&count).Error
	return count > 0, err
}

// DeleteByMessageID удаляет все read-события, связанные с сообщением
func (r *MessageReadReceiptRepository) DeleteByMessageID(messageID string) error {
	return r.DB.Where("message_id = ?", messageID).Delete(&chat.MessageReadReceipt{}).Error
}

// DeleteByUserID удаляет все read-события пользователя (например, при удалении аккаунта)
func (r *MessageReadReceiptRepository) DeleteByUserID(userID string) error {
	return r.DB.Where("user_id = ?", userID).Delete(&chat.MessageReadReceipt{}).Error
}

// GetUnreadCountByDialog возвращает количество непрочитанных сообщений в диалоге для пользователя
func (r *MessageReadReceiptRepository) GetUnreadCountByDialog(userID, dialogID string) (int64, error) {
	var count int64
	err := r.DB.
		Raw(`
			SELECT COUNT(*) FROM messages m
			WHERE m.dialog_id = ?
			AND NOT EXISTS (
				SELECT 1 FROM message_read_receipts r
				WHERE r.message_id = m.id AND r.user_id = ?
			)
		`, dialogID, userID).
		Scan(&count).Error
	return count, err
}

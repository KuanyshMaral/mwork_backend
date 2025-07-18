package chat

import (
	"mwork_backend/internal/models/chat"

	"gorm.io/gorm"
)

type MessageReactionRepository struct {
	DB *gorm.DB
}

func NewMessageReactionRepository(db *gorm.DB) *MessageReactionRepository {
	return &MessageReactionRepository{DB: db}
}

// Add добавляет реакцию к сообщению
func (r *MessageReactionRepository) Add(reaction *chat.MessageReaction) error {
	return r.DB.Create(reaction).Error
}

// Remove удаляет реакцию пользователя на сообщение
func (r *MessageReactionRepository) Remove(userID, messageID, emoji string) error {
	return r.DB.Where("user_id = ? AND message_id = ? AND emoji = ?", userID, messageID, emoji).
		Delete(&chat.MessageReaction{}).Error
}

// GetByMessageID возвращает все реакции на сообщение
func (r *MessageReactionRepository) GetByMessageID(messageID string) ([]chat.MessageReaction, error) {
	var reactions []chat.MessageReaction
	err := r.DB.Where("message_id = ?", messageID).Find(&reactions).Error
	return reactions, err
}

// Exists проверяет, поставил ли пользователь реакцию на сообщение
func (r *MessageReactionRepository) Exists(userID, messageID, emoji string) (bool, error) {
	var count int64
	err := r.DB.Model(&chat.MessageReaction{}).
		Where("user_id = ? AND message_id = ? AND emoji = ?", userID, messageID, emoji).
		Count(&count).Error
	return count > 0, err
}

// DeleteByMessageID удаляет все реакции, связанные с сообщением
func (r *MessageReactionRepository) DeleteByMessageID(messageID string) error {
	return r.DB.Where("message_id = ?", messageID).Delete(&chat.MessageReaction{}).Error
}

// ToggleReaction добавляет или удаляет реакцию по нажатию
func (r *MessageReactionRepository) ToggleReaction(userID, messageID, emoji string) error {
	exists, err := r.Exists(userID, messageID, emoji)
	if err != nil {
		return err
	}
	if exists {
		return r.Remove(userID, messageID, emoji)
	}
	reaction := &chat.MessageReaction{
		UserID:    userID,
		MessageID: messageID,
		Emoji:     emoji,
	}
	return r.Add(reaction)
}

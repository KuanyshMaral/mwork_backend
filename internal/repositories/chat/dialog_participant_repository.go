package chat

import (
	"time"

	"gorm.io/gorm"
	"mwork_front_fn/internal/models/chat"
)

type DialogParticipantRepository struct {
	DB *gorm.DB
}

func NewDialogParticipantRepository(db *gorm.DB) *DialogParticipantRepository {
	return &DialogParticipantRepository{DB: db}
}

// CreateMany добавляет сразу несколько участников
func (r *DialogParticipantRepository) CreateMany(participants []chat.DialogParticipant) error {
	return r.DB.Create(&participants).Error
}

// IsUserInDialog проверяет, состоит ли пользователь в диалоге
func (r *DialogParticipantRepository) IsUserInDialog(userID, dialogID string) (bool, error) {
	var count int64
	err := r.DB.Model(&chat.DialogParticipant{}).
		Where("user_id = ? AND dialog_id = ?", userID, dialogID).
		Count(&count).Error

	return count > 0, err
}

// GetParticipants возвращает всех участников диалога
func (r *DialogParticipantRepository) GetParticipants(dialogID string) ([]chat.DialogParticipant, error) {
	var participants []chat.DialogParticipant
	err := r.DB.Where("dialog_id = ?", dialogID).Find(&participants).Error
	return participants, err
}

// UpdateLastSeen обновляет время последнего просмотра чата
func (r *DialogParticipantRepository) UpdateLastSeen(userID, dialogID string, t time.Time) error {
	return r.DB.Model(&chat.DialogParticipant{}).
		Where("user_id = ? AND dialog_id = ?", userID, dialogID).
		Update("last_seen_at", t).Error
}

// SetTypingUntil обновляет статус "печатает до..."
func (r *DialogParticipantRepository) SetTypingUntil(userID, dialogID string, until time.Time) error {
	return r.DB.Model(&chat.DialogParticipant{}).
		Where("user_id = ? AND dialog_id = ?", userID, dialogID).
		Update("typing_until", until).Error
}

// LeaveDialog устанавливает дату выхода из чата
func (r *DialogParticipantRepository) LeaveDialog(userID, dialogID string) error {
	now := time.Now()
	return r.DB.Model(&chat.DialogParticipant{}).
		Where("user_id = ? AND dialog_id = ?", userID, dialogID).
		Update("left_at", now).Error
}

// UpdateRole меняет роль пользователя в чате
func (r *DialogParticipantRepository) UpdateRole(userID, dialogID, role string) error {
	return r.DB.Model(&chat.DialogParticipant{}).
		Where("user_id = ? AND dialog_id = ?", userID, dialogID).
		Update("role", role).Error
}

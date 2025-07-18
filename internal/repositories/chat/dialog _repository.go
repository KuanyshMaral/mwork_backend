package chat

import (
	"mwork_backend/internal/models/chat"

	"gorm.io/gorm"
)

type DialogRepository struct {
	DB *gorm.DB
}

func NewDialogRepository(db *gorm.DB) *DialogRepository {
	return &DialogRepository{DB: db}
}

// FindByID возвращает диалог по ID с последним сообщением и участниками
func (r *DialogRepository) FindByID(id string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	err := r.DB.Preload("Participants").Preload("LastMessage").First(&dialog, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &dialog, nil
}

// FindPrivateDialog ищет существующий личный диалог между двумя пользователями
func (r *DialogRepository) FindPrivateDialog(user1ID, user2ID string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	err := r.DB.Raw(`
		SELECT d.* FROM dialogs d
		JOIN dialog_participants dp1 ON dp1.dialog_id = d.id AND dp1.user_id = ?
		JOIN dialog_participants dp2 ON dp2.dialog_id = d.id AND dp2.user_id = ?
		WHERE d.is_group = false
		LIMIT 1`, user1ID, user2ID).Scan(&dialog).Error

	if err != nil || dialog.ID == "" {
		return nil, err
	}
	return &dialog, nil
}

// Create создаёт новый диалог
func (r *DialogRepository) Create(dialog *chat.Dialog) error {
	return r.DB.Create(dialog).Error
}

// UpdateLastMessage обновляет последнее сообщение в диалоге
func (r *DialogRepository) UpdateLastMessage(dialogID string, messageID string) error {
	return r.DB.Model(&chat.Dialog{}).Where("id = ?", dialogID).Update("last_message_id", messageID).Error
}

// Delete удаляет диалог (hard delete)
func (r *DialogRepository) Delete(dialogID string) error {
	return r.DB.Where("id = ?", dialogID).Delete(&chat.Dialog{}).Error
}

// FindAllByUser возвращает все диалоги, в которых участвует пользователь
func (r *DialogRepository) FindAllByUser(userID string) ([]chat.Dialog, error) {
	var dialogs []chat.Dialog
	err := r.DB.
		Joins("JOIN dialog_participants dp ON dp.dialog_id = dialogs.id").
		Where("dp.user_id = ?", userID).
		Preload("Participants").
		Preload("LastMessage").
		Order("dialogs.updated_at DESC").
		Find(&dialogs).Error
	return dialogs, err
}

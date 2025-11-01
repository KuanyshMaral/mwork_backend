package repositories

import (
	"errors"
	"mwork_backend/internal/models"
	"mwork_backend/internal/models/chat"
	"time"

	"gorm.io/gorm"
)

var (
	ErrDialogNotFound      = errors.New("dialog not found")
	ErrMessageNotFound     = errors.New("message not found")
	ErrParticipantNotFound = errors.New("participant not found")
	ErrUserNotInDialog     = errors.New("user is not a participant in this dialog")
	ErrDialogAccessDenied  = errors.New("access to dialog denied")
	ErrCastingDialogExists = errors.New("dialog for this casting already exists")
)

type ChatRepository interface {
	// Dialog operations
	CreateDialog(db *gorm.DB, dialog *chat.Dialog) error
	FindDialogByID(db *gorm.DB, id string) (*chat.Dialog, error)
	FindDialogByCasting(db *gorm.DB, castingID string) (*chat.Dialog, error)
	FindUserDialogs(db *gorm.DB, userID string) ([]chat.Dialog, error)
	FindDialogBetweenUsers(db *gorm.DB, user1ID, user2ID string) (*chat.Dialog, error)
	UpdateDialog(db *gorm.DB, dialog *chat.Dialog) error
	DeleteDialog(db *gorm.DB, id string) error
	GetDialogWithParticipants(db *gorm.DB, dialogID string) (*chat.Dialog, error)

	// DialogParticipant operations
	AddParticipant(db *gorm.DB, participant *chat.DialogParticipant) error
	AddParticipants(db *gorm.DB, participants []*chat.DialogParticipant) error
	FindParticipant(db *gorm.DB, dialogID, userID string) (*chat.DialogParticipant, error)
	FindParticipantsByDialog(db *gorm.DB, dialogID string) ([]chat.DialogParticipant, error)
	FindDialogsByUser(db *gorm.DB, userID string) ([]chat.Dialog, error)
	UpdateParticipant(db *gorm.DB, participant *chat.DialogParticipant) error
	UpdateLastSeen(db *gorm.DB, dialogID, userID string, lastSeen time.Time) error
	RemoveParticipant(db *gorm.DB, dialogID, userID string) error
	RemoveAllParticipants(db *gorm.DB, dialogID string) error
	IsUserInDialog(db *gorm.DB, dialogID, userID string) (bool, error)

	// Message operations
	CreateMessage(db *gorm.DB, message *chat.Message) error
	FindMessageByID(db *gorm.DB, id string) (*chat.Message, error)
	FindMessagesByDialog(db *gorm.DB, dialogID string, criteria MessageCriteria) ([]chat.Message, int64, error)
	FindLastMessage(db *gorm.DB, dialogID string) (*chat.Message, error)
	UpdateMessageStatus(db *gorm.DB, messageID string, status string) error
	UpdateMessage(db *gorm.DB, message *chat.Message) error
	MarkMessagesAsRead(db *gorm.DB, dialogID, userID string) error
	DeleteMessage(db *gorm.DB, messageID string) error
	DeleteUserMessages(db *gorm.DB, dialogID, userID string) error

	// MessageAttachment operations
	CreateAttachment(db *gorm.DB, attachment *chat.MessageAttachment) error
	FindAttachmentsByMessage(db *gorm.DB, messageID string) ([]chat.MessageAttachment, error)
	FindAttachmentsByDialog(db *gorm.DB, dialogID string) ([]chat.MessageAttachment, error)
	DeleteAttachment(db *gorm.DB, attachmentID string) error

	// MessageReaction operations
	AddReaction(db *gorm.DB, reaction *chat.MessageReaction) error
	FindReactionsByMessage(db *gorm.DB, messageID string) ([]chat.MessageReaction, error)
	FindReaction(db *gorm.DB, messageID, userID string) (*chat.MessageReaction, error)
	RemoveReaction(db *gorm.DB, messageID, userID string) error
	RemoveAllReactions(db *gorm.DB, messageID string) error

	// MessageReadReceipt operations
	CreateReadReceipt(db *gorm.DB, receipt *chat.MessageReadReceipt) error
	FindReadReceiptsByMessage(db *gorm.DB, messageID string) ([]chat.MessageReadReceipt, error)
	FindUnreadMessages(db *gorm.DB, dialogID, userID string) ([]chat.Message, error)
	GetUnreadCount(db *gorm.DB, dialogID, userID string) (int64, error)

	// Combined operations
	CreateCastingDialog(db *gorm.DB, casting *models.Casting, employerID, modelID string) (*chat.Dialog, error)
	SendMessageWithAttachments(db *gorm.DB, senderID, dialogID, content string, attachments []*chat.MessageAttachment) (*chat.Message, error)
	GetDialogWithMessages(db *gorm.DB, dialogID string, userID string, criteria MessageCriteria) (*DialogWithMessages, error)

	// Admin operations
	FindAllDialogs(db *gorm.DB, criteria DialogCriteria) ([]chat.Dialog, int64, error)
	GetChatStats(db *gorm.DB) (*ChatStats, error)
	CleanOldMessages(db *gorm.DB, days int) error
}

type ChatRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
}

// Search criteria for messages
type MessageCriteria struct {
	BeforeID  string    `form:"before_id"` // For pagination
	AfterID   string    `form:"after_id"`  // For loading newer messages
	Limit     int       `form:"limit" binding:"min=1,max=100"`
	Offset    int       `form:"offset"`
	Types     []string  `form:"types"` // Filter by message types
	StartDate time.Time `form:"start_date"`
	EndDate   time.Time `form:"end_date"`
}

// Search criteria for dialogs (admin)
type DialogCriteria struct {
	IsGroup   *bool     `form:"is_group"`
	CastingID *string   `form:"casting_id"`
	UserID    string    `form:"user_id"`
	StartDate time.Time `form:"start_date"`
	EndDate   time.Time `form:"end_date"`
	Page      int       `form:"page" binding:"min=1"`
	PageSize  int       `form:"page_size" binding:"min=1,max=100"`
}

// Combined dialog with messages
type DialogWithMessages struct {
	Dialog   *chat.Dialog   `json:"dialog"`
	Messages []chat.Message `json:"messages"`
	Total    int64          `json:"total"`
	HasMore  bool           `json:"has_more"`
}

// Chat statistics
type ChatStats struct {
	TotalDialogs     int64            `json:"total_dialogs"`
	TotalMessages    int64            `json:"total_messages"`
	TotalAttachments int64            `json:"total_attachments"`
	ActiveDialogs    int64            `json:"active_dialogs"` // Dialogs with messages in last 7 days
	TodayMessages    int64            `json:"today_messages"`
	ThisWeekMessages int64            `json:"this_week_messages"`
	ByType           map[string]int64 `json:"by_type"` // Message types distribution
}

// ✅ Конструктор не принимает db
func NewChatRepository() ChatRepository {
	return &ChatRepositoryImpl{}
}

// Dialog operations

func (r *ChatRepositoryImpl) CreateDialog(db *gorm.DB, dialog *chat.Dialog) error {
	// Check if casting dialog already exists
	if dialog.CastingID != nil {
		var existing chat.Dialog
		// ✅ Используем 'db' из параметра
		if err := db.Where("casting_id = ?", *dialog.CastingID).First(&existing).Error; err == nil {
			return ErrCastingDialogExists
		}
	}

	// ✅ Используем 'db' из параметра
	return db.Create(dialog).Error
}

func (r *ChatRepositoryImpl) FindDialogByID(db *gorm.DB, id string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	// ✅ Используем 'db' из параметра
	err := db.Preload("Participants").Preload("LastMessage").
		First(&dialog, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}
	return &dialog, nil
}

func (r *ChatRepositoryImpl) FindDialogByCasting(db *gorm.DB, castingID string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	// ✅ Используем 'db' из параметра
	err := db.Preload("Participants").Preload("LastMessage").
		Where("casting_id = ?", castingID).First(&dialog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}
	return &dialog, nil
}

func (r *ChatRepositoryImpl) FindUserDialogs(db *gorm.DB, userID string) ([]chat.Dialog, error) {
	var dialogs []chat.Dialog

	// ✅ Используем 'db' из параметра
	err := db.Preload("Participants").Preload("LastMessage").
		Joins("LEFT JOIN dialog_participants ON dialogs.id = dialog_participants.dialog_id").
		Where("dialog_participants.user_id = ? AND dialog_participants.left_at IS NULL", userID).
		Order("dialogs.updated_at DESC").
		Find(&dialogs).Error

	return dialogs, err
}

func (r *ChatRepositoryImpl) FindDialogBetweenUsers(db *gorm.DB, user1ID, user2ID string) (*chat.Dialog, error) {
	var dialog chat.Dialog

	// ✅ Используем 'db' из параметра
	err := db.Preload("Participants").Preload("LastMessage").
		Joins("LEFT JOIN dialog_participants dp1 ON dialogs.id = dp1.dialog_id").
		Joins("LEFT JOIN dialog_participants dp2 ON dialogs.id = dp2.dialog_id").
		Where("dialogs.is_group = ?", false).
		Where("dp1.user_id = ? AND dp2.user_id = ?", user1ID, user2ID).
		Where("dp1.left_at IS NULL AND dp2.left_at IS NULL").
		Group("dialogs.id").
		Having("COUNT(DISTINCT dialog_participants.user_id) = 2"). // Ошибка была здесь, исправил на dialog_participants.user_id
		First(&dialog).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}

	return &dialog, nil
}

func (r *ChatRepositoryImpl) UpdateDialog(db *gorm.DB, dialog *chat.Dialog) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(dialog).Updates(map[string]interface{}{
		"title":           dialog.Title,
		"image_url":       dialog.ImageURL,
		"last_message_id": dialog.LastMessageID,
		"updated_at":      time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDialogNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) DeleteDialog(db *gorm.DB, id string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	// Delete messages and related data first
	if err := db.Where("dialog_id = ?", id).Delete(&chat.MessageReadReceipt{}).Error; err != nil {
		return err
	}
	if err := db.Where("dialog_id = ?", id).Delete(&chat.MessageReaction{}).Error; err != nil {
		return err
	}
	if err := db.Where("dialog_id = ?", id).Delete(&chat.MessageAttachment{}).Error; err != nil {
		return err
	}
	if err := db.Where("dialog_id = ?", id).Delete(&chat.Message{}).Error; err != nil {
		return err
	}

	// Delete participants
	if err := db.Where("dialog_id = ?", id).Delete(&chat.DialogParticipant{}).Error; err != nil {
		return err
	}

	// Delete dialog
	result := db.Where("id = ?", id).Delete(&chat.Dialog{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDialogNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) GetDialogWithParticipants(db *gorm.DB, dialogID string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	// ✅ Используем 'db' из параметра
	err := db.Preload("Participants").First(&dialog, "id = ?", dialogID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}
	return &dialog, nil
}

// DialogParticipant operations

func (r *ChatRepositoryImpl) AddParticipant(db *gorm.DB, participant *chat.DialogParticipant) error {
	// ✅ Используем 'db' из параметра
	return db.Create(participant).Error
}

func (r *ChatRepositoryImpl) AddParticipants(db *gorm.DB, participants []*chat.DialogParticipant) error {
	if len(participants) == 0 {
		return nil
	}
	// ✅ Используем 'db' из параметра
	return db.CreateInBatches(participants, 50).Error
}

func (r *ChatRepositoryImpl) FindParticipant(db *gorm.DB, dialogID, userID string) (*chat.DialogParticipant, error) {
	var participant chat.DialogParticipant
	// ✅ Используем 'db' из параметра
	err := db.Where("dialog_id = ? AND user_id = ?", dialogID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}
	return &participant, nil
}

func (r *ChatRepositoryImpl) FindParticipantsByDialog(db *gorm.DB, dialogID string) ([]chat.DialogParticipant, error) {
	var participants []chat.DialogParticipant
	// ✅ Используем 'db' из параметра
	err := db.Where("dialog_id = ? AND left_at IS NULL", dialogID).Find(&participants).Error
	return participants, err
}

func (r *ChatRepositoryImpl) FindDialogsByUser(db *gorm.DB, userID string) ([]chat.Dialog, error) {
	var dialogs []chat.Dialog
	// ✅ Используем 'db' из параметра
	err := db.Preload("LastMessage").
		Joins("LEFT JOIN dialog_participants ON dialogs.id = dialog_participants.dialog_id").
		Where("dialog_participants.user_id = ? AND dialog_participants.left_at IS NULL", userID).
		Order("dialogs.updated_at DESC").
		Find(&dialogs).Error
	return dialogs, err
}

func (r *ChatRepositoryImpl) UpdateParticipant(db *gorm.DB, participant *chat.DialogParticipant) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(participant).Updates(map[string]interface{}{
		"role":         participant.Role,
		"is_muted":     participant.IsMuted,
		"typing_until": participant.TypingUntil,
		"last_seen_at": participant.LastSeenAt,
		"left_at":      participant.LeftAt,
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) UpdateLastSeen(db *gorm.DB, dialogID, userID string, lastSeen time.Time) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&chat.DialogParticipant{}).
		Where("dialog_id = ? AND user_id = ?", dialogID, userID).
		Update("last_seen_at", lastSeen)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) RemoveParticipant(db *gorm.DB, dialogID, userID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&chat.DialogParticipant{}).
		Where("dialog_id = ? AND user_id = ?", dialogID, userID).
		Update("left_at", time.Now())

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) RemoveAllParticipants(db *gorm.DB, dialogID string) error {
	// ✅ Используем 'db' из параметра
	return db.Model(&chat.DialogParticipant{}).
		Where("dialog_id = ?", dialogID).
		Update("left_at", time.Now()).Error
}

func (r *ChatRepositoryImpl) IsUserInDialog(db *gorm.DB, dialogID, userID string) (bool, error) {
	var count int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&chat.DialogParticipant{}).
		Where("dialog_id = ? AND user_id = ? AND left_at IS NULL", dialogID, userID).
		Count(&count).Error
	return count > 0, err
}

// Message operations

func (r *ChatRepositoryImpl) CreateMessage(db *gorm.DB, message *chat.Message) error {
	// ✅ Вложенная транзакция удалена.
	// Create message
	// ✅ Используем 'db' из параметра
	if err := db.Create(message).Error; err != nil {
		return err
	}

	// Update dialog's last message and updated_at
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Dialog{}).Where("id = ?", message.DialogID).
		Updates(map[string]interface{}{
			"last_message_id": message.ID,
			"updated_at":      time.Now(),
		}).Error; err != nil {
		return err
	}

	return nil
}

func (r *ChatRepositoryImpl) FindMessageByID(db *gorm.DB, id string) (*chat.Message, error) {
	var message chat.Message
	// ✅ Используем 'db' из параметра
	err := db.Preload("Attachments").Preload("Reactions").Preload("ReadReceipts").
		Preload("ForwardFrom").Preload("ReplyTo").
		First(&message, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

func (r *ChatRepositoryImpl) FindMessagesByDialog(db *gorm.DB, dialogID string, criteria MessageCriteria) ([]chat.Message, int64, error) {
	var messages []chat.Message
	// ✅ Используем 'db' из параметра
	query := db.Preload("Attachments").Preload("Reactions").Preload("ReadReceipts").
		Where("dialog_id = ? AND deleted_at IS NULL", dialogID)

	// Apply filters
	if criteria.BeforeID != "" {
		query = query.Where("id < ?", criteria.BeforeID)
	}

	if criteria.AfterID != "" {
		query = query.Where("id > ?", criteria.AfterID)
	}

	if len(criteria.Types) > 0 {
		query = query.Where("type IN ?", criteria.Types)
	}

	if !criteria.StartDate.IsZero() {
		query = query.Where("created_at >= ?", criteria.StartDate)
	}

	if !criteria.EndDate.IsZero() {
		query = query.Where("created_at <= ?", criteria.EndDate)
	}

	// Get total count
	var total int64
	// ✅ Используем 'db' (query) из параметра
	if err := query.Model(&chat.Message{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	limit := criteria.Limit
	if limit == 0 {
		limit = 50 // default limit
	}

	// ✅ Используем 'db' (query) из параметра
	err := query.Order("created_at DESC").
		Limit(limit).Offset(criteria.Offset).
		Find(&messages).Error

	return messages, total, err
}

func (r *ChatRepositoryImpl) FindLastMessage(db *gorm.DB, dialogID string) (*chat.Message, error) {
	var message chat.Message
	// ✅ Используем 'db' из параметра
	err := db.Where("dialog_id = ? AND deleted_at IS NULL", dialogID).
		Order("created_at DESC").First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

func (r *ChatRepositoryImpl) UpdateMessageStatus(db *gorm.DB, messageID string, status string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&chat.Message{}).Where("id = ?", messageID).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) MarkMessagesAsRead(db *gorm.DB, dialogID, userID string) error {
	// Get unread messages in this dialog
	var unreadMessages []chat.Message
	// ✅ Используем 'db' из параметра
	err := db.Where("dialog_id = ? AND sender_id != ? AND status != ?",
		dialogID, userID, "read").Find(&unreadMessages).Error
	if err != nil {
		return err
	}

	// Create read receipts for all unread messages
	now := time.Now()
	var receipts []chat.MessageReadReceipt
	for _, msg := range unreadMessages {
		receipts = append(receipts, chat.MessageReadReceipt{
			MessageID: msg.ID,
			UserID:    userID,
			ReadAt:    now,
		})
	}

	// ✅ Вложенная транзакция удалена.
	// Create read receipts
	if len(receipts) > 0 {
		// ✅ Используем 'db' из параметра
		if err := db.CreateInBatches(receipts, 50).Error; err != nil {
			return err
		}
	}

	// Update message status
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Message{}).
		Where("dialog_id = ? AND sender_id != ? AND status != ?",
			dialogID, userID, "read").
		Update("status", "read").Error; err != nil {
		return err
	}

	return nil
}

func (r *ChatRepositoryImpl) DeleteMessage(db *gorm.DB, messageID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&chat.Message{}).Where("id = ?", messageID).
		Update("deleted_at", time.Now())

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) DeleteUserMessages(db *gorm.DB, dialogID, userID string) error {
	// ✅ Используем 'db' из параметра
	return db.Model(&chat.Message{}).
		Where("dialog_id = ? AND sender_id = ?", dialogID, userID).
		Update("deleted_at", time.Now()).Error
}

// MessageAttachment operations

func (r *ChatRepositoryImpl) CreateAttachment(db *gorm.DB, attachment *chat.MessageAttachment) error {
	// ✅ Используем 'db' из параметра
	return db.Create(attachment).Error
}

func (r *ChatRepositoryImpl) FindAttachmentsByMessage(db *gorm.DB, messageID string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	// ✅ Используем 'db' из параметра
	err := db.Where("message_id = ?", messageID).Find(&attachments).Error
	return attachments, err
}

func (r *ChatRepositoryImpl) FindAttachmentsByDialog(db *gorm.DB, dialogID string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	// ✅ Используем 'db' из параметра
	err := db.Joins("LEFT JOIN messages ON message_attachments.message_id = messages.id").
		Where("messages.dialog_id = ?", dialogID).
		Find(&attachments).Error
	return attachments, err
}

func (r *ChatRepositoryImpl) DeleteAttachment(db *gorm.DB, attachmentID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", attachmentID).Delete(&chat.MessageAttachment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("attachment not found")
	}
	return nil
}

// MessageReaction operations

func (r *ChatRepositoryImpl) AddReaction(db *gorm.DB, reaction *chat.MessageReaction) error {
	// Check if reaction already exists
	var existing chat.MessageReaction
	// ✅ Используем 'db' из параметра
	if err := db.Where("message_id = ? AND user_id = ?", reaction.MessageID, reaction.UserID).
		First(&existing).Error; err == nil {
		// Update existing reaction
		// ✅ Используем 'db' из параметра
		return db.Model(&existing).Update("emoji", reaction.Emoji).Error
	}

	// ✅ Используем 'db' из параметра
	return db.Create(reaction).Error
}

func (r *ChatRepositoryImpl) FindReactionsByMessage(db *gorm.DB, messageID string) ([]chat.MessageReaction, error) {
	var reactions []chat.MessageReaction
	// ✅ Используем 'db' из параметра
	err := db.Where("message_id = ?", messageID).Find(&reactions).Error
	return reactions, err
}

func (r *ChatRepositoryImpl) FindReaction(db *gorm.DB, messageID, userID string) (*chat.MessageReaction, error) {
	var reaction chat.MessageReaction
	// ✅ Используем 'db' из параметра
	err := db.Where("message_id = ? AND user_id = ?", messageID, userID).First(&reaction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("reaction not found")
		}
		return nil, err
	}
	return &reaction, nil
}

func (r *ChatRepositoryImpl) RemoveReaction(db *gorm.DB, messageID, userID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Where("message_id = ? AND user_id = ?", messageID, userID).
		Delete(&chat.MessageReaction{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("reaction not found")
	}
	return nil
}

func (r *ChatRepositoryImpl) RemoveAllReactions(db *gorm.DB, messageID string) error {
	// ✅ Используем 'db' из параметра
	return db.Where("message_id = ?", messageID).Delete(&chat.MessageReaction{}).Error
}

// MessageReadReceipt operations

func (r *ChatRepositoryImpl) CreateReadReceipt(db *gorm.DB, receipt *chat.MessageReadReceipt) error {
	// ✅ Используем 'db' из параметра
	return db.Create(receipt).Error
}

func (r *ChatRepositoryImpl) FindReadReceiptsByMessage(db *gorm.DB, messageID string) ([]chat.MessageReadReceipt, error) {
	var receipts []chat.MessageReadReceipt
	// ✅ Используем 'db' из параметра
	err := db.Where("message_id = ?", messageID).Find(&receipts).Error
	return receipts, err
}

func (r *ChatRepositoryImpl) FindUnreadMessages(db *gorm.DB, dialogID, userID string) ([]chat.Message, error) {
	var messages []chat.Message
	// ✅ Используем 'db' из параметра
	err := db.Joins("LEFT JOIN message_read_receipts ON messages.id = message_read_receipts.message_id AND message_read_receipts.user_id = ?", userID).
		Where("messages.dialog_id = ? AND messages.sender_id != ? AND message_read_receipts.id IS NULL",
			dialogID, userID).
		Find(&messages).Error
	return messages, err
}

func (r *ChatRepositoryImpl) GetUnreadCount(db *gorm.DB, dialogID, userID string) (int64, error) {
	var count int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&chat.Message{}).
		Joins("LEFT JOIN message_read_receipts ON messages.id = message_read_receipts.message_id AND message_read_receipts.user_id = ?", userID).
		Where("messages.dialog_id = ? AND messages.sender_id != ? AND message_read_receipts.id IS NULL",
			dialogID, userID).
		Count(&count).Error
	return count, err
}

// Combined operations

func (r *ChatRepositoryImpl) CreateCastingDialog(db *gorm.DB, casting *models.Casting, employerID, modelID string) (*chat.Dialog, error) {
	dialog := &chat.Dialog{
		IsGroup:   false,
		Title:     &casting.Title,
		CastingID: &casting.ID,
	}

	// ✅ Вложенная транзакция удалена.
	// Create dialog
	// ✅ Используем 'db' из параметра
	if err := db.Create(dialog).Error; err != nil {
		return nil, err
	}

	// Add participants
	participants := []*chat.DialogParticipant{
		{
			DialogID: dialog.ID,
			UserID:   employerID,
			Role:     "owner",
			JoinedAt: time.Now(),
		},
		{
			DialogID: dialog.ID,
			UserID:   modelID,
			Role:     "member",
			JoinedAt: time.Now(),
		},
	}

	// ✅ Используем 'db' из параметра
	if err := db.CreateInBatches(participants, 2).Error; err != nil {
		return nil, err
	}
	return dialog, nil
}

func (r *ChatRepositoryImpl) SendMessageWithAttachments(db *gorm.DB, senderID, dialogID, content string, attachments []*chat.MessageAttachment) (*chat.Message, error) {
	message := &chat.Message{
		DialogID: dialogID,
		SenderID: senderID,
		Type:     "text",
		Content:  content,
		Status:   "sent",
	}

	// ✅ Вложенная транзакция удалена.
	// Create message
	// ✅ Используем 'db' из параметра
	if err := db.Create(message).Error; err != nil {
		return nil, err
	}

	// Create attachments
	if len(attachments) > 0 {
		for _, attachment := range attachments {
			attachment.MessageID = message.ID
		}
		// ✅ Используем 'db' из параметра
		if err := db.CreateInBatches(attachments, 10).Error; err != nil {
			return nil, err
		}
	}

	// Update dialog
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Dialog{}).Where("id = ?", dialogID).
		Updates(map[string]interface{}{
			"last_message_id": message.ID,
			"updated_at":      time.Now(),
		}).Error; err != nil {
		return nil, err
	}

	return message, nil
}

func (r *ChatRepositoryImpl) GetDialogWithMessages(db *gorm.DB, dialogID string, userID string, criteria MessageCriteria) (*DialogWithMessages, error) {
	// Check if user has access to dialog
	// ✅ 'db' пробрасывается во внутренние вызовы
	hasAccess, err := r.IsUserInDialog(db, dialogID, userID)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrDialogAccessDenied
	}

	// Get dialog
	// ✅ 'db' пробрасывается во внутренние вызовы
	dialog, err := r.FindDialogByID(db, dialogID)
	if err != nil {
		return nil, err
	}

	// Get messages
	// ✅ 'db' пробрасывается во внутренние вызовы
	messages, total, err := r.FindMessagesByDialog(db, dialogID, criteria)
	if err != nil {
		return nil, err
	}

	// Mark messages as read for this user
	// ✅ 'db' пробрасывается во внутренние вызовы
	if err := r.MarkMessagesAsRead(db, dialogID, userID); err != nil {
		// Не возвращаем ошибку, если не удалось отметить как прочитанное,
		// чтобы пользователь все равно получил сообщения
	}

	return &DialogWithMessages{
		Dialog:   dialog,
		Messages: messages,
		Total:    total,
		HasMore:  int64(criteria.Offset+len(messages)) < total,
	}, nil
}

// Admin operations

func (r *ChatRepositoryImpl) FindAllDialogs(db *gorm.DB, criteria DialogCriteria) ([]chat.Dialog, int64, error) {
	var dialogs []chat.Dialog
	// ✅ Используем 'db' из параметра
	query := db.Preload("Participants").Preload("LastMessage")

	// Apply filters
	if criteria.IsGroup != nil {
		query = query.Where("is_group = ?", *criteria.IsGroup)
	}

	if criteria.CastingID != nil {
		query = query.Where("casting_id = ?", *criteria.CastingID)
	}

	if criteria.UserID != "" {
		query = query.Joins("LEFT JOIN dialog_participants ON dialogs.id = dialog_participants.dialog_id").
			Where("dialog_participants.user_id = ?", criteria.UserID)
	}

	if !criteria.StartDate.IsZero() {
		query = query.Where("dialogs.created_at >= ?", criteria.StartDate)
	}

	if !criteria.EndDate.IsZero() {
		query = query.Where("dialogs.created_at <= ?", criteria.EndDate)
	}

	// Get total count
	var total int64
	// ✅ Используем 'db' (query) из параметра
	if err := query.Model(&chat.Dialog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	// ✅ Используем 'db' (query) из параметра
	err := query.Order("dialogs.updated_at DESC").
		Limit(limit).Offset(offset).
		Find(&dialogs).Error

	return dialogs, total, err
}

func (r *ChatRepositoryImpl) GetChatStats(db *gorm.DB) (*ChatStats, error) {
	var stats ChatStats
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))
	weekAgo := now.AddDate(0, 0, -7)

	// Total dialogs
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Dialog{}).Count(&stats.TotalDialogs).Error; err != nil {
		return nil, err
	}

	// Total messages
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Message{}).Count(&stats.TotalMessages).Error; err != nil {
		return nil, err
	}

	// Total attachments
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.MessageAttachment{}).Count(&stats.TotalAttachments).Error; err != nil {
		return nil, err
	}

	// Active dialogs (with messages in last 7 days)
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Dialog{}).
		Where("updated_at >= ?", weekAgo).Count(&stats.ActiveDialogs).Error; err != nil {
		return nil, err
	}

	// Today messages
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Message{}).Where("created_at >= ?", todayStart).
		Count(&stats.TodayMessages).Error; err != nil {
		return nil, err
	}

	// This week messages
	// ✅ Используем 'db' из параметра
	if err := db.Model(&chat.Message{}).Where("created_at >= ?", weekStart).
		Count(&stats.ThisWeekMessages).Error; err != nil {
		return nil, err
	}

	// Message types distribution
	stats.ByType = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}

	// ✅ Используем 'db' из параметра
	err := db.Model(&chat.Message{}).
		Select("type, COUNT(*) as count").
		Group("type").Scan(&typeStats).Error

	if err != nil {
		return nil, err
	}

	for _, ts := range typeStats {
		stats.ByType[ts.Type] = ts.Count
	}

	return &stats, nil
}

func (r *ChatRepositoryImpl) CleanOldMessages(db *gorm.DB, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	// ✅ Вложенная транзакция удалена.
	// Delete read receipts for old messages
	// ✅ Используем 'db' из параметра
	if err := db.Where("message_id IN (SELECT id FROM messages WHERE created_at < ?)", cutoffDate).
		Delete(&chat.MessageReadReceipt{}).Error; err != nil {
		return err
	}

	// Delete reactions for old messages
	// ✅ Используем 'db' из параметра
	if err := db.Where("message_id IN (SELECT id FROM messages WHERE created_at < ?)", cutoffDate).
		Delete(&chat.MessageReaction{}).Error; err != nil {
		return err
	}

	// Delete attachments for old messages
	// ✅ Используем 'db' из параметра
	if err := db.Where("message_id IN (SELECT id FROM messages WHERE created_at < ?)", cutoffDate).
		Delete(&chat.MessageAttachment{}).Error; err != nil {
		return err
	}

	// Delete old messages
	// ✅ Используем 'db' из параметра
	if err := db.Where("created_at < ?", cutoffDate).Delete(&chat.Message{}).Error; err != nil {
		return err
	}

	return nil
}

func (r *ChatRepositoryImpl) UpdateMessage(db *gorm.DB, message *chat.Message) error {
	result := db.Model(&chat.Message{}).Where("id = ?", message.ID).
		Updates(map[string]interface{}{
			"content":    message.Content,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

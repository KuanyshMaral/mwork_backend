package repositories

import (
	"errors"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/models/chat"

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
	CreateDialog(dialog *chat.Dialog) error
	FindDialogByID(id string) (*chat.Dialog, error)
	FindDialogByCasting(castingID string) (*chat.Dialog, error)
	FindUserDialogs(userID string) ([]chat.Dialog, error)
	FindDialogBetweenUsers(user1ID, user2ID string) (*chat.Dialog, error)
	UpdateDialog(dialog *chat.Dialog) error
	DeleteDialog(id string) error
	GetDialogWithParticipants(dialogID string) (*chat.Dialog, error)

	// DialogParticipant operations
	AddParticipant(participant *chat.DialogParticipant) error
	AddParticipants(participants []*chat.DialogParticipant) error
	FindParticipant(dialogID, userID string) (*chat.DialogParticipant, error)
	FindParticipantsByDialog(dialogID string) ([]chat.DialogParticipant, error)
	FindDialogsByUser(userID string) ([]chat.Dialog, error)
	UpdateParticipant(participant *chat.DialogParticipant) error
	UpdateLastSeen(dialogID, userID string, lastSeen time.Time) error
	RemoveParticipant(dialogID, userID string) error
	RemoveAllParticipants(dialogID string) error
	IsUserInDialog(dialogID, userID string) (bool, error)

	// Message operations
	CreateMessage(message *chat.Message) error
	FindMessageByID(id string) (*chat.Message, error)
	FindMessagesByDialog(dialogID string, criteria MessageCriteria) ([]chat.Message, int64, error)
	FindLastMessage(dialogID string) (*chat.Message, error)
	UpdateMessageStatus(messageID string, status string) error
	MarkMessagesAsRead(dialogID, userID string) error
	DeleteMessage(messageID string) error
	DeleteUserMessages(dialogID, userID string) error

	// MessageAttachment operations
	CreateAttachment(attachment *chat.MessageAttachment) error
	FindAttachmentsByMessage(messageID string) ([]chat.MessageAttachment, error)
	FindAttachmentsByDialog(dialogID string) ([]chat.MessageAttachment, error)
	DeleteAttachment(attachmentID string) error

	// MessageReaction operations
	AddReaction(reaction *chat.MessageReaction) error
	FindReactionsByMessage(messageID string) ([]chat.MessageReaction, error)
	FindReaction(messageID, userID string) (*chat.MessageReaction, error)
	RemoveReaction(messageID, userID string) error
	RemoveAllReactions(messageID string) error

	// MessageReadReceipt operations
	CreateReadReceipt(receipt *chat.MessageReadReceipt) error
	FindReadReceiptsByMessage(messageID string) ([]chat.MessageReadReceipt, error)
	FindUnreadMessages(dialogID, userID string) ([]chat.Message, error)
	GetUnreadCount(dialogID, userID string) (int64, error)

	// Combined operations
	CreateCastingDialog(casting *models.Casting, employerID, modelID string) (*chat.Dialog, error)
	SendMessageWithAttachments(senderID, dialogID, content string, attachments []*chat.MessageAttachment) (*chat.Message, error)
	GetDialogWithMessages(dialogID string, userID string, criteria MessageCriteria) (*DialogWithMessages, error)

	// Admin operations
	FindAllDialogs(criteria DialogCriteria) ([]chat.Dialog, int64, error)
	GetChatStats() (*ChatStats, error)
	CleanOldMessages(days int) error
}

type ChatRepositoryImpl struct {
	db *gorm.DB
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

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &ChatRepositoryImpl{db: db}
}

// Dialog operations

func (r *ChatRepositoryImpl) CreateDialog(dialog *chat.Dialog) error {
	// Check if casting dialog already exists
	if dialog.CastingID != nil {
		var existing chat.Dialog
		if err := r.db.Where("casting_id = ?", *dialog.CastingID).First(&existing).Error; err == nil {
			return ErrCastingDialogExists
		}
	}

	return r.db.Create(dialog).Error
}

func (r *ChatRepositoryImpl) FindDialogByID(id string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	err := r.db.Preload("Participants").Preload("LastMessage").
		First(&dialog, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}
	return &dialog, nil
}

func (r *ChatRepositoryImpl) FindDialogByCasting(castingID string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	err := r.db.Preload("Participants").Preload("LastMessage").
		Where("casting_id = ?", castingID).First(&dialog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}
	return &dialog, nil
}

func (r *ChatRepositoryImpl) FindUserDialogs(userID string) ([]chat.Dialog, error) {
	var dialogs []chat.Dialog

	err := r.db.Preload("Participants").Preload("LastMessage").
		Joins("LEFT JOIN dialog_participants ON dialogs.id = dialog_participants.dialog_id").
		Where("dialog_participants.user_id = ? AND dialog_participants.left_at IS NULL", userID).
		Order("dialogs.updated_at DESC").
		Find(&dialogs).Error

	return dialogs, err
}

func (r *ChatRepositoryImpl) FindDialogBetweenUsers(user1ID, user2ID string) (*chat.Dialog, error) {
	var dialog chat.Dialog

	// Find non-group dialog that has exactly these two users
	err := r.db.Preload("Participants").Preload("LastMessage").
		Joins("LEFT JOIN dialog_participants dp1 ON dialogs.id = dp1.dialog_id").
		Joins("LEFT JOIN dialog_participants dp2 ON dialogs.id = dp2.dialog_id").
		Where("dialogs.is_group = ?", false).
		Where("dp1.user_id = ? AND dp2.user_id = ?", user1ID, user2ID).
		Where("dp1.left_at IS NULL AND dp2.left_at IS NULL").
		Group("dialogs.id").
		Having("COUNT(DISTINCT dialog_participants.user_id) = 2").
		First(&dialog).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}

	return &dialog, nil
}

func (r *ChatRepositoryImpl) UpdateDialog(dialog *chat.Dialog) error {
	result := r.db.Model(dialog).Updates(map[string]interface{}{
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

func (r *ChatRepositoryImpl) DeleteDialog(id string) error {
	// Use transaction to delete dialog and all related data
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete messages and related data first
		if err := tx.Where("dialog_id = ?", id).Delete(&chat.MessageReadReceipt{}).Error; err != nil {
			return err
		}
		if err := tx.Where("dialog_id = ?", id).Delete(&chat.MessageReaction{}).Error; err != nil {
			return err
		}
		if err := tx.Where("dialog_id = ?", id).Delete(&chat.MessageAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("dialog_id = ?", id).Delete(&chat.Message{}).Error; err != nil {
			return err
		}

		// Delete participants
		if err := tx.Where("dialog_id = ?", id).Delete(&chat.DialogParticipant{}).Error; err != nil {
			return err
		}

		// Delete dialog
		result := tx.Where("id = ?", id).Delete(&chat.Dialog{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrDialogNotFound
		}
		return nil
	})
}

func (r *ChatRepositoryImpl) GetDialogWithParticipants(dialogID string) (*chat.Dialog, error) {
	var dialog chat.Dialog
	err := r.db.Preload("Participants").First(&dialog, "id = ?", dialogID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDialogNotFound
		}
		return nil, err
	}
	return &dialog, nil
}

// DialogParticipant operations

func (r *ChatRepositoryImpl) AddParticipant(participant *chat.DialogParticipant) error {
	return r.db.Create(participant).Error
}

func (r *ChatRepositoryImpl) AddParticipants(participants []*chat.DialogParticipant) error {
	if len(participants) == 0 {
		return nil
	}
	return r.db.CreateInBatches(participants, 50).Error
}

func (r *ChatRepositoryImpl) FindParticipant(dialogID, userID string) (*chat.DialogParticipant, error) {
	var participant chat.DialogParticipant
	err := r.db.Where("dialog_id = ? AND user_id = ?", dialogID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}
	return &participant, nil
}

func (r *ChatRepositoryImpl) FindParticipantsByDialog(dialogID string) ([]chat.DialogParticipant, error) {
	var participants []chat.DialogParticipant
	err := r.db.Where("dialog_id = ? AND left_at IS NULL", dialogID).Find(&participants).Error
	return participants, err
}

func (r *ChatRepositoryImpl) FindDialogsByUser(userID string) ([]chat.Dialog, error) {
	var dialogs []chat.Dialog
	err := r.db.Preload("LastMessage").
		Joins("LEFT JOIN dialog_participants ON dialogs.id = dialog_participants.dialog_id").
		Where("dialog_participants.user_id = ? AND dialog_participants.left_at IS NULL", userID).
		Order("dialogs.updated_at DESC").
		Find(&dialogs).Error
	return dialogs, err
}

func (r *ChatRepositoryImpl) UpdateParticipant(participant *chat.DialogParticipant) error {
	result := r.db.Model(participant).Updates(map[string]interface{}{
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

func (r *ChatRepositoryImpl) UpdateLastSeen(dialogID, userID string, lastSeen time.Time) error {
	result := r.db.Model(&chat.DialogParticipant{}).
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

func (r *ChatRepositoryImpl) RemoveParticipant(dialogID, userID string) error {
	result := r.db.Model(&chat.DialogParticipant{}).
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

func (r *ChatRepositoryImpl) RemoveAllParticipants(dialogID string) error {
	return r.db.Model(&chat.DialogParticipant{}).
		Where("dialog_id = ?", dialogID).
		Update("left_at", time.Now()).Error
}

func (r *ChatRepositoryImpl) IsUserInDialog(dialogID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&chat.DialogParticipant{}).
		Where("dialog_id = ? AND user_id = ? AND left_at IS NULL", dialogID, userID).
		Count(&count).Error
	return count > 0, err
}

// Message operations

func (r *ChatRepositoryImpl) CreateMessage(message *chat.Message) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create message
		if err := tx.Create(message).Error; err != nil {
			return err
		}

		// Update dialog's last message and updated_at
		if err := tx.Model(&chat.Dialog{}).Where("id = ?", message.DialogID).
			Updates(map[string]interface{}{
				"last_message_id": message.ID,
				"updated_at":      time.Now(),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ChatRepositoryImpl) FindMessageByID(id string) (*chat.Message, error) {
	var message chat.Message
	err := r.db.Preload("Attachments").Preload("Reactions").Preload("ReadReceipts").
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

func (r *ChatRepositoryImpl) FindMessagesByDialog(dialogID string, criteria MessageCriteria) ([]chat.Message, int64, error) {
	var messages []chat.Message
	query := r.db.Preload("Attachments").Preload("Reactions").Preload("ReadReceipts").
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
	if err := query.Model(&chat.Message{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	limit := criteria.Limit
	if limit == 0 {
		limit = 50 // default limit
	}

	err := query.Order("created_at DESC").
		Limit(limit).Offset(criteria.Offset).
		Find(&messages).Error

	return messages, total, err
}

func (r *ChatRepositoryImpl) FindLastMessage(dialogID string) (*chat.Message, error) {
	var message chat.Message
	err := r.db.Where("dialog_id = ? AND deleted_at IS NULL", dialogID).
		Order("created_at DESC").First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

func (r *ChatRepositoryImpl) UpdateMessageStatus(messageID string, status string) error {
	result := r.db.Model(&chat.Message{}).Where("id = ?", messageID).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) MarkMessagesAsRead(dialogID, userID string) error {
	// Get unread messages in this dialog
	var unreadMessages []chat.Message
	err := r.db.Where("dialog_id = ? AND sender_id != ? AND status != ?",
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

	// Use transaction
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create read receipts
		if len(receipts) > 0 {
			if err := tx.CreateInBatches(receipts, 50).Error; err != nil {
				return err
			}
		}

		// Update message status
		if err := tx.Model(&chat.Message{}).
			Where("dialog_id = ? AND sender_id != ? AND status != ?",
				dialogID, userID, "read").
			Update("status", "read").Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ChatRepositoryImpl) DeleteMessage(messageID string) error {
	result := r.db.Model(&chat.Message{}).Where("id = ?", messageID).
		Update("deleted_at", time.Now())

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *ChatRepositoryImpl) DeleteUserMessages(dialogID, userID string) error {
	return r.db.Model(&chat.Message{}).
		Where("dialog_id = ? AND sender_id = ?", dialogID, userID).
		Update("deleted_at", time.Now()).Error
}

// MessageAttachment operations

func (r *ChatRepositoryImpl) CreateAttachment(attachment *chat.MessageAttachment) error {
	return r.db.Create(attachment).Error
}

func (r *ChatRepositoryImpl) FindAttachmentsByMessage(messageID string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	err := r.db.Where("message_id = ?", messageID).Find(&attachments).Error
	return attachments, err
}

func (r *ChatRepositoryImpl) FindAttachmentsByDialog(dialogID string) ([]chat.MessageAttachment, error) {
	var attachments []chat.MessageAttachment
	err := r.db.Joins("LEFT JOIN messages ON message_attachments.message_id = messages.id").
		Where("messages.dialog_id = ?", dialogID).
		Find(&attachments).Error
	return attachments, err
}

func (r *ChatRepositoryImpl) DeleteAttachment(attachmentID string) error {
	result := r.db.Where("id = ?", attachmentID).Delete(&chat.MessageAttachment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("attachment not found")
	}
	return nil
}

// MessageReaction operations

func (r *ChatRepositoryImpl) AddReaction(reaction *chat.MessageReaction) error {
	// Check if reaction already exists
	var existing chat.MessageReaction
	if err := r.db.Where("message_id = ? AND user_id = ?", reaction.MessageID, reaction.UserID).
		First(&existing).Error; err == nil {
		// Update existing reaction
		return r.db.Model(&existing).Update("emoji", reaction.Emoji).Error
	}

	return r.db.Create(reaction).Error
}

func (r *ChatRepositoryImpl) FindReactionsByMessage(messageID string) ([]chat.MessageReaction, error) {
	var reactions []chat.MessageReaction
	err := r.db.Where("message_id = ?", messageID).Find(&reactions).Error
	return reactions, err
}

func (r *ChatRepositoryImpl) FindReaction(messageID, userID string) (*chat.MessageReaction, error) {
	var reaction chat.MessageReaction
	err := r.db.Where("message_id = ? AND user_id = ?", messageID, userID).First(&reaction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("reaction not found")
		}
		return nil, err
	}
	return &reaction, nil
}

func (r *ChatRepositoryImpl) RemoveReaction(messageID, userID string) error {
	result := r.db.Where("message_id = ? AND user_id = ?", messageID, userID).
		Delete(&chat.MessageReaction{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("reaction not found")
	}
	return nil
}

func (r *ChatRepositoryImpl) RemoveAllReactions(messageID string) error {
	return r.db.Where("message_id = ?", messageID).Delete(&chat.MessageReaction{}).Error
}

// MessageReadReceipt operations

func (r *ChatRepositoryImpl) CreateReadReceipt(receipt *chat.MessageReadReceipt) error {
	return r.db.Create(receipt).Error
}

func (r *ChatRepositoryImpl) FindReadReceiptsByMessage(messageID string) ([]chat.MessageReadReceipt, error) {
	var receipts []chat.MessageReadReceipt
	err := r.db.Where("message_id = ?", messageID).Find(&receipts).Error
	return receipts, err
}

func (r *ChatRepositoryImpl) FindUnreadMessages(dialogID, userID string) ([]chat.Message, error) {
	var messages []chat.Message
	err := r.db.Joins("LEFT JOIN message_read_receipts ON messages.id = message_read_receipts.message_id AND message_read_receipts.user_id = ?", userID).
		Where("messages.dialog_id = ? AND messages.sender_id != ? AND message_read_receipts.id IS NULL",
			dialogID, userID).
		Find(&messages).Error
	return messages, err
}

func (r *ChatRepositoryImpl) GetUnreadCount(dialogID, userID string) (int64, error) {
	var count int64
	err := r.db.Model(&chat.Message{}).
		Joins("LEFT JOIN message_read_receipts ON messages.id = message_read_receipts.message_id AND message_read_receipts.user_id = ?", userID).
		Where("messages.dialog_id = ? AND messages.sender_id != ? AND message_read_receipts.id IS NULL",
			dialogID, userID).
		Count(&count).Error
	return count, err
}

// Combined operations

func (r *ChatRepositoryImpl) CreateCastingDialog(casting *models.Casting, employerID, modelID string) (*chat.Dialog, error) {
	dialog := &chat.Dialog{
		IsGroup:   false,
		Title:     &casting.Title,
		CastingID: &casting.ID,
	}

	return dialog, r.db.Transaction(func(tx *gorm.DB) error {
		// Create dialog
		if err := tx.Create(dialog).Error; err != nil {
			return err
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

		return tx.CreateInBatches(participants, 2).Error
	})
}

func (r *ChatRepositoryImpl) SendMessageWithAttachments(senderID, dialogID, content string, attachments []*chat.MessageAttachment) (*chat.Message, error) {
	message := &chat.Message{
		DialogID: dialogID,
		SenderID: senderID,
		Type:     "text",
		Content:  content,
		Status:   "sent",
	}

	return message, r.db.Transaction(func(tx *gorm.DB) error {
		// Create message
		if err := tx.Create(message).Error; err != nil {
			return err
		}

		// Create attachments
		if len(attachments) > 0 {
			for _, attachment := range attachments {
				attachment.MessageID = message.ID
			}
			if err := tx.CreateInBatches(attachments, 10).Error; err != nil {
				return err
			}
		}

		// Update dialog
		if err := tx.Model(&chat.Dialog{}).Where("id = ?", dialogID).
			Updates(map[string]interface{}{
				"last_message_id": message.ID,
				"updated_at":      time.Now(),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ChatRepositoryImpl) GetDialogWithMessages(dialogID string, userID string, criteria MessageCriteria) (*DialogWithMessages, error) {
	// Check if user has access to dialog
	hasAccess, err := r.IsUserInDialog(dialogID, userID)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrDialogAccessDenied
	}

	// Get dialog
	dialog, err := r.FindDialogByID(dialogID)
	if err != nil {
		return nil, err
	}

	// Get messages
	messages, total, err := r.FindMessagesByDialog(dialogID, criteria)
	if err != nil {
		return nil, err
	}

	// Mark messages as read for this user
	if err := r.MarkMessagesAsRead(dialogID, userID); err != nil {
		return nil, err
	}

	return &DialogWithMessages{
		Dialog:   dialog,
		Messages: messages,
		Total:    total,
		HasMore:  int64(criteria.Offset+len(messages)) < total,
	}, nil
}

// Admin operations

func (r *ChatRepositoryImpl) FindAllDialogs(criteria DialogCriteria) ([]chat.Dialog, int64, error) {
	var dialogs []chat.Dialog
	query := r.db.Preload("Participants").Preload("LastMessage")

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
	if err := query.Model(&chat.Dialog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Order("dialogs.updated_at DESC").
		Limit(limit).Offset(offset).
		Find(&dialogs).Error

	return dialogs, total, err
}

func (r *ChatRepositoryImpl) GetChatStats() (*ChatStats, error) {
	var stats ChatStats
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))
	weekAgo := now.AddDate(0, 0, -7)

	// Total dialogs
	if err := r.db.Model(&chat.Dialog{}).Count(&stats.TotalDialogs).Error; err != nil {
		return nil, err
	}

	// Total messages
	if err := r.db.Model(&chat.Message{}).Count(&stats.TotalMessages).Error; err != nil {
		return nil, err
	}

	// Total attachments
	if err := r.db.Model(&chat.MessageAttachment{}).Count(&stats.TotalAttachments).Error; err != nil {
		return nil, err
	}

	// Active dialogs (with messages in last 7 days)
	if err := r.db.Model(&chat.Dialog{}).
		Where("updated_at >= ?", weekAgo).Count(&stats.ActiveDialogs).Error; err != nil {
		return nil, err
	}

	// Today messages
	if err := r.db.Model(&chat.Message{}).Where("created_at >= ?", todayStart).
		Count(&stats.TodayMessages).Error; err != nil {
		return nil, err
	}

	// This week messages
	if err := r.db.Model(&chat.Message{}).Where("created_at >= ?", weekStart).
		Count(&stats.ThisWeekMessages).Error; err != nil {
		return nil, err
	}

	// Message types distribution
	stats.ByType = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}

	err := r.db.Model(&chat.Message{}).
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

func (r *ChatRepositoryImpl) CleanOldMessages(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete read receipts for old messages
		if err := tx.Where("message_id IN (SELECT id FROM messages WHERE created_at < ?)", cutoffDate).
			Delete(&chat.MessageReadReceipt{}).Error; err != nil {
			return err
		}

		// Delete reactions for old messages
		if err := tx.Where("message_id IN (SELECT id FROM messages WHERE created_at < ?)", cutoffDate).
			Delete(&chat.MessageReaction{}).Error; err != nil {
			return err
		}

		// Delete attachments for old messages
		if err := tx.Where("message_id IN (SELECT id FROM messages WHERE created_at < ?)", cutoffDate).
			Delete(&chat.MessageAttachment{}).Error; err != nil {
			return err
		}

		// Delete old messages
		if err := tx.Where("created_at < ?", cutoffDate).Delete(&chat.Message{}).Error; err != nil {
			return err
		}

		return nil
	})
}

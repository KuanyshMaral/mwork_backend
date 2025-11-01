package services

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/models/chat"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type ChatService interface {
	// Dialog operations
	CreateDialog(db *gorm.DB, userID string, req *dto.CreateDialogRequest) (*dto.DialogResponse, error)
	CreateCastingDialog(db *gorm.DB, castingID, employerID, modelID string) (*dto.DialogResponse, error)
	GetDialog(db *gorm.DB, dialogID, userID string) (*dto.DialogResponse, error)
	GetUserDialogs(db *gorm.DB, userID string) ([]*dto.DialogResponse, error)
	GetDialogBetweenUsers(db *gorm.DB, user1ID, user2ID string) (*dto.DialogResponse, error)
	UpdateDialog(db *gorm.DB, userID, dialogID string, req *dto.UpdateDialogRequest) error
	DeleteDialog(db *gorm.DB, userID, dialogID string) error
	LeaveDialog(db *gorm.DB, userID, dialogID string) error

	// Participant operations
	AddParticipants(db *gorm.DB, userID, dialogID string, participantIDs []string) error
	RemoveParticipant(db *gorm.DB, userID, dialogID, targetUserID string) error
	UpdateParticipantRole(db *gorm.DB, userID, dialogID, targetUserID, role string) error
	MuteDialog(db *gorm.DB, userID, dialogID string, muted bool) error
	UpdateLastSeen(db *gorm.DB, userID, dialogID string) error
	SetTyping(db *gorm.DB, userID, dialogID string, typing bool) error

	// Message operations
	SendMessage(db *gorm.DB, userID string, req *dto.SendMessageRequest) (*dto.MessageResponse, error)
	SendMessageWithAttachments(db *gorm.DB, userID string, req *dto.SendMessageRequest, files []*multipart.FileHeader) (*dto.MessageResponse, error)
	GetMessages(db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.MessageListResponse, error)
	GetMessage(db *gorm.DB, messageID, userID string) (*dto.MessageResponse, error)
	UpdateMessage(db *gorm.DB, userID, messageID string, req *dto.UpdateMessageRequest) error
	DeleteMessage(db *gorm.DB, userID, messageID string) error
	ForwardMessage(db *gorm.DB, userID string, req *dto.ForwardMessageRequest) (*dto.MessageResponse, error)

	// Attachment operations
	UploadAttachment(db *gorm.DB, userID string, file *multipart.FileHeader) (*dto.AttachmentResponse, error)
	GetMessageAttachments(db *gorm.DB, messageID, userID string) ([]*dto.AttachmentResponse, error)
	GetDialogAttachments(db *gorm.DB, dialogID, userID string) ([]*dto.AttachmentResponse, error)
	DeleteAttachment(db *gorm.DB, userID, attachmentID string) error

	// Reaction operations
	AddReaction(db *gorm.DB, userID, messageID, emoji string) error
	RemoveReaction(db *gorm.DB, userID, messageID string) error
	GetMessageReactions(db *gorm.DB, messageID, userID string) ([]*dto.ReactionResponse, error)

	// Read receipts
	MarkMessagesAsRead(db *gorm.DB, userID, dialogID string) error
	GetUnreadCount(db *gorm.DB, dialogID, userID string) (int64, error)
	GetReadReceipts(db *gorm.DB, messageID, userID string) ([]*dto.ReadReceiptResponse, error)

	// Combined operations
	GetDialogWithMessages(db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.DialogWithMessagesResponse, error)
	SearchMessages(db *gorm.DB, userID, dialogID, query string) ([]*dto.MessageResponse, error)

	// Admin operations
	GetAllDialogs(db *gorm.DB, criteria dto.DialogCriteria) (*dto.DialogListResponse, error)
	GetChatStats(db *gorm.DB) (*repositories.ChatStats, error)
	CleanOldMessages(db *gorm.DB, days int) error
	DeleteUserMessages(db *gorm.DB, adminID, dialogID string, userID string) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type chatService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	chatRepo         repositories.ChatRepository
	userRepo         repositories.UserRepository
	castingRepo      repositories.CastingRepository
	responseRepo     repositories.ResponseRepository
	profileRepo      repositories.ProfileRepository
	notificationRepo repositories.NotificationRepository
}

// ✅ Конструктор обновлен (db убран)
func NewChatService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	chatRepo repositories.ChatRepository,
	userRepo repositories.UserRepository,
	castingRepo repositories.CastingRepository,
	profileRepo repositories.ProfileRepository,
	notificationRepo repositories.NotificationRepository,
	responseRepo repositories.ResponseRepository,
) ChatService {
	return &chatService{
		// ❌ 'db: db,' УДАЛЕНО
		chatRepo:         chatRepo,
		userRepo:         userRepo,
		castingRepo:      castingRepo,
		profileRepo:      profileRepo,
		notificationRepo: notificationRepo,
		responseRepo:     responseRepo,
	}
}

var FileConfig = dto.FileConfig{
	MaxSize: 50 * 1024 * 1024, // 50MB
	AllowedTypes: []string{
		"image/jpeg", "image/png", "image/gif",
		"video/mp4", "video/avi", "video/mov",
		"application/pdf", "application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	},
	StoragePath: "./uploads/chat",
}

// Dialog operations

// CreateDialog - 'db' добавлен
func (s *chatService) CreateDialog(db *gorm.DB, userID string, req *dto.CreateDialogRequest) (*dto.DialogResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	for _, participantID := range req.UserIDs {
		// ✅ Передаем tx
		if _, err := s.userRepo.FindByID(tx, participantID); err != nil {
			return nil, fmt.Errorf("user not found: %s", participantID)
		}
	}

	participants := append([]string{userID}, req.UserIDs...)
	uniqueParticipants := removeDuplicates(participants)

	dialog := &chat.Dialog{
		IsGroup:   req.IsGroup,
		Title:     req.Title,
		ImageURL:  req.ImageURL,
		CastingID: req.CastingID,
	}

	// ✅ Передаем tx
	if err := s.chatRepo.CreateDialog(tx, dialog); err != nil {
		return nil, apperrors.InternalError(err)
	}

	var chatParticipants []*chat.DialogParticipant
	for i, participantID := range uniqueParticipants {
		role := "member"
		if i == 0 {
			role = "owner"
		}
		chatParticipants = append(chatParticipants, &chat.DialogParticipant{
			DialogID: dialog.ID,
			UserID:   participantID,
			Role:     role,
			JoinedAt: time.Now(),
		})
	}

	// ✅ Передаем tx
	if err := s.chatRepo.AddParticipants(tx, chatParticipants); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Строим ответ *вне* транзакции (read-only), передаем 'db' (пул)
	return s.buildDialogResponse(db, dialog, userID)
}

// CreateCastingDialog - 'db' добавлен
func (s *chatService) CreateCastingDialog(db *gorm.DB, castingID, employerID, modelID string) (*dto.DialogResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return nil, apperrors.ErrDialogNotFound // (или handleCastingError)
	}

	// ✅ Передаем tx
	dialog, err := s.chatRepo.CreateCastingDialog(tx, casting, employerID, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	systemMessage := &chat.Message{
		DialogID: dialog.ID,
		SenderID: "system",
		Type:     "system",
		Content:  fmt.Sprintf("Чат создан для кастинга '%s'", casting.Title),
		Status:   "sent",
	}

	// ✅ Передаем tx
	if err := s.chatRepo.CreateMessage(tx, systemMessage); err != nil {
		fmt.Printf("Failed to create system message: %v\n", err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Строим ответ *вне* транзакции (read-only), передаем 'db' (пул)
	return s.buildDialogResponse(db, dialog, employerID)
}

// GetDialog - 'db' добавлен
func (s *chatService) GetDialog(db *gorm.DB, dialogID, userID string) (*dto.DialogResponse, error) {
	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	dialog, err := s.chatRepo.FindDialogByID(db, dialogID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Используем 'db' из параметра
	return s.buildDialogResponse(db, dialog, userID)
}

// GetUserDialogs - 'db' добавлен
func (s *chatService) GetUserDialogs(db *gorm.DB, userID string) ([]*dto.DialogResponse, error) {
	// ✅ Используем 'db' из параметра
	dialogs, err := s.chatRepo.FindUserDialogs(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.DialogResponse
	for _, dialog := range dialogs {
		// ✅ Используем 'db' из параметра
		response, err := s.buildDialogResponse(db, &dialog, userID)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// GetDialogBetweenUsers - 'db' добавлен
func (s *chatService) GetDialogBetweenUsers(db *gorm.DB, user1ID, user2ID string) (*dto.DialogResponse, error) {
	// ✅ Используем 'db' из параметра
	dialog, err := s.chatRepo.FindDialogBetweenUsers(db, user1ID, user2ID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Используем 'db' из параметра
	return s.buildDialogResponse(db, dialog, user1ID)
}

// UpdateDialog - 'db' добавлен
func (s *chatService) UpdateDialog(db *gorm.DB, userID, dialogID string, req *dto.UpdateDialogRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}

	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}

	// ✅ Передаем tx
	dialog, err := s.chatRepo.FindDialogByID(tx, dialogID)
	if err != nil {
		return handleChatError(err)
	}

	if req.Title != nil {
		dialog.Title = req.Title
	}
	if req.ImageURL != nil {
		dialog.ImageURL = req.ImageURL
	}

	// ✅ Передаем tx
	if err := s.chatRepo.UpdateDialog(tx, dialog); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteDialog - 'db' добавлен
func (s *chatService) DeleteDialog(db *gorm.DB, userID, dialogID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" {
		return errors.New("only dialog owner can delete dialog")
	}

	// ✅ Передаем tx
	if err := s.chatRepo.DeleteDialog(tx, dialogID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// LeaveDialog - 'db' добавлен
func (s *chatService) LeaveDialog(db *gorm.DB, userID, dialogID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.chatRepo.RemoveParticipant(tx, dialogID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Participant operations

// AddParticipants - 'db' добавлен
func (s *chatService) AddParticipants(db *gorm.DB, userID, dialogID string, participantIDs []string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}

	for _, participantID := range participantIDs {
		// ✅ Передаем tx
		if _, err := s.userRepo.FindByID(tx, participantID); err != nil {
			return fmt.Errorf("user not found: %s", participantID)
		}
	}

	var participants []*chat.DialogParticipant
	for _, participantID := range participantIDs {
		participants = append(participants, &chat.DialogParticipant{
			DialogID: dialogID,
			UserID:   participantID,
			Role:     "member",
			JoinedAt: time.Now(),
		})
	}

	// ✅ Передаем tx
	if err := s.chatRepo.AddParticipants(tx, participants); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// RemoveParticipant - 'db' добавлен
func (s *chatService) RemoveParticipant(db *gorm.DB, userID, dialogID, targetUserID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	// ✅ Передаем tx
	targetParticipant, err := s.chatRepo.FindParticipant(tx, dialogID, targetUserID)
	if err != nil {
		return apperrors.ErrParticipantNotFound
	}

	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}
	if targetParticipant.Role == "owner" {
		return errors.New("cannot remove dialog owner")
	}

	// ✅ Передаем tx
	if err := s.chatRepo.RemoveParticipant(tx, dialogID, targetUserID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// UpdateParticipantRole - 'db' добавлен
func (s *chatService) UpdateParticipantRole(db *gorm.DB, userID, dialogID, targetUserID, role string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" {
		return errors.New("only owner can update roles")
	}

	// ✅ Передаем tx
	targetParticipant, err := s.chatRepo.FindParticipant(tx, dialogID, targetUserID)
	if err != nil {
		return apperrors.ErrParticipantNotFound
	}

	targetParticipant.Role = role
	// ✅ Передаем tx
	if err := s.chatRepo.UpdateParticipant(tx, targetParticipant); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// MuteDialog - 'db' добавлен
func (s *chatService) MuteDialog(db *gorm.DB, userID, dialogID string, muted bool) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}

	participant.IsMuted = muted
	// ✅ Передаем tx
	if err := s.chatRepo.UpdateParticipant(tx, participant); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// UpdateLastSeen - 'db' добавлен
func (s *chatService) UpdateLastSeen(db *gorm.DB, userID, dialogID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.chatRepo.UpdateLastSeen(tx, dialogID, userID, time.Now()); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// SetTyping - 'db' добавлен
func (s *chatService) SetTyping(db *gorm.DB, userID, dialogID string, typing bool) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}

	if typing {
		typingUntil := time.Now().Add(10 * time.Second)
		participant.TypingUntil = &typingUntil
	} else {
		participant.TypingUntil = nil
	}

	// ✅ Передаем tx
	if err := s.chatRepo.UpdateParticipant(tx, participant); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Message operations

// SendMessage - 'db' добавлен
func (s *chatService) SendMessage(db *gorm.DB, userID string, req *dto.SendMessageRequest) (*dto.MessageResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, req.DialogID, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	if !isValidMessageType(req.Type) {
		return nil, apperrors.ErrInvalidMessageType
	}

	message := &chat.Message{
		DialogID:      req.DialogID,
		SenderID:      userID,
		Type:          req.Type,
		Content:       req.Content,
		ReplyToID:     req.ReplyToID,
		ForwardFromID: req.ForwardFromID,
		Status:        "sent",
	}

	// ✅ Передаем tx
	if err := s.chatRepo.CreateMessage(tx, message); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Уведомляем *после* коммита, передаем 'db' (пул)
	go s.notifyNewMessage(db, req.DialogID, userID, message.ID)

	// ✅ Строим ответ *вне* транзакции (read-only), передаем 'db' (пул)
	return s.buildMessageResponse(db, message)
}

// SendMessageWithAttachments - 'db' добавлен
func (s *chatService) SendMessageWithAttachments(db *gorm.DB, userID string, req *dto.SendMessageRequest, files []*multipart.FileHeader) (*dto.MessageResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, req.DialogID, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	if !isValidMessageType(req.Type) {
		return nil, apperrors.ErrInvalidMessageType
	}

	message := &chat.Message{
		DialogID:      req.DialogID,
		SenderID:      userID,
		Type:          req.Type,
		Content:       req.Content,
		ReplyToID:     req.ReplyToID,
		ForwardFromID: req.ForwardFromID,
		Status:        "sent",
	}

	// ✅ Передаем tx
	if err := s.chatRepo.CreateMessage(tx, message); err != nil {
		return nil, apperrors.InternalError(err)
	}

	var attachments []*chat.MessageAttachment
	for _, file := range files {
		attachment, err := s.processAttachment(userID, file)
		if err != nil {
			log.Printf("Skipping problematic attachment: %v", err)
			continue
		}
		attachment.MessageID = message.ID
		attachments = append(attachments, attachment)
	}

	for _, attachment := range attachments {
		// ✅ Передаем tx
		if err := s.chatRepo.CreateAttachment(tx, attachment); err != nil {
			log.Printf("Failed to create attachment: %v\n", err)
			return nil, apperrors.InternalError(err)
		}
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Уведомляем *после* коммита, передаем 'db' (пул)
	go s.notifyNewMessage(db, req.DialogID, userID, message.ID)

	// ✅ Получаем сообщение *вне* транзакции (read-only), передаем 'db' (пул)
	return s.GetMessage(db, message.ID, userID)
}

// GetMessages - 'db' добавлен
func (s *chatService) GetMessages(db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.MessageListResponse, error) {
	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	messages, total, err := s.chatRepo.FindMessagesByDialog(db, dialogID, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var messageResponses []*dto.MessageResponse
	for _, message := range messages {
		// ✅ Используем 'db' из параметра
		messageResponse, err := s.buildMessageResponse(db, &message)
		if err != nil {
			continue
		}
		messageResponses = append(messageResponses, messageResponse)
	}

	pageSize := criteria.Limit
	if pageSize <= 0 {
		pageSize = 10
	}
	page := 1
	if criteria.Offset > 0 && pageSize > 0 {
		page = (criteria.Offset / pageSize) + 1
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = calculateTotalPages(total, pageSize)
	} else if total > 0 {
		totalPages = 1
	}

	return &dto.MessageListResponse{
		Messages:   messageResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    int64(criteria.Offset+len(messages)) < total,
	}, nil
}

// GetMessage - 'db' добавлен
func (s *chatService) GetMessage(db *gorm.DB, messageID, userID string) (*dto.MessageResponse, error) {
	// ✅ Используем 'db' из параметра
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	return s.buildMessageResponse(db, message)
}

// UpdateMessage - 'db' добавлен
func (s *chatService) UpdateMessage(db *gorm.DB, userID, messageID string, req *dto.UpdateMessageRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	message, err := s.chatRepo.FindMessageByID(tx, messageID)
	if err != nil {
		return handleChatError(err)
	}

	if message.SenderID != userID {
		return errors.New("can only edit own messages")
	}
	if time.Since(message.CreatedAt) > 15*time.Minute {
		return errors.New("message can only be edited within 15 minutes")
	}

	message.Content = req.Content
	// ✅ Передаем tx
	if err := s.chatRepo.UpdateMessage(tx, message); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteMessage - 'db' добавлен
func (s *chatService) DeleteMessage(db *gorm.DB, userID, messageID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	message, err := s.chatRepo.FindMessageByID(tx, messageID)
	if err != nil {
		return handleChatError(err)
	}

	// ✅ Передаем tx
	participant, err := s.chatRepo.FindParticipant(tx, message.DialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}

	if message.SenderID != userID && participant.Role != "owner" && participant.Role != "admin" {
		return apperrors.ErrCannotDeleteMessage
	}

	// ✅ Передаем tx
	if err := s.chatRepo.DeleteMessage(tx, messageID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ForwardMessage - 'db' добавлен
func (s *chatService) ForwardMessage(db *gorm.DB, userID string, req *dto.ForwardMessageRequest) (*dto.MessageResponse, error) {
	// ✅ Используем 'db' из параметра
	originalMessage, err := s.chatRepo.FindMessageByID(db, req.MessageID)
	if err != nil {
		return nil, handleChatError(err)
	}

	var forwardedMessage *dto.MessageResponse
	for _, dialogID := range req.DialogIDs {
		// ✅ Используем 'db' из параметра
		hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
		if err != nil || !hasAccess {
			continue
		}

		forwardReq := &dto.SendMessageRequest{
			DialogID:      dialogID,
			Type:          "forward",
			Content:       originalMessage.Content,
			ForwardFromID: &originalMessage.ID,
		}

		// ✅ Передаем 'db'
		forwardedMessage, err = s.SendMessage(db, userID, forwardReq)
		if err != nil {
			fmt.Printf("Failed to forward message to dialog %s: %v\n", dialogID, err)
		}
	}

	return forwardedMessage, nil
}

// Attachment operations

// UploadAttachment - 'db' добавлен
func (s *chatService) UploadAttachment(db *gorm.DB, userID string, file *multipart.FileHeader) (*dto.AttachmentResponse, error) {
	attachment, err := s.processAttachment(userID, file)
	if err != nil {
		return nil, err
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.chatRepo.CreateAttachment(tx, attachment); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	return s.buildAttachmentResponse(attachment), nil
}

// GetMessageAttachments - 'db' добавлен
func (s *chatService) GetMessageAttachments(db *gorm.DB, messageID, userID string) ([]*dto.AttachmentResponse, error) {
	// ✅ Используем 'db' из параметра
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	attachments, err := s.chatRepo.FindAttachmentsByMessage(db, messageID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.AttachmentResponse
	for _, attachment := range attachments {
		responses = append(responses, s.buildAttachmentResponse(&attachment))
	}

	return responses, nil
}

// GetDialogAttachments - 'db' добавлен
func (s *chatService) GetDialogAttachments(db *gorm.DB, dialogID, userID string) ([]*dto.AttachmentResponse, error) {
	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	attachments, err := s.chatRepo.FindAttachmentsByDialog(db, dialogID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.AttachmentResponse
	for _, attachment := range attachments {
		responses = append(responses, s.buildAttachmentResponse(&attachment))
	}

	return responses, nil
}

// DeleteAttachment - 'db' добавлен
func (s *chatService) DeleteAttachment(db *gorm.DB, userID, attachmentID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// TODO: Добавить проверку прав

	// ✅ Передаем tx
	if err := s.chatRepo.DeleteAttachment(tx, attachmentID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Reaction operations

// AddReaction - 'db' добавлен
func (s *chatService) AddReaction(db *gorm.DB, userID, messageID, emoji string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	message, err := s.chatRepo.FindMessageByID(tx, messageID)
	if err != nil {
		return handleChatError(err)
	}

	// ✅ Передаем tx
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, message.DialogID, userID)
	if err != nil || !hasAccess {
		return apperrors.ErrDialogAccessDenied
	}

	reaction := &chat.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}

	// ✅ Передаем tx
	if err := s.chatRepo.AddReaction(tx, reaction); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// RemoveReaction - 'db' добавлен
func (s *chatService) RemoveReaction(db *gorm.DB, userID, messageID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.chatRepo.RemoveReaction(tx, messageID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetMessageReactions - 'db' добавлен
func (s *chatService) GetMessageReactions(db *gorm.DB, messageID, userID string) ([]*dto.ReactionResponse, error) {
	// ✅ Используем 'db' из параметра
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	reactions, err := s.chatRepo.FindReactionsByMessage(db, messageID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.ReactionResponse
	for _, reaction := range reactions {
		// ✅ Используем 'db' из параметра
		user, err := s.userRepo.FindByID(db, reaction.UserID)
		if err != nil {
			continue
		}

		responses = append(responses, &dto.ReactionResponse{
			ID:        reaction.ID,
			UserID:    reaction.UserID,
			UserName:  user.Email,
			Emoji:     reaction.Emoji,
			CreatedAt: reaction.CreatedAt,
		})
	}

	return responses, nil
}

// Read receipts

// MarkMessagesAsRead - 'db' добавлен
func (s *chatService) MarkMessagesAsRead(db *gorm.DB, userID, dialogID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, dialogID, userID)
	if err != nil || !hasAccess {
		return apperrors.ErrDialogAccessDenied
	}

	// ✅ Передаем tx
	if err := s.chatRepo.MarkMessagesAsRead(tx, dialogID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetUnreadCount - 'db' добавлен
func (s *chatService) GetUnreadCount(db *gorm.DB, dialogID, userID string) (int64, error) {
	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil || !hasAccess {
		return 0, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	return s.chatRepo.GetUnreadCount(db, dialogID, userID)
}

// GetReadReceipts - 'db' добавлен
func (s *chatService) GetReadReceipts(db *gorm.DB, messageID, userID string) ([]*dto.ReadReceiptResponse, error) {
	// ✅ Используем 'db' из параметра
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Используем 'db' из параметра
	receipts, err := s.chatRepo.FindReadReceiptsByMessage(db, messageID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.ReadReceiptResponse
	for _, receipt := range receipts {
		// ✅ Используем 'db' из параметра
		user, err := s.userRepo.FindByID(db, receipt.UserID)
		if err != nil {
			continue
		}

		responses = append(responses, &dto.ReadReceiptResponse{
			UserID:   receipt.UserID,
			UserName: user.Email,
			ReadAt:   receipt.ReadAt,
		})
	}

	return responses, nil
}

// Combined operations

// GetDialogWithMessages - 'db' добавлен
func (s *chatService) GetDialogWithMessages(db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.DialogWithMessagesResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, dialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	// ✅ Передаем tx
	dialog, err := s.chatRepo.FindDialogByID(tx, dialogID)
	if err != nil {
		return nil, handleChatError(err)
	}

	// ✅ Передаем tx
	messages, total, err := s.chatRepo.FindMessagesByDialog(tx, dialogID, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	if err := s.chatRepo.MarkMessagesAsRead(tx, dialogID, userID); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	dialogResponse, err := s.buildDialogResponse(tx, dialog, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var messageResponses []*dto.MessageResponse
	for _, message := range messages {
		// ✅ Передаем tx
		messageResponse, err := s.buildMessageResponse(tx, &message)
		if err != nil {
			continue
		}
		messageResponses = append(messageResponses, messageResponse)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	pageSize := criteria.Limit
	if pageSize <= 0 {
		pageSize = 10
	}
	page := 1
	if criteria.Offset > 0 && pageSize > 0 {
		page = (criteria.Offset / pageSize) + 1
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = calculateTotalPages(total, pageSize)
	} else if total > 0 {
		totalPages = 1
	}

	return &dto.DialogWithMessagesResponse{
		Dialog:     dialogResponse,
		Messages:   messageResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    int64(criteria.Offset+len(messages)) < total,
	}, nil
}

// SearchMessages - 'db' добавлен
func (s *chatService) SearchMessages(db *gorm.DB, userID, dialogID, query string) ([]*dto.MessageResponse, error) {
	// ✅ Используем 'db' из параметра
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	criteria := dto.MessageCriteria{
		Limit: 1000,
	}

	// ✅ Используем 'db' из параметра
	messages, _, err := s.chatRepo.FindMessagesByDialog(db, dialogID, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var results []*dto.MessageResponse
	for _, message := range messages {
		if strings.Contains(strings.ToLower(message.Content), strings.ToLower(query)) {
			// ✅ Используем 'db' из параметра
			messageResponse, err := s.buildMessageResponse(db, &message)
			if err != nil {
				log.Printf("Failed to build message response for message %s: %v", message.ID, err)
				continue
			}
			results = append(results, messageResponse)
		}
	}

	return results, nil
}

// Admin operations

// GetAllDialogs - 'db' добавлен
func (s *chatService) GetAllDialogs(db *gorm.DB, criteria dto.DialogCriteria) (*dto.DialogListResponse, error) {
	// ✅ Используем 'db' из параметра
	dialogs, total, err := s.chatRepo.FindAllDialogs(db, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.DialogResponse
	for _, dialog := range dialogs {
		response := &dto.DialogResponse{
			ID:        dialog.ID,
			IsGroup:   dialog.IsGroup,
			Title:     dialog.Title,
			ImageURL:  dialog.ImageURL,
			CastingID: dialog.CastingID,
			CreatedAt: dialog.CreatedAt,
			UpdatedAt: dialog.UpdatedAt,
		}
		responses = append(responses, response)
	}

	pageSize := criteria.PageSize
	page := criteria.Page
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = calculateTotalPages(total, pageSize)
	} else if total > 0 {
		totalPages = 1
	}

	return &dto.DialogListResponse{
		Dialogs:    responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetChatStats - 'db' добавлен
func (s *chatService) GetChatStats(db *gorm.DB) (*repositories.ChatStats, error) {
	// ✅ Используем 'db' из параметра
	return s.chatRepo.GetChatStats(db)
}

// CleanOldMessages - 'db' добавлен
func (s *chatService) CleanOldMessages(db *gorm.DB, days int) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.chatRepo.CleanOldMessages(tx, days); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteUserMessages - 'db' добавлен
func (s *chatService) DeleteUserMessages(db *gorm.DB, adminID, dialogID string, userID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleChatError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}

	// ✅ Передаем tx
	if err := s.chatRepo.DeleteUserMessages(tx, dialogID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Helper methods

// buildDialogResponse - 'db' добавлен
func (s *chatService) buildDialogResponse(db *gorm.DB, dialog *chat.Dialog, userID string) (*dto.DialogResponse, error) {
	response := &dto.DialogResponse{
		ID:        dialog.ID,
		IsGroup:   dialog.IsGroup,
		Title:     dialog.Title,
		ImageURL:  dialog.ImageURL,
		CastingID: dialog.CastingID,
		CreatedAt: dialog.CreatedAt,
		UpdatedAt: dialog.UpdatedAt,
	}

	var participantResponses []*dto.ParticipantResponse
	for _, participant := range dialog.Participants {
		// ✅ Используем 'db' из параметра
		user, err := s.userRepo.FindByID(db, participant.UserID)
		if err != nil {
			continue
		}
		participantResponses = append(participantResponses, &dto.ParticipantResponse{
			UserID:     participant.UserID,
			UserName:   user.Email,
			Role:       participant.Role,
			LastSeenAt: participant.LastSeenAt,
			IsMuted:    participant.IsMuted,
			IsOnline:   false,
		})
	}
	response.Participants = participantResponses

	if dialog.LastMessage != nil {
		// ✅ Используем 'db' из параметра
		lastMessage, err := s.buildMessageResponse(db, dialog.LastMessage)
		if err == nil {
			response.LastMessage = lastMessage
		}
	}

	// ✅ Используем 'db' из параметра
	unreadCount, err := s.chatRepo.GetUnreadCount(db, dialog.ID, userID)
	if err == nil {
		response.UnreadCount = unreadCount
	}

	for _, participant := range dialog.Participants {
		if participant.UserID == userID {
			response.IsMuted = participant.IsMuted
			break
		}
	}

	return response, nil
}

// buildMessageResponse - 'db' добавлен
func (s *chatService) buildMessageResponse(db *gorm.DB, message *chat.Message) (*dto.MessageResponse, error) {
	response := &dto.MessageResponse{
		ID:             message.ID,
		DialogID:       message.DialogID,
		SenderID:       message.SenderID,
		Type:           message.Type,
		Content:        message.Content,
		AttachmentURL:  message.AttachmentURL,
		AttachmentName: message.AttachmentName,
		Status:         message.Status,
		CreatedAt:      message.CreatedAt,
	}

	// ✅ Используем 'db' из параметра
	sender, err := s.userRepo.FindByID(db, message.SenderID)
	if err == nil {
		response.SenderName = sender.Email
	}

	if message.ReplyTo != nil {
		// ✅ Используем 'db' из параметра
		replyTo, err := s.buildMessageResponse(db, message.ReplyTo)
		if err == nil {
			response.ReplyTo = replyTo
		}
	}

	if message.ForwardFrom != nil {
		// ✅ Используем 'db' из параметра
		forwardFrom, err := s.buildMessageResponse(db, message.ForwardFrom)
		if err == nil {
			response.ForwardFrom = forwardFrom
		}
	}

	var reactionResponses []*dto.ReactionResponse
	for _, reaction := range message.Reactions {
		// ✅ Используем 'db' из параметра
		user, err := s.userRepo.FindByID(db, reaction.UserID)
		if err != nil {
			continue
		}
		reactionResponses = append(reactionResponses, &dto.ReactionResponse{
			ID:        reaction.ID,
			UserID:    reaction.UserID,
			UserName:  user.Email,
			Emoji:     reaction.Emoji,
			CreatedAt: reaction.CreatedAt,
		})
	}
	response.Reactions = reactionResponses

	var attachmentResponses []*dto.AttachmentResponse
	for _, attachment := range message.Attachments {
		att := attachment
		attachmentResponses = append(attachmentResponses, s.buildAttachmentResponse(&att))
	}
	response.Attachments = attachmentResponses

	var readBy []string
	for _, receipt := range message.ReadReceipts {
		readBy = append(readBy, receipt.UserID)
	}
	response.ReadBy = readBy

	return response, nil
}

// (buildAttachmentResponse - чистая функция, без изменений)
func (s *chatService) buildAttachmentResponse(attachment *chat.MessageAttachment) *dto.AttachmentResponse {
	return &dto.AttachmentResponse{
		ID:        attachment.ID,
		MessageID: attachment.MessageID,
		FileType:  attachment.FileType,
		MimeType:  attachment.MimeType,
		FileName:  attachment.FileName,
		URL:       attachment.URL,
		Size:      attachment.Size,
		CreatedAt: attachment.CreatedAt,
	}
}

// (processAttachment - чистая функция, без изменений)
func (s *chatService) processAttachment(userID string, file *multipart.FileHeader) (*chat.MessageAttachment, error) {
	if file.Size > FileConfig.MaxSize {
		return nil, apperrors.ErrFileTooLarge
	}
	if !isValidFileType(file.Header.Get("Content-Type")) {
		return nil, apperrors.ErrInvalidFileType
	}

	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), generateRandomString(8), fileExt)
	filePath := filepath.Join(FileConfig.StoragePath, fileName)

	// TODO: Логика сохранения файла

	return &chat.MessageAttachment{
		UploaderID: userID,
		FileType:   getFileTypeFromMIME(file.Header.Get("Content-Type")),
		MimeType:   file.Header.Get("Content-Type"),
		FileName:   file.Filename,
		URL:        filePath,
		Size:       file.Size,
	}, nil
}

// notifyNewMessage - 'db' добавлен
func (s *chatService) notifyNewMessage(db *gorm.DB, dialogID, senderID, messageID string) {
	// ✅ Используем 'db' из параметра
	participants, err := s.chatRepo.FindParticipantsByDialog(db, dialogID)
	if err != nil {
		log.Printf("notifyNewMessage: failed to find participants: %v", err)
		return
	}

	for _, participant := range participants {
		if participant.UserID != senderID && !participant.IsMuted {
			// ✅ Используем 'db' из параметра
			sender, err := s.userRepo.FindByID(db, senderID)
			if err != nil {
				log.Printf("notifyNewMessage: failed to find sender: %v", err)
				continue
			}

			// ✅ Используем 'db' из параметра
			s.notificationRepo.CreateNewMessageNotification(
				db,
				participant.UserID,
				sender.Email,
				dialogID,
			)
		}
	}
}

// (isValidMessageType - чистая функция, без изменений)
func isValidMessageType(messageType string) bool {
	validTypes := map[string]bool{
		"text": true, "image": true, "video": true, "file": true, "system": true, "forward": true,
	}
	return validTypes[messageType]
}

// (isValidFileType - чистая функция, без изменений)
func isValidFileType(mimeType string) bool {
	for _, allowedType := range FileConfig.AllowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

// (getFileTypeFromMIME - чистая функция, без изменений)
func getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	} else {
		return "file"
	}
}

// (removeDuplicates - чистая функция, без изменений)
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range slice {
		if _, value := keys[item]; !value {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// (generateRandomString - чистая функция, без изменений)
func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}

// (handleChatError - хелпер, без изменений)
func handleChatError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrDialogNotFound) ||
		errors.Is(err, repositories.ErrMessageNotFound) ||
		errors.Is(err, repositories.ErrParticipantNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

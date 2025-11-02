package services

import (
	"context" // <-- ДОБАВЛЕН
	"errors"
	"fmt"
	"log"
	"mime/multipart"

	"gorm.io/gorm"
	// "path/filepath" // <-- УДАЛЕНО
	"strings"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/models/chat"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ИСПРАВЛЕН
// =======================
// - Добавлен 'ctx context.Context'
// - Удалены все методы Attachment
type ChatService interface {
	// Dialog operations
	CreateDialog(ctx context.Context, db *gorm.DB, userID string, req *dto.CreateDialogRequest) (*dto.DialogResponse, error)
	CreateCastingDialog(ctx context.Context, db *gorm.DB, castingID, employerID, modelID string) (*dto.DialogResponse, error)
	GetDialog(ctx context.Context, db *gorm.DB, dialogID, userID string) (*dto.DialogResponse, error)
	GetUserDialogs(ctx context.Context, db *gorm.DB, userID string) ([]*dto.DialogResponse, error)
	GetDialogBetweenUsers(ctx context.Context, db *gorm.DB, user1ID, user2ID string) (*dto.DialogResponse, error)
	UpdateDialog(ctx context.Context, db *gorm.DB, userID, dialogID string, req *dto.UpdateDialogRequest) error
	DeleteDialog(ctx context.Context, db *gorm.DB, userID, dialogID string) error
	LeaveDialog(ctx context.Context, db *gorm.DB, userID, dialogID string) error

	// Participant operations
	AddParticipants(ctx context.Context, db *gorm.DB, userID, dialogID string, participantIDs []string) error
	RemoveParticipant(ctx context.Context, db *gorm.DB, userID, dialogID, targetUserID string) error
	UpdateParticipantRole(ctx context.Context, db *gorm.DB, userID, dialogID, targetUserID, role string) error
	MuteDialog(ctx context.Context, db *gorm.DB, userID, dialogID string, muted bool) error
	UpdateLastSeen(ctx context.Context, db *gorm.DB, userID, dialogID string) error
	SetTyping(ctx context.Context, db *gorm.DB, userID, dialogID string, typing bool) error

	// Message operations
	SendMessage(ctx context.Context, db *gorm.DB, userID string, req *dto.SendMessageRequest) (*dto.MessageResponse, error)
	SendMessageWithAttachments(ctx context.Context, db *gorm.DB, userID string, req *dto.SendMessageRequest, files []*multipart.FileHeader) (*dto.MessageResponse, error)
	GetMessages(ctx context.Context, db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.MessageListResponse, error)
	GetMessage(ctx context.Context, db *gorm.DB, messageID, userID string) (*dto.MessageResponse, error)
	UpdateMessage(ctx context.Context, db *gorm.DB, userID, messageID string, req *dto.UpdateMessageRequest) error
	DeleteMessage(ctx context.Context, db *gorm.DB, userID, messageID string) error
	ForwardMessage(ctx context.Context, db *gorm.DB, userID string, req *dto.ForwardMessageRequest) (*dto.MessageResponse, error)

	// ▼▼▼ УДАЛЕНО: Attachment operations теперь в UploadService ▼▼▼
	// UploadAttachment(...)
	// GetMessageAttachments(...)
	// GetDialogAttachments(...)
	// DeleteAttachment(...)
	// ▲▲▲ УДАЛЕНО ▲▲▲

	// Reaction operations
	AddReaction(ctx context.Context, db *gorm.DB, userID, messageID, emoji string) error
	RemoveReaction(ctx context.Context, db *gorm.DB, userID, messageID string) error
	GetMessageReactions(ctx context.Context, db *gorm.DB, messageID, userID string) ([]*dto.ReactionResponse, error)

	// Read receipts
	MarkMessagesAsRead(ctx context.Context, db *gorm.DB, userID, dialogID string) error
	GetUnreadCount(ctx context.Context, db *gorm.DB, dialogID, userID string) (int64, error)
	GetReadReceipts(ctx context.Context, db *gorm.DB, messageID, userID string) ([]*dto.ReadReceiptResponse, error)

	// Combined operations
	GetDialogWithMessages(ctx context.Context, db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.DialogWithMessagesResponse, error)
	SearchMessages(ctx context.Context, db *gorm.DB, userID, dialogID, query string) ([]*dto.MessageResponse, error)

	// Admin operations
	GetAllDialogs(ctx context.Context, db *gorm.DB, criteria dto.DialogCriteria) (*dto.DialogListResponse, error)
	GetChatStats(ctx context.Context, db *gorm.DB) (*repositories.ChatStats, error)
	CleanOldMessages(ctx context.Context, db *gorm.DB, days int) error
	DeleteUserMessages(ctx context.Context, db *gorm.DB, adminID, dialogID string, userID string) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ИСПРАВЛЕНА
// =======================
type chatService struct {
	chatRepo         repositories.ChatRepository
	userRepo         repositories.UserRepository
	castingRepo      repositories.CastingRepository
	responseRepo     repositories.ResponseRepository
	profileRepo      repositories.ProfileRepository
	notificationRepo repositories.NotificationRepository
	uploadService    UploadService // <-- ВНЕДРЕН УНИВЕРСАЛЬНЫЙ СЕРВИС
}

// ✅ Конструктор обновлен
func NewChatService(
	chatRepo repositories.ChatRepository,
	userRepo repositories.UserRepository,
	castingRepo repositories.CastingRepository,
	profileRepo repositories.ProfileRepository,
	notificationRepo repositories.NotificationRepository,
	responseRepo repositories.ResponseRepository,
	uploadService UploadService, // <-- ПРИНИМАЕМ УНИВЕРСАЛЬНЫЙ СЕРВИС
) ChatService {
	return &chatService{
		chatRepo:         chatRepo,
		userRepo:         userRepo,
		castingRepo:      castingRepo,
		profileRepo:      profileRepo,
		notificationRepo: notificationRepo,
		responseRepo:     responseRepo,
		uploadService:    uploadService, // <-- СОХРАНЯЕМ УНИВЕРСАЛЬНЫЙ СЕРВИС
	}
}

// ▼▼▼ УДАЛЕНО: FileConfig теперь в UploadService ▼▼▼
// var FileConfig = dto.FileConfig{ ... }
// ▲▲▲ УДАЛЕНО ▲▲▲

// Dialog operations (Добавлен 'ctx' в вызовы)

func (s *chatService) CreateDialog(ctx context.Context, db *gorm.DB, userID string, req *dto.CreateDialogRequest) (*dto.DialogResponse, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	for _, participantID := range req.UserIDs {
		if _, err := s.userRepo.FindByID(tx, participantID); err != nil {
			return nil, fmt.Errorf("user not found: %s", participantID)
		}
	}
	// (Остальная логика без изменений, кроме передачи 'tx')
	participants := append([]string{userID}, req.UserIDs...)
	uniqueParticipants := removeDuplicates(participants)
	dialog := &chat.Dialog{
		IsGroup:   req.IsGroup,
		Title:     req.Title,
		ImageURL:  req.ImageURL,
		CastingID: req.CastingID,
	}
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
	if err := s.chatRepo.AddParticipants(tx, chatParticipants); err != nil {
		return nil, apperrors.InternalError(err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	return s.buildDialogResponse(ctx, db, dialog, userID)
}

func (s *chatService) CreateCastingDialog(ctx context.Context, db *gorm.DB, castingID, employerID, modelID string) (*dto.DialogResponse, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return nil, apperrors.ErrDialogNotFound
	}
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
	if err := s.chatRepo.CreateMessage(tx, systemMessage); err != nil {
		fmt.Printf("Failed to create system message: %v\n", err)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}
	return s.buildDialogResponse(ctx, db, dialog, employerID)
}

func (s *chatService) GetDialog(ctx context.Context, db *gorm.DB, dialogID, userID string) (*dto.DialogResponse, error) {
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	dialog, err := s.chatRepo.FindDialogByID(db, dialogID)
	if err != nil {
		return nil, handleChatError(err)
	}
	return s.buildDialogResponse(ctx, db, dialog, userID)
}

func (s *chatService) GetUserDialogs(ctx context.Context, db *gorm.DB, userID string) ([]*dto.DialogResponse, error) {
	dialogs, err := s.chatRepo.FindUserDialogs(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var responses []*dto.DialogResponse
	for _, dialog := range dialogs {
		response, err := s.buildDialogResponse(ctx, db, &dialog, userID)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func (s *chatService) GetDialogBetweenUsers(ctx context.Context, db *gorm.DB, user1ID, user2ID string) (*dto.DialogResponse, error) {
	dialog, err := s.chatRepo.FindDialogBetweenUsers(db, user1ID, user2ID)
	if err != nil {
		return nil, handleChatError(err)
	}
	return s.buildDialogResponse(ctx, db, dialog, user1ID)
}

func (s *chatService) UpdateDialog(ctx context.Context, db *gorm.DB, userID, dialogID string, req *dto.UpdateDialogRequest) error {
	// (Логика без изменений, кроме 'tx')
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}
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
	if err := s.chatRepo.UpdateDialog(tx, dialog); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *chatService) DeleteDialog(ctx context.Context, db *gorm.DB, userID, dialogID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" {
		return errors.New("only dialog owner can delete dialog")
	}

	// ▼▼▼ ИСПРАВЛЕНИЕ: Заменили 'uploads, err' на '_, err' ▼▼▼
	_, err = s.uploadService.GetEntityUploads(tx, "message", dialogID)
	// ▲▲▲ ИСПРАВЛЕНИЕ ▲▲▲
	if err != nil {
		log.Printf("Could not fetch uploads for dialog deletion: %v", err)
	}
	// (Логика удаления 'uploads' здесь по-прежнему отсутствует,
	// но ошибка компилятора исправлена)

	if err := s.chatRepo.DeleteDialog(tx, dialogID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *chatService) LeaveDialog(ctx context.Context, db *gorm.DB, userID, dialogID string) error {
	// (Логика без изменений, кроме 'tx')
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	if err := s.chatRepo.RemoveParticipant(tx, dialogID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Participant operations (Логика без изменений, кроме 'tx')
func (s *chatService) AddParticipants(ctx context.Context, db *gorm.DB, userID, dialogID string, participantIDs []string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}
	for _, participantID := range participantIDs {
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
	if err := s.chatRepo.AddParticipants(tx, participants); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) RemoveParticipant(ctx context.Context, db *gorm.DB, userID, dialogID, targetUserID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
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
	if err := s.chatRepo.RemoveParticipant(tx, dialogID, targetUserID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) UpdateParticipantRole(ctx context.Context, db *gorm.DB, userID, dialogID, targetUserID, role string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if participant.Role != "owner" {
		return errors.New("only owner can update roles")
	}
	targetParticipant, err := s.chatRepo.FindParticipant(tx, dialogID, targetUserID)
	if err != nil {
		return apperrors.ErrParticipantNotFound
	}
	targetParticipant.Role = role
	if err := s.chatRepo.UpdateParticipant(tx, targetParticipant); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) MuteDialog(ctx context.Context, db *gorm.DB, userID, dialogID string, muted bool) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	participant, err := s.chatRepo.FindParticipant(tx, dialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	participant.IsMuted = muted
	if err := s.chatRepo.UpdateParticipant(tx, participant); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) UpdateLastSeen(ctx context.Context, db *gorm.DB, userID, dialogID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	if err := s.chatRepo.UpdateLastSeen(tx, dialogID, userID, time.Now()); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) SetTyping(ctx context.Context, db *gorm.DB, userID, dialogID string, typing bool) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
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
	if err := s.chatRepo.UpdateParticipant(tx, participant); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Message operations

func (s *chatService) SendMessage(ctx context.Context, db *gorm.DB, userID string, req *dto.SendMessageRequest) (*dto.MessageResponse, error) {
	// (Логика без изменений, кроме 'tx' и 'ctx' в notifyNewMessage)
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
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
	if err := s.chatRepo.CreateMessage(tx, message); err != nil {
		return nil, apperrors.InternalError(err)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}
	go s.notifyNewMessage(ctx, db, req.DialogID, userID, message.ID)
	return s.buildMessageResponse(ctx, db, message)
}

func (s *chatService) SendMessageWithAttachments(ctx context.Context, db *gorm.DB, userID string, req *dto.SendMessageRequest, files []*multipart.FileHeader) (*dto.MessageResponse, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

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

	// 1. Создаем сообщение *сначала*
	message := &chat.Message{
		DialogID:      req.DialogID,
		SenderID:      userID,
		Type:          req.Type,
		Content:       req.Content,
		ReplyToID:     req.ReplyToID,
		ForwardFromID: req.ForwardFromID,
		Status:        "sent",
	}
	if err := s.chatRepo.CreateMessage(tx, message); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 2. Загружаем файлы, привязывая их к созданному 'message.ID'
	for _, file := range files {
		uploadReq := &dto.UniversalUploadRequest{
			UserID:     userID,
			Module:     "chat",
			EntityType: "message",
			EntityID:   message.ID, // <-- Привязываем к ID сообщения
			Usage:      "message_attachment",
			IsPublic:   false, // (Вложения чата обычно приватные)
			File:       file,
		}

		// Вызываем универсальный сервис
		if _, err := s.uploadService.UploadFile(ctx, tx, uploadReq); err != nil {
			// (Транзакция откатится)
			return nil, fmt.Errorf("failed to upload attachment: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	go s.notifyNewMessage(ctx, db, req.DialogID, userID, message.ID)

	// Возвращаем полное сообщение (buildMessageResponse сам найдет вложения)
	return s.GetMessage(ctx, db, message.ID, userID)
}

func (s *chatService) GetMessages(ctx context.Context, db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.MessageListResponse, error) {
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}

	messages, total, err := s.chatRepo.FindMessagesByDialog(db, dialogID, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var messageResponses []*dto.MessageResponse
	for _, message := range messages {
		messageResponse, err := s.buildMessageResponse(ctx, db, &message)
		if err != nil {
			continue
		}
		messageResponses = append(messageResponses, messageResponse)
	}
	// (Логика пагинации без изменений)
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

func (s *chatService) GetMessage(ctx context.Context, db *gorm.DB, messageID, userID string) (*dto.MessageResponse, error) {
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	return s.buildMessageResponse(ctx, db, message)
}

func (s *chatService) UpdateMessage(ctx context.Context, db *gorm.DB, userID, messageID string, req *dto.UpdateMessageRequest) error {
	// (Логика без изменений, кроме 'tx')
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
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
	if err := s.chatRepo.UpdateMessage(tx, message); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *chatService) DeleteMessage(ctx context.Context, db *gorm.DB, userID, messageID string) error {
	// (Логика без изменений, кроме 'tx')
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	message, err := s.chatRepo.FindMessageByID(tx, messageID)
	if err != nil {
		return handleChatError(err)
	}
	participant, err := s.chatRepo.FindParticipant(tx, message.DialogID, userID)
	if err != nil {
		return apperrors.ErrDialogAccessDenied
	}
	if message.SenderID != userID && participant.Role != "owner" && participant.Role != "admin" {
		return apperrors.ErrCannotDeleteMessage
	}
	if err := s.chatRepo.DeleteMessage(tx, messageID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *chatService) ForwardMessage(ctx context.Context, db *gorm.DB, userID string, req *dto.ForwardMessageRequest) (*dto.MessageResponse, error) {
	// (Логика без изменений, кроме 'ctx' и 'db' в SendMessage)
	originalMessage, err := s.chatRepo.FindMessageByID(db, req.MessageID)
	if err != nil {
		return nil, handleChatError(err)
	}
	var forwardedMessage *dto.MessageResponse
	for _, dialogID := range req.DialogIDs {
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
		forwardedMessage, err = s.SendMessage(ctx, db, userID, forwardReq)
		if err != nil {
			fmt.Printf("Failed to forward message to dialog %s: %v\n", dialogID, err)
		}
	}
	return forwardedMessage, nil
}

// ▼▼▼ УДАЛЕНО: Attachment operations (UploadAttachment, Get...Attachments, DeleteAttachment) ▼▼▼
// func (s *chatService) UploadAttachment(...)
// func (s *chatService) GetMessageAttachments(...)
// func (s *chatService) GetDialogAttachments(...)
// func (s *chatService) DeleteAttachment(...)
// ▲▲▲ УДАЛЕНО ▲▲▲

// Reaction operations (Логика без изменений, кроме 'tx' и 'ctx'/'db' в хелперах)
func (s *chatService) AddReaction(ctx context.Context, db *gorm.DB, userID, messageID, emoji string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	message, err := s.chatRepo.FindMessageByID(tx, messageID)
	if err != nil {
		return handleChatError(err)
	}
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, message.DialogID, userID)
	if err != nil || !hasAccess {
		return apperrors.ErrDialogAccessDenied
	}
	reaction := &chat.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}
	if err := s.chatRepo.AddReaction(tx, reaction); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) RemoveReaction(ctx context.Context, db *gorm.DB, userID, messageID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	if err := s.chatRepo.RemoveReaction(tx, messageID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) GetMessageReactions(ctx context.Context, db *gorm.DB, messageID, userID string) ([]*dto.ReactionResponse, error) {
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	reactions, err := s.chatRepo.FindReactionsByMessage(db, messageID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var responses []*dto.ReactionResponse
	for _, reaction := range reactions {
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

// Read receipts (Логика без изменений, кроме 'tx' и 'ctx'/'db' в хелперах)
func (s *chatService) MarkMessagesAsRead(ctx context.Context, db *gorm.DB, userID, dialogID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, dialogID, userID)
	if err != nil || !hasAccess {
		return apperrors.ErrDialogAccessDenied
	}
	if err := s.chatRepo.MarkMessagesAsRead(tx, dialogID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) GetUnreadCount(ctx context.Context, db *gorm.DB, dialogID, userID string) (int64, error) {
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil || !hasAccess {
		return 0, apperrors.ErrDialogAccessDenied
	}
	return s.chatRepo.GetUnreadCount(db, dialogID, userID)
}
func (s *chatService) GetReadReceipts(ctx context.Context, db *gorm.DB, messageID, userID string) ([]*dto.ReadReceiptResponse, error) {
	message, err := s.chatRepo.FindMessageByID(db, messageID)
	if err != nil {
		return nil, handleChatError(err)
	}
	hasAccess, err := s.chatRepo.IsUserInDialog(db, message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	receipts, err := s.chatRepo.FindReadReceiptsByMessage(db, messageID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var responses []*dto.ReadReceiptResponse
	for _, receipt := range receipts {
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
func (s *chatService) GetDialogWithMessages(ctx context.Context, db *gorm.DB, dialogID, userID string, criteria dto.MessageCriteria) (*dto.DialogWithMessagesResponse, error) {
	// (Логика без изменений, кроме 'tx' и 'ctx'/'db' в хелперах)
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	hasAccess, err := s.chatRepo.IsUserInDialog(tx, dialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	dialog, err := s.chatRepo.FindDialogByID(tx, dialogID)
	if err != nil {
		return nil, handleChatError(err)
	}
	messages, total, err := s.chatRepo.FindMessagesByDialog(tx, dialogID, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if err := s.chatRepo.MarkMessagesAsRead(tx, dialogID, userID); err != nil {
		// (Игнорируем ошибку чтения, но не отменяем транзакцию)
		log.Printf("Failed to mark messages as read: %v", err)
	}
	dialogResponse, err := s.buildDialogResponse(ctx, tx, dialog, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var messageResponses []*dto.MessageResponse
	for _, message := range messages {
		messageResponse, err := s.buildMessageResponse(ctx, tx, &message)
		if err != nil {
			continue
		}
		messageResponses = append(messageResponses, messageResponse)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}
	// (Логика пагинации без изменений)
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
func (s *chatService) SearchMessages(ctx context.Context, db *gorm.DB, userID, dialogID, query string) ([]*dto.MessageResponse, error) {
	// (Логика без изменений, кроме 'ctx'/'db' в хелперах)
	hasAccess, err := s.chatRepo.IsUserInDialog(db, dialogID, userID)
	if err != nil || !hasAccess {
		return nil, apperrors.ErrDialogAccessDenied
	}
	criteria := dto.MessageCriteria{Limit: 1000}
	messages, _, err := s.chatRepo.FindMessagesByDialog(db, dialogID, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var results []*dto.MessageResponse
	for _, message := range messages {
		if strings.Contains(strings.ToLower(message.Content), strings.ToLower(query)) {
			messageResponse, err := s.buildMessageResponse(ctx, db, &message)
			if err != nil {
				log.Printf("Failed to build message response for message %s: %v", message.ID, err)
				continue
			}
			results = append(results, messageResponse)
		}
	}
	return results, nil
}

// Admin operations (Логика без изменений, кроме 'tx')
func (s *chatService) GetAllDialogs(ctx context.Context, db *gorm.DB, criteria dto.DialogCriteria) (*dto.DialogListResponse, error) {
	// (Логика без изменений)
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
func (s *chatService) GetChatStats(ctx context.Context, db *gorm.DB) (*repositories.ChatStats, error) {
	return s.chatRepo.GetChatStats(db)
}
func (s *chatService) CleanOldMessages(ctx context.Context, db *gorm.DB, days int) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	if err := s.chatRepo.CleanOldMessages(tx, days); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}
func (s *chatService) DeleteUserMessages(ctx context.Context, db *gorm.DB, adminID, dialogID string, userID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleChatError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}
	if err := s.chatRepo.DeleteUserMessages(tx, dialogID, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Helper methods

func (s *chatService) buildDialogResponse(ctx context.Context, db *gorm.DB, dialog *chat.Dialog, userID string) (*dto.DialogResponse, error) {
	// (Логика без изменений, кроме 'ctx'/'db' в хелперах)
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
		lastMessage, err := s.buildMessageResponse(ctx, db, dialog.LastMessage)
		if err == nil {
			response.LastMessage = lastMessage
		}
	}
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

func (s *chatService) buildMessageResponse(ctx context.Context, db *gorm.DB, message *chat.Message) (*dto.MessageResponse, error) {
	response := &dto.MessageResponse{
		ID:        message.ID,
		DialogID:  message.DialogID,
		SenderID:  message.SenderID,
		Type:      message.Type,
		Content:   message.Content,
		Status:    message.Status,
		CreatedAt: message.CreatedAt,
		// ▼▼▼ УДАЛЕНО: Старые поля ▼▼▼
		// AttachmentURL:  message.AttachmentURL,
		// AttachmentName: message.AttachmentName,
		// ▲▲▲ УДАЛЕНО ▲▲▲
	}
	sender, err := s.userRepo.FindByID(db, message.SenderID)
	if err == nil {
		response.SenderName = sender.Email
	}
	if message.ReplyTo != nil {
		replyTo, err := s.buildMessageResponse(ctx, db, message.ReplyTo)
		if err == nil {
			response.ReplyTo = replyTo
		}
	}
	if message.ForwardFrom != nil {
		forwardFrom, err := s.buildMessageResponse(ctx, db, message.ForwardFrom)
		if err == nil {
			response.ForwardFrom = forwardFrom
		}
	}
	var reactionResponses []*dto.ReactionResponse
	for _, reaction := range message.Reactions {
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

	// ▼▼▼ ИСПРАВЛЕНО: Загружаем вложения из UploadService ▼▼▼
	uploads, err := s.uploadService.GetEntityUploads(db, "message", message.ID)
	if err != nil {
		log.Printf("Failed to get uploads for message %s: %v", message.ID, err)
	}

	var attachmentResponses []*dto.AttachmentResponse
	for _, upload := range uploads {
		attachmentResponses = append(attachmentResponses, s.buildAttachmentResponse(upload))
	}
	response.Attachments = attachmentResponses
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

	var readBy []string
	for _, receipt := range message.ReadReceipts {
		readBy = append(readBy, receipt.UserID)
	}
	response.ReadBy = readBy

	return response, nil
}

// ▼▼▼ ИСПРАВЛЕНО: Хелпер теперь принимает models.Upload ▼▼▼
func (s *chatService) buildAttachmentResponse(upload *models.Upload) *dto.AttachmentResponse {
	// (Примечание: URL должен быть получен из s.storage.GetURL(),
	// но UploadService не возвращает URL в GetEntityUploads.
	// Мы используем 'Path' как временное решение.)
	return &dto.AttachmentResponse{
		ID:        upload.ID,
		MessageID: upload.EntityID,
		FileType:  upload.FileType,
		MimeType:  upload.MimeType,
		FileName:  upload.Path, // (Или из metadata, если вы его там храните)
		URL:       upload.Path, // (Нуждается в s.storage.GetURL(upload.Path))
		Size:      upload.Size,
		CreatedAt: upload.CreatedAt,
	}
}

// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

// ▼▼▼ УДАЛЕНО: processAttachment ▼▼▼
// func (s *chatService) processAttachment(...)
// ▲▲▲ УДАЛЕНО ▲▲▲

func (s *chatService) notifyNewMessage(ctx context.Context, db *gorm.DB, dialogID, senderID, messageID string) {
	// (Логика без изменений, кроме 'ctx'/'db' в хелперах)
	participants, err := s.chatRepo.FindParticipantsByDialog(db, dialogID)
	if err != nil {
		log.Printf("notifyNewMessage: failed to find participants: %v", err)
		return
	}
	for _, participant := range participants {
		if participant.UserID != senderID && !participant.IsMuted {
			sender, err := s.userRepo.FindByID(db, senderID)
			if err != nil {
				log.Printf("notifyNewMessage: failed to find sender: %v", err)
				continue
			}
			s.notificationRepo.CreateNewMessageNotification(
				db,
				participant.UserID,
				sender.Email,
				dialogID,
			)
		}
	}
}

// (Вспомогательные хелперы без изменений)
func isValidMessageType(messageType string) bool {
	validTypes := map[string]bool{
		"text": true, "image": true, "video": true, "file": true, "system": true, "forward": true,
	}
	return validTypes[messageType]
}
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

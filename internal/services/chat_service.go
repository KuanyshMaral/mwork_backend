package services

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/models"
	"mwork_backend/internal/models/chat"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

type ChatService interface {
	// Dialog operations
	CreateDialog(userID string, req *dto.CreateDialogRequest) (*dto.DialogResponse, error)
	CreateCastingDialog(castingID, employerID, modelID string) (*dto.DialogResponse, error)
	GetDialog(dialogID, userID string) (*dto.DialogResponse, error)
	GetUserDialogs(userID string) ([]*dto.DialogResponse, error)
	GetDialogBetweenUsers(user1ID, user2ID string) (*dto.DialogResponse, error)
	UpdateDialog(userID, dialogID string, req *dto.UpdateDialogRequest) error
	DeleteDialog(userID, dialogID string) error
	LeaveDialog(userID, dialogID string) error

	// Participant operations
	AddParticipants(userID, dialogID string, participantIDs []string) error
	RemoveParticipant(userID, dialogID, targetUserID string) error
	UpdateParticipantRole(userID, dialogID, targetUserID, role string) error
	MuteDialog(userID, dialogID string, muted bool) error
	UpdateLastSeen(userID, dialogID string) error
	SetTyping(userID, dialogID string, typing bool) error

	// Message operations
	SendMessage(userID string, req *dto.SendMessageRequest) (*dto.MessageResponse, error)
	SendMessageWithAttachments(userID string, req *dto.SendMessageRequest, files []*multipart.FileHeader) (*dto.MessageResponse, error)
	GetMessages(dialogID, userID string, criteria dto.MessageCriteria) (*dto.MessageListResponse, error)
	GetMessage(messageID, userID string) (*dto.MessageResponse, error)
	UpdateMessage(userID, messageID string, req *dto.UpdateMessageRequest) error
	DeleteMessage(userID, messageID string) error
	ForwardMessage(userID string, req *dto.ForwardMessageRequest) (*dto.MessageResponse, error)

	// Attachment operations
	UploadAttachment(userID string, file *multipart.FileHeader) (*dto.AttachmentResponse, error)
	GetMessageAttachments(messageID, userID string) ([]*dto.AttachmentResponse, error)
	GetDialogAttachments(dialogID, userID string) ([]*dto.AttachmentResponse, error)
	DeleteAttachment(userID, attachmentID string) error

	// Reaction operations
	AddReaction(userID, messageID, emoji string) error
	RemoveReaction(userID, messageID string) error
	GetMessageReactions(messageID, userID string) ([]*dto.ReactionResponse, error)

	// Read receipts
	MarkMessagesAsRead(userID, dialogID string) error
	GetUnreadCount(dialogID, userID string) (int64, error)
	GetReadReceipts(messageID, userID string) ([]*dto.ReadReceiptResponse, error)

	// Combined operations
	GetDialogWithMessages(dialogID, userID string, criteria dto.MessageCriteria) (*dto.DialogWithMessagesResponse, error)
	SearchMessages(userID, dialogID, query string) ([]*dto.MessageResponse, error)

	// Admin operations
	GetAllDialogs(criteria dto.DialogCriteria) (*dto.DialogListResponse, error)
	GetChatStats() (*repositories.ChatStats, error)
	CleanOldMessages(days int) error
	DeleteUserMessages(adminID, userID string) error
}

type chatService struct {
	chatRepo         repositories.ChatRepository
	userRepo         repositories.UserRepository
	castingRepo      repositories.CastingRepository
	responseRepo     repositories.ResponseRepository
	profileRepo      repositories.ProfileRepository
	notificationRepo repositories.NotificationRepository
}

func NewChatService(
	chatRepo repositories.ChatRepository,
	userRepo repositories.UserRepository,
	castingRepo repositories.CastingRepository,
	profileRepo repositories.ProfileRepository,
	notificationRepo repositories.NotificationRepository,
) ChatService {
	return &chatService{
		chatRepo:         chatRepo,
		userRepo:         userRepo,
		castingRepo:      castingRepo,
		profileRepo:      profileRepo,
		notificationRepo: notificationRepo,
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

func (s *chatService) CreateDialog(userID string, req *dto.CreateDialogRequest) (*dto.DialogResponse, error) {
	// Validate users exist
	for _, participantID := range req.UserIDs {
		if _, err := s.userRepo.FindByID(participantID); err != nil {
			return nil, fmt.Errorf("user not found: %s", participantID)
		}
	}

	// Ensure creator is in participants
	participants := append([]string{userID}, req.UserIDs...)
	uniqueParticipants := removeDuplicates(participants)

	// Create dialog
	dialog := &chat.Dialog{
		IsGroup:   req.IsGroup,
		Title:     req.Title,
		ImageURL:  req.ImageURL,
		CastingID: req.CastingID,
	}

	if err := s.chatRepo.CreateDialog(dialog); err != nil {
		return nil, err
	}

	// Add participants
	var chatParticipants []*chat.DialogParticipant
	for i, participantID := range uniqueParticipants {
		role := "member"
		if i == 0 { // Creator is owner
			role = "owner"
		}

		chatParticipants = append(chatParticipants, &chat.DialogParticipant{
			DialogID: dialog.ID,
			UserID:   participantID,
			Role:     role,
			JoinedAt: time.Now(),
		})
	}

	if err := s.chatRepo.AddParticipants(chatParticipants); err != nil {
		// Clean up dialog if participants fail
		s.chatRepo.DeleteDialog(dialog.ID)
		return nil, err
	}

	return s.buildDialogResponse(dialog, userID)
}

func (s *chatService) CreateCastingDialog(castingID, employerID, modelID string) (*dto.DialogResponse, error) {
	// Get casting details
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, appErrors.ErrDialogNotFound
	}

	// Create dialog through repository
	dialog, err := s.chatRepo.CreateCastingDialog(casting, employerID, modelID)
	if err != nil {
		return nil, err
	}

	// Send system message
	systemMessage := &chat.Message{
		DialogID: dialog.ID,
		SenderID: "system",
		Type:     "system",
		Content:  fmt.Sprintf("Чат создан для кастинга '%s'", casting.Title),
		Status:   "sent",
	}

	if err := s.chatRepo.CreateMessage(systemMessage); err != nil {
		// Log error but don't fail
		fmt.Printf("Failed to create system message: %v\n", err)
	}

	return s.buildDialogResponse(dialog, employerID)
}

func (s *chatService) GetDialog(dialogID, userID string) (*dto.DialogResponse, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	dialog, err := s.chatRepo.FindDialogByID(dialogID)
	if err != nil {
		return nil, err
	}

	return s.buildDialogResponse(dialog, userID)
}

func (s *chatService) GetUserDialogs(userID string) ([]*dto.DialogResponse, error) {
	dialogs, err := s.chatRepo.FindUserDialogs(userID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.DialogResponse
	for _, dialog := range dialogs {
		response, err := s.buildDialogResponse(&dialog, userID)
		if err != nil {
			continue // Skip problematic dialogs
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (s *chatService) GetDialogBetweenUsers(user1ID, user2ID string) (*dto.DialogResponse, error) {
	dialog, err := s.chatRepo.FindDialogBetweenUsers(user1ID, user2ID)
	if err != nil {
		if errors.Is(err, repositories.ErrDialogNotFound) {
			return nil, appErrors.ErrDialogNotFound
		}
		return nil, err
	}

	return s.buildDialogResponse(dialog, user1ID)
}

func (s *chatService) UpdateDialog(userID, dialogID string, req *dto.UpdateDialogRequest) error {
	// Check permissions
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}

	dialog, err := s.chatRepo.FindDialogByID(dialogID)
	if err != nil {
		return err
	}

	// Update fields
	if req.Title != nil {
		dialog.Title = req.Title
	}
	if req.ImageURL != nil {
		dialog.ImageURL = req.ImageURL
	}

	return s.chatRepo.UpdateDialog(dialog)
}

func (s *chatService) DeleteDialog(userID, dialogID string) error {
	// Check permissions
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	if participant.Role != "owner" {
		return errors.New("only dialog owner can delete dialog")
	}

	return s.chatRepo.DeleteDialog(dialogID)
}

func (s *chatService) LeaveDialog(userID, dialogID string) error {
	return s.chatRepo.RemoveParticipant(dialogID, userID)
}

// Participant operations

func (s *chatService) AddParticipants(userID, dialogID string, participantIDs []string) error {
	// Check permissions
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}

	// Validate users
	for _, participantID := range participantIDs {
		if _, err := s.userRepo.FindByID(participantID); err != nil {
			return fmt.Errorf("user not found: %s", participantID)
		}
	}

	// Add participants
	var participants []*chat.DialogParticipant
	for _, participantID := range participantIDs {
		participants = append(participants, &chat.DialogParticipant{
			DialogID: dialogID,
			UserID:   participantID,
			Role:     "member",
			JoinedAt: time.Now(),
		})
	}

	return s.chatRepo.AddParticipants(participants)
}

func (s *chatService) RemoveParticipant(userID, dialogID, targetUserID string) error {
	// Check permissions
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	targetParticipant, err := s.chatRepo.FindParticipant(dialogID, targetUserID)
	if err != nil {
		return appErrors.ErrParticipantNotFound
	}

	// Only owner/admin can remove, and cannot remove owner
	if participant.Role != "owner" && participant.Role != "admin" {
		return errors.New("insufficient permissions")
	}
	if targetParticipant.Role == "owner" {
		return errors.New("cannot remove dialog owner")
	}

	return s.chatRepo.RemoveParticipant(dialogID, targetUserID)
}

func (s *chatService) UpdateParticipantRole(userID, dialogID, targetUserID, role string) error {
	// Check permissions
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	if participant.Role != "owner" {
		return errors.New("only owner can update roles")
	}

	targetParticipant, err := s.chatRepo.FindParticipant(dialogID, targetUserID)
	if err != nil {
		return appErrors.ErrParticipantNotFound
	}

	targetParticipant.Role = role
	return s.chatRepo.UpdateParticipant(targetParticipant)
}

func (s *chatService) MuteDialog(userID, dialogID string, muted bool) error {
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	participant.IsMuted = muted
	return s.chatRepo.UpdateParticipant(participant)
}

func (s *chatService) UpdateLastSeen(userID, dialogID string) error {
	return s.chatRepo.UpdateLastSeen(dialogID, userID, time.Now())
}

func (s *chatService) SetTyping(userID, dialogID string, typing bool) error {
	participant, err := s.chatRepo.FindParticipant(dialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	if typing {
		typingUntil := time.Now().Add(10 * time.Second) // Typing indicator lasts 10 seconds
		participant.TypingUntil = &typingUntil
	} else {
		participant.TypingUntil = nil
	}

	return s.chatRepo.UpdateParticipant(participant)
}

// Message operations

func (s *chatService) SendMessage(userID string, req *dto.SendMessageRequest) (*dto.MessageResponse, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(req.DialogID, userID)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	// Validate message type
	if !isValidMessageType(req.Type) {
		return nil, appErrors.ErrInvalidMessageType
	}

	// Create message
	message := &chat.Message{
		DialogID:      req.DialogID,
		SenderID:      userID,
		Type:          req.Type,
		Content:       req.Content,
		ReplyToID:     req.ReplyToID,
		ForwardFromID: req.ForwardFromID,
		Status:        "sent",
	}

	if err := s.chatRepo.CreateMessage(message); err != nil {
		return nil, err
	}

	// Send notifications to other participants
	go s.notifyNewMessage(req.DialogID, userID, message.ID)

	return s.buildMessageResponse(message)
}

func (s *chatService) SendMessageWithAttachments(userID string, req *dto.SendMessageRequest, files []*multipart.FileHeader) (*dto.MessageResponse, error) {
	// First create the message
	message, err := s.SendMessage(userID, req)
	if err != nil {
		return nil, err
	}

	// Process and attach files
	var attachments []*chat.MessageAttachment
	for _, file := range files {
		attachment, err := s.processAttachment(userID, file)
		if err != nil {
			continue // Skip problematic files
		}
		attachment.MessageID = message.ID
		attachments = append(attachments, attachment)
	}

	// Save attachments
	for _, attachment := range attachments {
		if err := s.chatRepo.CreateAttachment(attachment); err != nil {
			// Log error but don't fail the whole operation
			fmt.Printf("Failed to create attachment: %v\n", err)
		}
	}

	// Reload message with attachments
	return s.GetMessage(message.ID, userID)
}

func (s *chatService) GetMessages(dialogID, userID string, criteria dto.MessageCriteria) (*dto.MessageListResponse, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	messages, total, err := s.chatRepo.FindMessagesByDialog(dialogID, criteria)
	if err != nil {
		return nil, err
	}

	var messageResponses []*dto.MessageResponse
	for _, message := range messages {
		messageResponse, err := s.buildMessageResponse(&message)
		if err != nil {
			// Обработайте ошибку - пропустите сообщение, верните ошибку, или используйте логирование
			continue // или return nil, err
		}
		messageResponses = append(messageResponses, messageResponse)
	}

	return &dto.MessageListResponse{
		Messages:   messageResponses,
		Total:      total,
		Page:       dto.CriteriaPage{}.Page,                                 // 0
		PageSize:   dto.CriteriaPage{}.PageSize,                             // 0
		TotalPages: calculateTotalPages(total, dto.CriteriaPage{}.PageSize), // 0
		HasMore:    int64(criteria.Offset+len(messages)) < total,
	}, nil
}

func (s *chatService) GetMessage(messageID, userID string) (*dto.MessageResponse, error) {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// Check access to dialog
	hasAccess, err := s.chatRepo.IsUserInDialog(message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	return s.buildMessageResponse(message)
}

func (s *chatService) UpdateMessage(userID, messageID string, req *dto.UpdateMessageRequest) error {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return err
	}

	// Check ownership
	if message.SenderID != userID {
		return errors.New("can only edit own messages")
	}

	// Check if message can be edited (within time limit)
	if time.Since(message.CreatedAt) > 15*time.Minute {
		return errors.New("message can only be edited within 15 minutes")
	}

	message.Content = req.Content
	// This would require an UpdateMessage method in repository
	// For now, we'll implement basic update
	return s.chatRepo.UpdateMessageStatus(messageID, "edited")
}

func (s *chatService) DeleteMessage(userID, messageID string) error {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return err
	}

	// Check ownership or admin rights
	participant, err := s.chatRepo.FindParticipant(message.DialogID, userID)
	if err != nil {
		return appErrors.ErrDialogAccessDenied
	}

	if message.SenderID != userID && participant.Role != "owner" && participant.Role != "admin" {
		return appErrors.ErrCannotDeleteMessage
	}

	return s.chatRepo.DeleteMessage(messageID)
}

func (s *chatService) ForwardMessage(userID string, req *dto.ForwardMessageRequest) (*dto.MessageResponse, error) {
	// Get original message
	originalMessage, err := s.chatRepo.FindMessageByID(req.MessageID)
	if err != nil {
		return nil, err
	}

	// Forward to each dialog
	var forwardedMessage *dto.MessageResponse
	for _, dialogID := range req.DialogIDs {
		// Check access to each dialog
		hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
		if err != nil || !hasAccess {
			continue // Skip inaccessible dialogs
		}

		forwardReq := &dto.SendMessageRequest{
			DialogID:      dialogID,
			Type:          "forward",
			Content:       originalMessage.Content,
			ForwardFromID: &originalMessage.ID,
		}

		forwardedMessage, err = s.SendMessage(userID, forwardReq)
		if err != nil {
			// Log but continue with other dialogs
			fmt.Printf("Failed to forward message to dialog %s: %v\n", dialogID, err)
		}
	}

	return forwardedMessage, nil
}

// Attachment operations

func (s *chatService) UploadAttachment(userID string, file *multipart.FileHeader) (*dto.AttachmentResponse, error) {
	attachment, err := s.processAttachment(userID, file)
	if err != nil {
		return nil, err
	}

	return s.buildAttachmentResponse(attachment), nil
}

func (s *chatService) GetMessageAttachments(messageID, userID string) ([]*dto.AttachmentResponse, error) {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	attachments, err := s.chatRepo.FindAttachmentsByMessage(messageID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.AttachmentResponse
	for _, attachment := range attachments {
		responses = append(responses, s.buildAttachmentResponse(&attachment))
	}

	return responses, nil
}

func (s *chatService) GetDialogAttachments(dialogID, userID string) ([]*dto.AttachmentResponse, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	attachments, err := s.chatRepo.FindAttachmentsByDialog(dialogID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.AttachmentResponse
	for _, attachment := range attachments {
		responses = append(responses, s.buildAttachmentResponse(&attachment))
	}

	return responses, nil
}

func (s *chatService) DeleteAttachment(userID, attachmentID string) error {
	// This would require additional checks in a real implementation
	return s.chatRepo.DeleteAttachment(attachmentID)
}

// Reaction operations

func (s *chatService) AddReaction(userID, messageID, emoji string) error {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return err
	}

	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(message.DialogID, userID)
	if err != nil || !hasAccess {
		return appErrors.ErrDialogAccessDenied
	}

	reaction := &chat.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}

	return s.chatRepo.AddReaction(reaction)
}

func (s *chatService) RemoveReaction(userID, messageID string) error {
	return s.chatRepo.RemoveReaction(messageID, userID)
}

func (s *chatService) GetMessageReactions(messageID, userID string) ([]*dto.ReactionResponse, error) {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	reactions, err := s.chatRepo.FindReactionsByMessage(messageID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.ReactionResponse
	for _, reaction := range reactions {
		user, err := s.userRepo.FindByID(reaction.UserID)
		if err != nil {
			continue
		}

		responses = append(responses, &dto.ReactionResponse{
			ID:        reaction.ID,
			UserID:    reaction.UserID,
			UserName:  user.Email, // Or user name if available
			Emoji:     reaction.Emoji,
			CreatedAt: reaction.CreatedAt,
		})
	}

	return responses, nil
}

// Read receipts

func (s *chatService) MarkMessagesAsRead(userID, dialogID string) error {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil || !hasAccess {
		return appErrors.ErrDialogAccessDenied
	}

	return s.chatRepo.MarkMessagesAsRead(dialogID, userID)
}

func (s *chatService) GetUnreadCount(dialogID, userID string) (int64, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil || !hasAccess {
		return 0, appErrors.ErrDialogAccessDenied
	}

	return s.chatRepo.GetUnreadCount(dialogID, userID)
}

func (s *chatService) GetReadReceipts(messageID, userID string) ([]*dto.ReadReceiptResponse, error) {
	message, err := s.chatRepo.FindMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(message.DialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	receipts, err := s.chatRepo.FindReadReceiptsByMessage(messageID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.ReadReceiptResponse
	for _, receipt := range receipts {
		user, err := s.userRepo.FindByID(receipt.UserID)
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

func (s *chatService) GetDialogWithMessages(dialogID, userID string, criteria dto.MessageCriteria) (*dto.DialogWithMessagesResponse, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	dialog, err := s.chatRepo.FindDialogByID(dialogID)
	if err != nil {
		return nil, err
	}

	messages, total, err := s.chatRepo.FindMessagesByDialog(dialogID, criteria)
	if err != nil {
		return nil, err
	}

	// Mark as read
	if err := s.MarkMessagesAsRead(userID, dialogID); err != nil {
		return nil, err
	}

	dialogResponse, err := s.buildDialogResponse(dialog, userID)
	if err != nil {
		return nil, err
	}

	var messageResponses []*dto.MessageResponse
	for _, message := range messages {
		messageResponse, err := s.buildMessageResponse(&message)
		if err != nil {
			// Обработайте ошибку - пропустите сообщение, верните ошибку, или используйте логирование
			continue // или return nil, err
		}
		messageResponses = append(messageResponses, messageResponse)
	}

	return &dto.DialogWithMessagesResponse{
		Dialog:   dialogResponse,
		Messages: messageResponses,
		Total:    total,
		HasMore:  int64(criteria.Offset+len(messages)) < total,
	}, nil
}

func (s *chatService) SearchMessages(userID, dialogID, query string) ([]*dto.MessageResponse, error) {
	// Check access
	hasAccess, err := s.chatRepo.IsUserInDialog(dialogID, userID)
	if err != nil || !hasAccess {
		return nil, appErrors.ErrDialogAccessDenied
	}

	// This would require a search method in repository
	// For now, implement basic search by loading all messages and filtering
	criteria := dto.MessageCriteria{
		Limit: 1000, // Large limit for search
	}

	messages, _, err := s.chatRepo.FindMessagesByDialog(dialogID, criteria)
	if err != nil {
		return nil, err
	}

	var results []*dto.MessageResponse
	for _, message := range messages {
		if strings.Contains(strings.ToLower(message.Content), strings.ToLower(query)) {
			messageResponse, err := s.buildMessageResponse(&message)
			if err != nil {
				// Пропускаем сообщения с ошибками преобразования
				log.Printf("Failed to build message response for message %s: %v", message.ID, err)
				continue
			}
			results = append(results, messageResponse)
		}
	}

	return results, nil
}

// Admin operations

func (s *chatService) GetAllDialogs(criteria dto.DialogCriteria) (*dto.DialogListResponse, error) {
	dialogs, total, err := s.chatRepo.FindAllDialogs(criteria)
	if err != nil {
		return nil, err
	}

	var responses []*dto.DialogResponse
	for _, dialog := range dialogs {
		// For admin, we don't need to build full dialog with user-specific data
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

	return &dto.DialogListResponse{
		Dialogs:    responses,
		Total:      total,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		TotalPages: calculateTotalPages(total, criteria.PageSize),
	}, nil
}

func (s *chatService) GetChatStats() (*repositories.ChatStats, error) {
	return s.chatRepo.GetChatStats()
}

func (s *chatService) CleanOldMessages(days int) error {
	return s.chatRepo.CleanOldMessages(days)
}

func (s *chatService) DeleteUserMessages(adminID, userID string) error {
	// Verify admin permissions
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}

	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}

	// This would require additional implementation
	// For now, return not implemented
	return errors.New("not implemented")
}

// Helper methods

func (s *chatService) buildDialogResponse(dialog *chat.Dialog, userID string) (*dto.DialogResponse, error) {
	response := &dto.DialogResponse{
		ID:        dialog.ID,
		IsGroup:   dialog.IsGroup,
		Title:     dialog.Title,
		ImageURL:  dialog.ImageURL,
		CastingID: dialog.CastingID,
		CreatedAt: dialog.CreatedAt,
		UpdatedAt: dialog.UpdatedAt,
	}

	// Build participants
	var participantResponses []*dto.ParticipantResponse
	for _, participant := range dialog.Participants {
		user, err := s.userRepo.FindByID(participant.UserID)
		if err != nil {
			continue
		}

		participantResponses = append(participantResponses, &dto.ParticipantResponse{
			UserID:     participant.UserID,
			UserName:   user.Email, // Or user name if available
			Role:       participant.Role,
			LastSeenAt: participant.LastSeenAt,
			IsMuted:    participant.IsMuted,
			IsOnline:   false, // Would require online status tracking
		})
	}
	response.Participants = participantResponses

	// Build last message
	if dialog.LastMessage != nil {
		lastMessage, err := s.buildMessageResponse(dialog.LastMessage)
		if err == nil {
			response.LastMessage = lastMessage
		}
	}

	// Get unread count
	unreadCount, err := s.GetUnreadCount(dialog.ID, userID)
	if err == nil {
		response.UnreadCount = unreadCount
	}

	// Get muted status for current user
	for _, participant := range dialog.Participants {
		if participant.UserID == userID {
			response.IsMuted = participant.IsMuted
			break
		}
	}

	return response, nil
}

func (s *chatService) buildMessageResponse(message *chat.Message) (*dto.MessageResponse, error) {
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

	// Get sender name
	sender, err := s.userRepo.FindByID(message.SenderID)
	if err == nil {
		response.SenderName = sender.Email // Or user name if available
	}

	// Build reply to message
	if message.ReplyTo != nil {
		replyTo, err := s.buildMessageResponse(message.ReplyTo)
		if err == nil {
			response.ReplyTo = replyTo
		}
	}

	// Build forward from message
	if message.ForwardFrom != nil {
		forwardFrom, err := s.buildMessageResponse(message.ForwardFrom)
		if err == nil {
			response.ForwardFrom = forwardFrom
		}
	}

	// Build reactions
	var reactionResponses []*dto.ReactionResponse
	for _, reaction := range message.Reactions {
		user, err := s.userRepo.FindByID(reaction.UserID)
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

	// Build attachments
	var attachmentResponses []*dto.AttachmentResponse
	for _, attachment := range message.Attachments {
		att := attachment // Создаем локальную копию
		attachmentResponses = append(attachmentResponses, s.buildAttachmentResponse(&att))
	}
	response.Attachments = attachmentResponses

	// Build read receipts
	var readBy []string
	for _, receipt := range message.ReadReceipts {
		readBy = append(readBy, receipt.UserID)
	}
	response.ReadBy = readBy

	return response, nil
}

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

func (s *chatService) processAttachment(userID string, file *multipart.FileHeader) (*chat.MessageAttachment, error) {
	// Validate file size
	if file.Size > FileConfig.MaxSize {
		return nil, appErrors.ErrFileTooLarge
	}

	// Validate file type
	if !isValidFileType(file.Header.Get("Content-Type")) {
		return nil, appErrors.ErrInvalidFileType
	}

	// Generate file path
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), generateRandomString(8), fileExt)
	filePath := filepath.Join(FileConfig.StoragePath, fileName)

	// In real implementation, save the file
	// For now, create attachment record

	return &chat.MessageAttachment{
		UploaderID: userID,
		FileType:   getFileTypeFromMIME(file.Header.Get("Content-Type")),
		MimeType:   file.Header.Get("Content-Type"),
		FileName:   file.Filename,
		URL:        filePath,
		Size:       file.Size,
	}, nil
}

func (s *chatService) notifyNewMessage(dialogID, senderID, messageID string) {
	// Get dialog participants
	participants, err := s.chatRepo.FindParticipantsByDialog(dialogID)
	if err != nil {
		return
	}

	// Send notifications to participants except sender
	for _, participant := range participants {
		if participant.UserID != senderID && !participant.IsMuted {
			// Get sender info for notification
			sender, err := s.userRepo.FindByID(senderID)
			if err != nil {
				continue
			}

			// Send notification
			s.notificationRepo.CreateNewMessageNotification(
				participant.UserID,
				sender.Email,
				dialogID,
			)
		}
	}
}

// Utility functions

func isValidMessageType(messageType string) bool {
	validTypes := map[string]bool{
		"text":    true,
		"image":   true,
		"video":   true,
		"file":    true,
		"system":  true,
		"forward": true,
	}
	return validTypes[messageType]
}

func isValidFileType(mimeType string) bool {
	for _, allowedType := range FileConfig.AllowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

func getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	} else {
		return "file"
	}
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

func calculateTotalPages(total int64, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	pages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		pages++
	}
	return pages
}

func generateRandomString(length int) string {
	// Implementation for generating random string
	// For now, return a placeholder
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		// In real implementation, use crypto/rand
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}

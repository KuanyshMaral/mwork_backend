package chat

import (
	"errors"
	"github.com/google/uuid"
	modelChat "mwork_backend/internal/models/chat"
	repoChat "mwork_backend/internal/repositories/chat"
	"time"
)

type ChatService struct {
	Dialogs      *repoChat.DialogRepository
	Participants *repoChat.DialogParticipantRepository
	Messages     *repoChat.MessageRepository
	ReadReceipts *repoChat.MessageReadReceiptRepository
	Attachments  *AttachmentService
}

func NewChatService(
	dialogs *repoChat.DialogRepository,
	participants *repoChat.DialogParticipantRepository,
	messages *repoChat.MessageRepository,
	readReceipts *repoChat.MessageReadReceiptRepository,
) *ChatService {
	return &ChatService{
		Dialogs:      dialogs,
		Participants: participants,
		Messages:     messages,
		ReadReceipts: readReceipts,
	}
}

type SendMessageInput struct {
	DialogID      string
	SenderID      string
	Content       string
	ReplyToID     *string
	ForwardFrom   *string
	AttachmentIDs []string
	Attachments   []AttachmentInput
}

func (s *ChatService) SendMessage(input SendMessageInput) (*modelChat.Message, error) {
	isMember, err := s.Participants.IsUserInDialog(input.SenderID, input.DialogID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a participant of the dialog")
	}

	message := &modelChat.Message{
		ID:            uuid.New().String(),
		DialogID:      input.DialogID,
		SenderID:      input.SenderID,
		Content:       input.Content,
		ReplyToID:     input.ReplyToID,
		ForwardFromID: input.ForwardFrom,
		CreatedAt:     time.Now(),
	}

	if err := s.Messages.Create(message); err != nil {
		return nil, err
	}

	if s.Attachments != nil && len(input.Attachments) > 0 {
		if _, err := s.Attachments.AddToMessage(message.ID, input.SenderID, input.Attachments); err != nil {
			return nil, err
		}
	}

	if err := s.Dialogs.UpdateLastMessage(input.DialogID, message.ID); err != nil {
		return nil, err
	}

	receipt := &modelChat.MessageReadReceipt{
		UserID:    input.SenderID,
		MessageID: message.ID,
		ReadAt:    time.Now(),
	}
	_ = s.ReadReceipts.Create(receipt)

	return message, nil
}

type CreateDialogInput struct {
	CreatorID      string
	ParticipantIDs []string
	Title          *string
	IsGroup        bool
}

func (s *ChatService) CreateDialog(input CreateDialogInput) (*modelChat.Dialog, error) {
	if len(input.ParticipantIDs) == 0 {
		return nil, errors.New("must have at least one participant")
	}

	dialog := &modelChat.Dialog{
		ID:        uuid.New().String(),
		IsGroup:   input.IsGroup,
		Title:     input.Title,
		CreatedAt: time.Now(),
	}

	err := s.Dialogs.Create(dialog)
	if err != nil {
		return nil, err
	}

	participants := make([]modelChat.DialogParticipant, 0, len(input.ParticipantIDs)+1)
	participants = append(participants, modelChat.DialogParticipant{
		DialogID: dialog.ID,
		UserID:   input.CreatorID,
		Role:     "admin",
		JoinedAt: time.Now(),
	})

	for _, pid := range input.ParticipantIDs {
		if pid == input.CreatorID {
			continue
		}
		participants = append(participants, modelChat.DialogParticipant{
			DialogID: dialog.ID,
			UserID:   pid,
			Role:     "member",
			JoinedAt: time.Now(),
		})
	}

	err = s.Participants.CreateMany(participants)
	if err != nil {
		return nil, err
	}

	return dialog, nil
}

func (s *ChatService) MarkAllAsRead(userID, dialogID string) error {
	messages, err := s.Messages.GetByDialog(dialogID)
	if err != nil {
		return err
	}

	receipts := make([]modelChat.MessageReadReceipt, 0)
	for _, msg := range messages {
		exists, err := s.ReadReceipts.Exists(userID, msg.ID)
		if err != nil {
			return err
		}
		if !exists {
			receipts = append(receipts, modelChat.MessageReadReceipt{
				UserID:    userID,
				MessageID: msg.ID,
				ReadAt:    time.Now(),
			})
		}
	}
	if len(receipts) > 0 {
		return s.ReadReceipts.CreateMany(receipts)
	}
	return nil
}

func (s *ChatService) LeaveDialog(userID, dialogID string) error {
	return s.Participants.LeaveDialog(userID, dialogID)
}

func (s *ChatService) GetMessages(userID, dialogID string) ([]modelChat.Message, error) {
	isMember, err := s.Participants.IsUserInDialog(userID, dialogID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}
	return s.Messages.GetByDialog(dialogID)
}

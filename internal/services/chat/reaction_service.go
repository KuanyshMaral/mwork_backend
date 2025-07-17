package chat

import (
	modelChat "mwork_front_fn/internal/models/chat"
	repoChat "mwork_front_fn/internal/repositories/chat"
)

type ReactionService struct {
	Repo *repoChat.MessageReactionRepository
}

func NewReactionService(repo *repoChat.MessageReactionRepository) *ReactionService {
	return &ReactionService{Repo: repo}
}

// Add добавляет реакцию пользователя к сообщению
func (s *ReactionService) Add(userID, messageID, emoji string) error {
	reaction := &modelChat.MessageReaction{
		UserID:    userID,
		MessageID: messageID,
		Emoji:     emoji,
	}
	return s.Repo.Add(reaction)
}

// Remove удаляет реакцию пользователя
func (s *ReactionService) Remove(userID, messageID, emoji string) error {
	return s.Repo.Remove(userID, messageID, emoji)
}

// Toggle включение/выключение реакции
func (s *ReactionService) Toggle(userID, messageID, emoji string) error {
	return s.Repo.ToggleReaction(userID, messageID, emoji)
}

// GetByMessageID возвращает все реакции на сообщение
func (s *ReactionService) GetByMessageID(messageID string) ([]modelChat.MessageReaction, error) {
	return s.Repo.GetByMessageID(messageID)
}

package services

import (
	"context"
	"errors"
	"mwork_backend/internal/auth"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type RefreshTokenService struct {
	repo     *repositories.RefreshTokenRepository
	userRepo *repositories.UserRepository // ⬅️ Добавили для получения role
}

func NewRefreshTokenService(repo *repositories.RefreshTokenRepository, userRepo *repositories.UserRepository) *RefreshTokenService {
	return &RefreshTokenService{
		repo:     repo,
		userRepo: userRepo,
	}
}

// Сохраняет refresh token в базу
func (s *RefreshTokenService) Create(ctx context.Context, userID, token string, expires time.Time) error {
	rt := &models.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
	return s.repo.Save(ctx, rt)
}

// Удаляет refresh token (logout)
func (s *RefreshTokenService) DeleteByToken(ctx context.Context, token string) error {
	return s.repo.Delete(ctx, token)
}

// Проверяет существует ли токен
func (s *RefreshTokenService) IsValid(ctx context.Context, token string) (bool, error) {
	return s.repo.Exists(ctx, token)
}

// Возвращает userID по refresh токену
func (s *RefreshTokenService) GetUserID(ctx context.Context, token string) (string, error) {
	return s.repo.GetUserIDByToken(ctx, token)
}

// Генерирует новый access token по refresh токену
func (s *RefreshTokenService) ValidateAndRefresh(ctx context.Context, refreshToken string) (string, error) {
	token, err := s.repo.GetByToken(ctx, refreshToken)
	if err != nil || token == nil {
		return "", errors.New("invalid refresh token")
	}
	if time.Now().After(token.ExpiresAt) {
		return "", errors.New("refresh token expired")
	}

	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil || user == nil {
		return "", errors.New("user not found")
	}

	newAccessToken, err := auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", errors.New("failed to generate access token")
	}

	return newAccessToken, nil
}

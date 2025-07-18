package services

import (
	"context"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
)

type ModelProfileService struct {
	repo repositories.ModelProfileRepository
}

func NewModelProfileService(repo repositories.ModelProfileRepository) *ModelProfileService {
	return &ModelProfileService{repo: repo}
}

func (s *ModelProfileService) CreateProfile(ctx context.Context, profile *models.ModelProfile) error {
	return s.repo.Create(ctx, profile)
}

func (s *ModelProfileService) GetProfileByUserID(ctx context.Context, userID string) (*models.ModelProfile, error) {
	return s.repo.GetByUserID(ctx, userID)
}

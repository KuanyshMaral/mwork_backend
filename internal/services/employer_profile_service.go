package services

import (
	"context"
	"mwork_front_fn/internal/models"
	"mwork_front_fn/internal/repositories"
)

type EmployerProfileService struct {
	repo *repositories.EmployerProfileRepository
}

func NewEmployerProfileService(repo *repositories.EmployerProfileRepository) *EmployerProfileService {
	return &EmployerProfileService{repo: repo}
}

func (s *EmployerProfileService) CreateProfile(ctx context.Context, profile *models.EmployerProfile) error {
	return s.repo.Create(ctx, profile)
}

func (s *EmployerProfileService) GetProfileByUserID(ctx context.Context, userID string) (*models.EmployerProfile, error) {
	return s.repo.GetByUserID(ctx, userID)
}

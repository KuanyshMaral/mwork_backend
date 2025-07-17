package services

import (
	"context"
	"mwork_front_fn/internal/models"
	"mwork_front_fn/internal/repositories"
)

type CastingService struct {
	repo *repositories.CastingRepository
}

func NewCastingService(repo *repositories.CastingRepository) *CastingService {
	return &CastingService{repo: repo}
}

func (s *CastingService) Create(ctx context.Context, casting *models.Casting) error {
	return s.repo.Create(ctx, casting)
}

func (s *CastingService) GetByID(ctx context.Context, id string) (*models.Casting, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CastingService) Update(ctx context.Context, casting *models.Casting) error {
	return s.repo.Update(ctx, casting)
}

func (s *CastingService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *CastingService) ListByEmployer(ctx context.Context, employerID string) ([]*models.Casting, error) {
	return s.repo.ListByEmployer(ctx, employerID)
}

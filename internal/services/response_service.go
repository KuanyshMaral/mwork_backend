package services

import (
	"context"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
)

type ResponseService struct {
	repo *repositories.ResponseRepository
}

func NewResponseService(repo *repositories.ResponseRepository) *ResponseService {
	return &ResponseService{repo: repo}
}

func (s *ResponseService) Create(ctx context.Context, res *models.CastingResponse) error {
	return s.repo.Create(ctx, res)
}

func (s *ResponseService) GetByID(ctx context.Context, id string) (*models.CastingResponse, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ResponseService) ListByCasting(ctx context.Context, castingID string) ([]models.CastingResponse, error) {
	return s.repo.ListByCasting(ctx, castingID)
}

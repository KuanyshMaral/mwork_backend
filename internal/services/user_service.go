package services

import (
	"context"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *UserService) FindByID(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, user *models.User) error {
	return s.repo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

package subscription

import (
	"context"
	"mwork_backend/internal/dto"
	subscriptionrepo "mwork_backend/internal/repositories/subscription"
	"time"
)

type PlanService struct {
	repo *subscriptionrepo.SubscriptionPlanRepository
}

func NewPlanService(repo *subscriptionrepo.SubscriptionPlanRepository) *PlanService {
	return &PlanService{repo: repo}
}

func (s *PlanService) GetAllPlans(ctx context.Context) ([]dto.PlanBase, error) {
	return s.repo.GetAll(ctx)
}

func (s *PlanService) GetPlansWithStats(ctx context.Context) ([]dto.PlanWithStats, error) {
	return s.repo.GetPlansWithStats(ctx)
}

func (s *PlanService) GetRevenueByPeriod(ctx context.Context, start, end time.Time) ([]dto.PlanRevenue, error) {
	return s.repo.GetRevenueByPeriod(ctx, start, end)
}

func (s *PlanService) CreatePlan(ctx context.Context, plan *dto.PlanBase) error {
	return s.repo.Create(ctx, plan)
}

func (s *PlanService) DeletePlan(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *PlanService) GetByID(ctx context.Context, id string) (*dto.PlanBase, error) {
	return s.repo.GetByID(ctx, id)
}

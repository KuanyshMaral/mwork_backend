package services

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/datatypes"
	"mwork_front_fn/backend/models"
	"mwork_front_fn/backend/repositories"
	"time"
)

type SubscriptionService struct {
	repo *repositories.SubscriptionRepository
}

func NewSubscriptionService(repo *repositories.SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

// Получить все доступные планы
func (s *SubscriptionService) GetPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	return s.repo.GetAllPlans(ctx)
}

// Получить план по ID
func (s *SubscriptionService) GetPlan(ctx context.Context, id string) (*models.SubscriptionPlan, error) {
	return s.repo.GetPlanByID(ctx, id)
}

// Получить подписку пользователя
func (s *SubscriptionService) GetUserSubscription(ctx context.Context, userID string) (*models.UserSubscription, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// Оформить подписку пользователю
func (s *SubscriptionService) CreateSubscription(ctx context.Context, userID, planID string, autoRenew bool) (*models.UserSubscription, error) {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	end := now.AddDate(0, 1, 0)
	if plan.Duration == "yearly" {
		end = now.AddDate(1, 0, 0)
	}

	usageJSON, err := json.Marshal(map[string]int{})
	if err != nil {
		return nil, err
	}

	sub := &models.UserSubscription{
		UserID:    userID,
		PlanID:    planID,
		Status:    "active",
		StartDate: now,
		EndDate:   end,
		AutoRenew: autoRenew,
		Usage:     datatypes.JSON(usageJSON),
	}

	err = s.repo.CreateUserSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

// Проверка лимита использования
func (s *SubscriptionService) CheckUsageLimit(ctx context.Context, userID string, action string) (bool, int, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return false, 0, err
	}

	var limits map[string]int
	if err := json.Unmarshal(plan.Limits, &limits); err != nil {
		return false, 0, err
	}

	limit, ok := limits[action]
	if !ok {
		return true, -1, nil // неограниченно
	}

	var usage map[string]int
	if err := json.Unmarshal(sub.Usage, &usage); err != nil {
		return false, 0, err
	}

	used := usage[action]
	remaining := limit - used
	return remaining > 0, remaining, nil
}

// Увеличить счетчик использования
func (s *SubscriptionService) IncrementUsage(ctx context.Context, userID string, action string, amount int) error {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	var usage map[string]int
	if err := json.Unmarshal(sub.Usage, &usage); err != nil {
		return err
	}

	usage[action] += amount

	updatedJSON, err := json.Marshal(usage)
	if err != nil {
		return err
	}

	return s.repo.UpdateUsage(ctx, sub.ID, datatypes.JSON(updatedJSON))
}

// Обновить статус подписки
func (s *SubscriptionService) UpdateStatus(ctx context.Context, subID, status string) error {
	if status != "active" && status != "expired" && status != "cancelled" {
		return errors.New("invalid status")
	}
	return s.repo.UpdateStatus(ctx, subID, status)
}

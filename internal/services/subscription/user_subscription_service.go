package subscription

import (
	"context"
	"mwork_backend/internal/dto"
	"mwork_backend/internal/models"
	subscriptionrepo "mwork_backend/internal/repositories/subscription"
	"time"
)

type UserSubscriptionService struct {
	repo *subscriptionrepo.UserSubscriptionRepository
}

func NewUserSubscriptionService(repo *subscriptionrepo.UserSubscriptionRepository) *UserSubscriptionService {
	return &UserSubscriptionService{repo: repo}
}

// Создание подписки
func (s *UserSubscriptionService) Create(ctx context.Context, sub *models.UserSubscription) error {
	return s.repo.Create(ctx, sub)
}

// Отмена текущей активной подписки пользователем
func (s *UserSubscriptionService) CancelSubscription(ctx context.Context, userID string) error {
	return s.repo.CancelSubscription(ctx, userID)
}

// Принудительная отмена подписки админом
func (s *UserSubscriptionService) ForceCancelSubscription(ctx context.Context, subscriptionID string) error {
	return s.repo.ForceCancel(ctx, subscriptionID)
}

// Принудительное продление подписки (админ)
func (s *UserSubscriptionService) ForceExtendSubscription(ctx context.Context, subscriptionID string, newEndDate time.Time) error {
	return s.repo.ForceExtend(ctx, subscriptionID, newEndDate)
}

// Получение всех подписок юзера
func (s *UserSubscriptionService) GetAllUserSubscriptions(ctx context.Context, status *string) ([]models.UserSubscription, error) {
	return s.repo.GetAll(ctx, status)
}

// Получение статистики по подпискам (по планам)
func (s *UserSubscriptionService) GetStats(ctx context.Context) ([]dto.PlanStats, error) {
	return s.repo.GetStatsByPlan(ctx)
}

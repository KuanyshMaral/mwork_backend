package workers

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

type SubscriptionWorker struct {
	db *gorm.DB
}

func NewSubscriptionWorker(db *gorm.DB) *SubscriptionWorker {
	return &SubscriptionWorker{db: db}
}

// Start запускает фоновые задачи для подписок
func (w *SubscriptionWorker) Start(ctx context.Context) {
	// Проверка истечения подписок каждые 6 часов
	go w.checkExpiredSubscriptions(ctx)

	// Сброс лимитов использования каждый день в полночь
	go w.resetUsageLimits(ctx)
}

// checkExpiredSubscriptions помечает истекшие подписки
func (w *SubscriptionWorker) checkExpiredSubscriptions(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Subscription worker stopped")
			return
		case <-ticker.C:
			result := w.db.Exec(`
				UPDATE user_subscriptions 
				SET status = 'expired', updated_at = NOW()
				WHERE status = 'active' 
				AND end_date < NOW()
			`)
			if result.Error != nil {
				log.Printf("Error checking expired subscriptions: %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("Marked %d subscriptions as expired", result.RowsAffected)
			}
		}
	}
}

// resetUsageLimits сбрасывает счетчики использования
func (w *SubscriptionWorker) resetUsageLimits(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Вычисляем время до следующей полуночи
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			duration := next.Sub(now)

			timer := time.NewTimer(duration)
			<-timer.C

			// Сбрасываем счетчики для подписок, у которых прошел период
			result := w.db.Exec(`
				UPDATE user_subscriptions 
				SET usage = jsonb_set(
					usage, 
					'{publications_used}', 
					'0'::jsonb
				),
				updated_at = NOW()
				WHERE status = 'active'
				AND (usage->>'last_reset')::timestamp < NOW() - INTERVAL '30 days'
			`)
			if result.Error != nil {
				log.Printf("Error resetting usage limits: %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("Reset usage limits for %d subscriptions", result.RowsAffected)
			}
		}
	}
}

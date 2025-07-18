package subscription

import (
	"context"
	"database/sql"
	"mwork_backend/internal/dto"
	"mwork_backend/internal/models"
	"time"
)

type UserSubscriptionRepository struct {
	db *sql.DB
}

func NewUserSubscriptionRepository(db *sql.DB) *UserSubscriptionRepository {
	return &UserSubscriptionRepository{db: db}
}

// Получить все подписки, возможно с фильтрацией по статусу
func (r *UserSubscriptionRepository) GetAll(ctx context.Context, status *string) ([]models.UserSubscription, error) {
	query := `SELECT * FROM user_subscriptions`
	var args []interface{}
	if status != nil {
		query += ` WHERE status = $1`
		args = append(args, *status)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.UserSubscription
	for rows.Next() {
		var sub models.UserSubscription
		if err := scanUserSubscription(rows, &sub); err != nil {
			return nil, err
		}
		result = append(result, sub)
	}
	return result, nil
}

// Найти активную подписку пользователя
func (r *UserSubscriptionRepository) FindActiveByUserID(ctx context.Context, userID string) (*models.UserSubscription, error) {
	query := `
		SELECT * FROM user_subscriptions
		WHERE user_id = $1 AND status = 'active' AND end_date > NOW()
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, userID)
	var sub models.UserSubscription
	if err := scanUserSubscription(row, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// Принудительное отключение подписки
func (r *UserSubscriptionRepository) ForceCancel(ctx context.Context, id string) error {
	query := `UPDATE user_subscriptions SET status = 'cancelled', cancelled_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Принудительное продление подписки
func (r *UserSubscriptionRepository) ForceExtend(ctx context.Context, id string, newEnd time.Time) error {
	query := `UPDATE user_subscriptions SET end_date = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, newEnd, id)
	return err
}

// Найти подписки с автообновлением, истекающие в течение N минут/часов
func (r *UserSubscriptionRepository) FindExpiringAutoRenew(ctx context.Context, within time.Duration) ([]models.UserSubscription, error) {
	query := `
		SELECT * FROM user_subscriptions
		WHERE auto_renew = true AND end_date <= NOW() + $1::interval AND status = 'active'
	`
	rows, err := r.db.QueryContext(ctx, query, within.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.UserSubscription
	for rows.Next() {
		var sub models.UserSubscription
		if err := scanUserSubscription(rows, &sub); err != nil {
			return nil, err
		}
		result = append(result, sub)
	}
	return result, nil
}

// 📊 Получить число пользователей по каждому плану
func (r *UserSubscriptionRepository) GetStatsByPlan(ctx context.Context) ([]dto.PlanStats, error) {
	query := `
		SELECT plan_id, COUNT(*) AS total_users
		FROM user_subscriptions
		WHERE status = 'active'
		GROUP BY plan_id
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []dto.PlanStats
	for rows.Next() {
		var s dto.PlanStats
		if err := rows.Scan(&s.PlanID, &s.TotalUsers); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// 📊 Доход по каждому плану (для админ-аналитики)
func (r *UserSubscriptionRepository) GetRevenueByPlan(ctx context.Context) ([]dto.PlanRevenue, error) {
	query := `
		SELECT
			us.plan_id,
			sp.name,
			COUNT(*) AS purchase_count,
			COALESCE(SUM(sp.price), 0) AS total_revenue
		FROM user_subscriptions us
		INNER JOIN subscription_plans sp ON sp.id = us.plan_id
		WHERE us.status = 'active'
		GROUP BY us.plan_id, sp.name
		ORDER BY total_revenue DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revenue []dto.PlanRevenue
	for rows.Next() {
		var r dto.PlanRevenue
		if err := rows.Scan(&r.PlanID, &r.PlanName, &r.PurchaseCount, &r.TotalRevenue); err != nil {
			return nil, err
		}
		revenue = append(revenue, r)
	}
	return revenue, nil
}

func scanUserSubscription(scanner interface {
	Scan(dest ...interface{}) error
}, sub *models.UserSubscription) error {
	return scanner.Scan(
		&sub.ID,
		&sub.UserID,
		&sub.PlanID,
		&sub.StartDate,
		&sub.EndDate,
		&sub.Status,
		&sub.AutoRenew,
		&sub.CreatedAt,
		&sub.CancelledAt,
	)
}

func (r *UserSubscriptionRepository) Create(ctx context.Context, sub *models.UserSubscription) error {
	query := `INSERT INTO user_subscriptions (id, user_id, plan_id, start_date, end_date, status, auto_renew, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID,
		sub.UserID,
		sub.PlanID,
		sub.StartDate,
		sub.EndDate,
		sub.Status,
		sub.AutoRenew,
		sub.CreatedAt,
	)
	return err
}

func (r *UserSubscriptionRepository) CancelSubscription(ctx context.Context, id string) error {
	query := `UPDATE user_subscriptions SET status = 'cancelled', cancelled_at = NOW() WHERE id = $1 AND status = 'active'`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

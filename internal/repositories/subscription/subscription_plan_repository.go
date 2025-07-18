package subscription

import (
	"context"
	"database/sql"
	"mwork_backend/internal/dto"
	"time"
)

type SubscriptionPlanRepository struct {
	db *sql.DB
}

func NewSubscriptionPlanRepository(db *sql.DB) *SubscriptionPlanRepository {
	return &SubscriptionPlanRepository{db: db}
}

// Вернуть все планы с количеством подписчиков на каждом
func (r *SubscriptionPlanRepository) GetPlansWithStats(ctx context.Context) ([]dto.PlanWithStats, error) {
	query := `
				SELECT 
					sp.id, sp.name, sp.price, sp.currency, sp.duration_days, 
					COUNT(us.id) AS user_count
				FROM subscription_plans sp
				LEFT JOIN user_subscriptions us ON us.plan_id = sp.id AND us.status = 'active'
				GROUP BY sp.id
				ORDER BY sp.price ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []dto.PlanWithStats
	for rows.Next() {
		var plan dto.PlanWithStats
		if err := rows.Scan(
			&plan.ID,
			&plan.Name,
			&plan.Price,
			&plan.Currency,
			&plan.DurationDays,
			&plan.UserCount,
		); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// Получить сумму доходов по каждому плану за период
func (r *SubscriptionPlanRepository) GetRevenueByPeriod(ctx context.Context, start, end time.Time) ([]dto.PlanRevenue, error) {
	query := `
		SELECT sp.id, sp.name, 
		       SUM(sp.price) AS total_revenue, 
		       COUNT(us.id) AS purchases
		FROM subscription_plans sp
		JOIN user_subscriptions us 
		     ON us.plan_id = sp.id
		WHERE us.created_at BETWEEN $1 AND $2
		  AND us.status = 'active'
		GROUP BY sp.id
	`
	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []dto.PlanRevenue
	for rows.Next() {
		var stat dto.PlanRevenue
		if err := rows.Scan(
			&stat.PlanID,
			&stat.PlanName,
			&stat.TotalRevenue,
			&stat.PurchaseCount,
		); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

func (r *SubscriptionPlanRepository) GetAll(ctx context.Context) ([]dto.PlanBase, error) {
	query := `SELECT id, name, price, currency, duration_days FROM subscription_plans ORDER BY price ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []dto.PlanBase
	for rows.Next() {
		var plan dto.PlanBase
		if err := rows.Scan(
			&plan.ID,
			&plan.Name,
			&plan.Price,
			&plan.Currency,
			&plan.DurationDays,
		); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func (r *SubscriptionPlanRepository) Create(ctx context.Context, plan *dto.PlanBase) error {
	query := `INSERT INTO subscription_plans (id, name, price, currency, duration_days)
	          VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, plan.ID, plan.Name, plan.Price, plan.Currency, plan.DurationDays)
	return err
}

func (r *SubscriptionPlanRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM subscription_plans WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SubscriptionPlanRepository) GetByID(ctx context.Context, id string) (*dto.PlanBase, error) {
	query := `SELECT id, name, price, currency, duration_days FROM subscription_plans WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var plan dto.PlanBase
	err := row.Scan(&plan.ID, &plan.Name, &plan.Price, &plan.Currency, &plan.DurationDays)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

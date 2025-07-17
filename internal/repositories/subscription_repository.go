package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"gorm.io/datatypes"
	"mwork_front_fn/internal/models"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// ====== Subscription Plan ======

func (r *SubscriptionRepository) GetAllPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, price, currency, duration, features, limits, created_at, updated_at FROM subscription_plans
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.SubscriptionPlan
	for rows.Next() {
		var plan models.SubscriptionPlan
		var featuresJSON []byte
		var limitsJSON []byte

		if err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Price, &plan.Currency, &plan.Duration,
			&featuresJSON, &limitsJSON, &plan.CreatedAt, &plan.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
			return nil, err
		}

		plans = append(plans, plan)
	}
	return plans, nil
}

func (r *SubscriptionRepository) GetPlanByID(ctx context.Context, id string) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	var featuresJSON, limitsJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, price, currency, duration, features, limits, created_at, updated_at
		FROM subscription_plans WHERE id = $1
	`, id).Scan(
		&plan.ID, &plan.Name, &plan.Price, &plan.Currency, &plan.Duration,
		&featuresJSON, &limitsJSON, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
		return nil, err
	}

	return &plan, nil
}

// ====== User Subscription ======

func (r *SubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*models.UserSubscription, error) {
	var sub models.UserSubscription
	var usageJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, plan_id, status, start_date, end_date, auto_renew, usage, created_at, updated_at
		FROM user_subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.AutoRenew, &usageJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(usageJSON, &sub.Usage); err != nil {
		return nil, err
	}

	return &sub, nil
}

func (r *SubscriptionRepository) CreateUserSubscription(ctx context.Context, sub *models.UserSubscription) error {
	usageJSON, err := json.Marshal(sub.Usage)
	if err != nil {
		return err
	}

	return r.db.QueryRowContext(ctx, `
		INSERT INTO user_subscriptions (user_id, plan_id, status, start_date, end_date, auto_renew, usage)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, sub.UserID, sub.PlanID, sub.Status, sub.StartDate, sub.EndDate, sub.AutoRenew, usageJSON).Scan(
		&sub.ID, &sub.CreatedAt, &sub.UpdatedAt,
	)
}

func (r *SubscriptionRepository) UpdateUsage(ctx context.Context, subID string, usage datatypes.JSON) error {
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE user_subscriptions SET usage = $1, updated_at = NOW() WHERE id = $2
	`, usageJSON, subID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("no rows updated")
	}

	return nil
}

func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, subID string, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE user_subscriptions SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, subID)
	return err
}

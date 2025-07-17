package dto

import "time"

// SubscriptionPlanResponse — DTO для плана подписки
type SubscriptionPlanResponse struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Price    float64                `json:"price"`
	Currency string                 `json:"currency"`
	Duration string                 `json:"duration"`
	Features map[string]interface{} `json:"features"`
	Limits   map[string]interface{} `json:"limits"`
}

// UserSubscriptionResponse — DTO для подписки пользователя
type UserSubscriptionResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	PlanID    string                 `json:"plan_id"`
	Status    string                 `json:"status"`
	StartDate time.Time              `json:"start_date"`
	EndDate   time.Time              `json:"end_date"`
	AutoRenew bool                   `json:"auto_renew"`
	Usage     map[string]interface{} `json:"usage"`
}

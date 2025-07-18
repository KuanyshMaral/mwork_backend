package dto

// SubscriptionPlanDTO — версия для Swagger
type SubscriptionPlanDTO struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Price     float64                `json:"price"`
	Currency  string                 `json:"currency"`
	Duration  string                 `json:"duration"`
	Features  map[string]interface{} `json:"features"` // JSON-friendly
	Limits    map[string]interface{} `json:"limits"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// UserSubscriptionDTO — версия для Swagger
type UserSubscriptionDTO struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	PlanID      string                 `json:"plan_id"`
	Status      string                 `json:"status"`
	InvID       string                 `json:"inv_id"`
	Plan        SubscriptionPlanDTO    `json:"plan"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	AutoRenew   bool                   `json:"auto_renew"`
	Usage       map[string]interface{} `json:"usage"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	CancelledAt *string                `json:"cancelled_at,omitempty"`
}

type CreateSubscriptionRequest struct {
	PlanID    string `json:"plan_id" example:"premium"`
	AutoRenew bool   `json:"auto_renew" example:"true"`
}

type ForceExtendSubscriptionRequest struct {
	NewEndDate string `json:"new_end_date"`
}

type InitiatePaymentRequest struct {
	PlanID string `json:"plan_id"`
}

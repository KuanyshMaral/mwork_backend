package models

import "time"

type CreatePlanRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description,omitempty"`
	Price       float64        `json:"price" binding:"required,min=0"`
	Currency    string         `json:"currency" binding:"required"`
	Duration    string         `json:"duration" binding:"required"`
	Features    map[string]any `json:"features" binding:"required"`
	Limits      map[string]int `json:"limits" binding:"required"`
	IsActive    bool           `json:"is_active"`
}

type UpdatePlanRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Price       *float64       `json:"price,omitempty"`
	Currency    *string        `json:"currency,omitempty"`
	Duration    *string        `json:"duration,omitempty"`
	Features    map[string]any `json:"features,omitempty"`
	Limits      map[string]int `json:"limits,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
}

type PaymentResponse struct {
	PaymentID  string    `json:"payment_id"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	PaymentURL string    `json:"payment_url,omitempty"`
	InvoiceID  string    `json:"invoice_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type RobokassaInitResponse struct {
	PaymentURL string  `json:"payment_url"`
	InvoiceID  string  `json:"invoice_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
}

type RobokassaCallbackData struct {
	InvID         string  `json:"InvId"`
	OutSum        float64 `json:"OutSum"`
	Signature     string  `json:"SignatureValue"`
	Currency      string  `json:"IncCurrLabel"`
	Email         string  `json:"Email"`
	Fee           float64 `json:"Fee"`
	PaymentMethod string  `json:"PaymentMethod"`
}

type PaymentStatusResponse struct {
	PaymentID string    `json:"payment_id"`
	Status    string    `json:"status"`
	Amount    float64   `json:"amount"`
	PaidAt    time.Time `json:"paid_at,omitempty"`
	InvoiceID string    `json:"invoice_id"`
}

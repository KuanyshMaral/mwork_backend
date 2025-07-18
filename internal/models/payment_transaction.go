package models

import (
	"github.com/google/uuid"
	"time"
)

type PaymentTransaction struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID         uuid.UUID
	SubscriptionID uuid.UUID
	Amount         float64
	Status         string // "pending", "paid", "failed"
	InvID          string `gorm:"uniqueIndex"` // тот же, что передавали в Robokassa
	PaidAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

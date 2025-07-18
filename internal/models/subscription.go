package models

import (
	"time"

	"gorm.io/datatypes"
)

type SubscriptionPlan struct {
	ID        string `gorm:"primaryKey"`
	Name      string
	Price     float64
	Currency  string
	Duration  string
	Features  datatypes.JSON `gorm:"type:jsonb"` // ✅ JSONB
	Limits    datatypes.JSON `gorm:"type:jsonb"` // ✅ JSONB
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserSubscription struct {
	ID          string `gorm:"primaryKey"`
	UserID      string
	PlanID      string
	Status      string
	InvID       string `gorm:"uniqueIndex"` // ID от Robokassa
	Plan        SubscriptionPlan
	StartDate   time.Time
	EndDate     time.Time
	AutoRenew   bool
	Usage       datatypes.JSON `gorm:"type:jsonb"` // ✅ JSONB
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
}

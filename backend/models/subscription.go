package models

import (
	"gorm.io/datatypes"
	"time"
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
	ID        string `gorm:"primaryKey"`
	UserID    string
	PlanID    string
	Status    string
	StartDate time.Time
	EndDate   time.Time
	AutoRenew bool
	Usage     datatypes.JSON `gorm:"type:jsonb"` // ✅ JSONB
	CreatedAt time.Time
	UpdatedAt time.Time
}

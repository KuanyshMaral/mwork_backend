package models

import (
	"gorm.io/datatypes"
	"time"
)

type SubscriptionPlan struct {
	BaseModel
	Name          string         `gorm:"not null"`
	Price         float64        `gorm:"not null"`
	Currency      string         `gorm:"default:'KZT'"`
	Duration      string         `gorm:"not null"`   // "monthly", "yearly"
	Features      datatypes.JSON `gorm:"type:jsonb"` // {"premium_support": true, ...}
	Limits        datatypes.JSON `gorm:"type:jsonb"` // {"publications": 5, "responses": 10}
	IsActive      bool           `gorm:"default:true"`
	PaymentStatus string
}

type UserSubscription struct {
	BaseModel
	UserID       string             `gorm:"not null;index"`
	PlanID       string             `gorm:"not null;index"`
	Status       SubscriptionStatus `gorm:"default:'active'"`
	InvID        string             `gorm:"uniqueIndex"` // ID от Robokassa
	CurrentUsage datatypes.JSON     `gorm:"type:jsonb"`  // {"publications": 2, "responses": 5}
	StartDate    time.Time
	EndDate      time.Time
	AutoRenew    bool `gorm:"default:true"`
	CancelledAt  *time.Time

	// Relations
	Plan SubscriptionPlan `gorm:"foreignKey:PlanID"`
}

type PaymentTransaction struct {
	BaseModel
	UserID         string `gorm:"not null;index"`
	SubscriptionID string `gorm:"not null;index"`
	Amount         float64
	Status         PaymentStatus `gorm:"default:'pending'"`
	InvID          string        `gorm:"uniqueIndex"`
	PaidAt         *time.Time

	// Relations
	Subscription UserSubscription `gorm:"foreignKey:SubscriptionID"`
}

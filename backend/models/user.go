package models

import "time"

type User struct {
	ID                string `gorm:"primaryKey;type:uuid"` // UUID primary key
	Email             string `gorm:"uniqueIndex;not null"` // уникальный индекс
	PasswordHash      string `gorm:"not null"`
	Role              string `gorm:"type:varchar(20);default:'model'"`   // model | employer | admin
	Subscription      string `gorm:"type:varchar(20);default:'free'"`    // free | premium | pro
	Status            string `gorm:"type:varchar(20);default:'pending'"` // pending | active | suspended | banned
	IsVerified        bool   `gorm:"default:false"`
	VerificationToken string `gorm:"type:varchar(255)"`
	ResetToken        string `gorm:"type:varchar(255)"`
	ResetTokenExp     time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

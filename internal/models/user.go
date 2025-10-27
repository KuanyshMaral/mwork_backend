package models

import "time"

type User struct {
	BaseModel
	Email             string     `gorm:"uniqueIndex;not null"`
	PasswordHash      string     `gorm:"not null"`
	Role              UserRole   `gorm:"type:varchar(20);not null"`
	Status            UserStatus `gorm:"type:varchar(20);default:'pending'"`
	IsVerified        bool       `gorm:"default:false"`
	VerificationToken string
	ResetToken        string
	ResetTokenExp     *time.Time

	// Relations
	ModelProfile    *ModelProfile     `gorm:"foreignKey:UserID"`
	EmployerProfile *EmployerProfile  `gorm:"foreignKey:UserID"`
	Subscription    *UserSubscription `gorm:"foreignKey:UserID"`
	RefreshTokens   []RefreshToken    `gorm:"foreignKey:UserID"`
}

type RefreshToken struct {
	BaseModel
	UserID    string    `gorm:"not null;index"`
	Token     string    `gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
}

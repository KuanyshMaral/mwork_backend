package models

import "time"

type Review struct {
	BaseModel
	ModelID    string  `gorm:"not null;index"`
	EmployerID string  `gorm:"not null;index"`
	CastingID  *string `gorm:"index"`
	Rating     int     `gorm:"not null;check:rating >= 1 AND rating <= 5"`
	ReviewText string
	CreatedAt  time.Time `gorm:"default:now()"`
	Status     string    `gorm:"default:'pending'"`

	// Relations
	Model    ModelProfile    `gorm:"foreignKey:ModelID"`
	Employer EmployerProfile `gorm:"foreignKey:EmployerID"`
	Casting  *Casting        `gorm:"foreignKey:CastingID"`
}

const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
)

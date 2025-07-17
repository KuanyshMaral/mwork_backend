package models

import (
	"gorm.io/datatypes"
	"time"
)

type Casting struct {
	ID              string `gorm:"primaryKey"`
	EmployerID      string
	Title           string
	Description     string
	PaymentMin      float64
	PaymentMax      float64
	CastingDate     *time.Time
	CastingTime     *string
	Address         *string
	City            string
	Categories      datatypes.JSON `gorm:"type:jsonb"` // ✅ JSONB
	Gender          string
	AgeMin          *int
	AgeMax          *int
	HeightMin       *float64
	HeightMax       *float64
	WeightMin       *float64
	WeightMax       *float64
	ClothingSize    *string
	ShoeSize        *string
	ExperienceLevel *string
	Languages       datatypes.JSON `gorm:"type:jsonb"` // ✅ JSONB
	JobType         string
	Status          string
	Views           int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

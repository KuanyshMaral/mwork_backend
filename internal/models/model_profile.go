package models

import (
	"time"

	"github.com/lib/pq"
)

type ModelProfile struct {
	ID             string
	UserID         string
	Name           string
	Age            int
	Height         float64
	Weight         float64
	Gender         string
	Experience     int
	HourlyRate     float64
	Description    string
	ClothingSize   string
	ShoeSize       string
	City           string
	Languages      pq.StringArray `gorm:"type:text[]" json:"languages" swaggerignore:"true"`
	Categories     pq.StringArray `gorm:"type:text[]" json:"categories" swaggerignore:"true"`
	BarterAccepted bool
	ProfileViews   int
	Rating         float64
	IsPublic       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

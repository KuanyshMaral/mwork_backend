package models

import (
	"github.com/lib/pq"
	"time"
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
	Languages      pq.StringArray `gorm:"type:text[]" json:"languages"`
	Categories     pq.StringArray `gorm:"type:text[]" json:"categories"`
	BarterAccepted bool
	ProfileViews   int
	Rating         float64
	IsPublic       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

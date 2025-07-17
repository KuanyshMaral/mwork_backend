package models

import "time"

type Rating struct {
	ID        string `gorm:"primaryKey"`
	ModelID   string `gorm:"index"` // UUID вместо uint
	Score     int
	Comment   string
	AuthorID  string `gorm:"index"` // UUID вместо uint
	CreatedAt time.Time
}

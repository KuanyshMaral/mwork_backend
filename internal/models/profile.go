package models

import (
	"gorm.io/datatypes"
)

type ModelProfile struct {
	BaseModel
	UserID         string `gorm:"uniqueIndex;not null"`
	Name           string `gorm:"not null"`
	Age            int    `gorm:"not null"`
	Height         float64
	Weight         float64
	Gender         string
	Experience     int // years
	HourlyRate     float64
	Description    string
	ClothingSize   string
	ShoeSize       string
	City           string         `gorm:"not null"`
	Languages      datatypes.JSON `gorm:"type:jsonb"` // ["русский", "английский"]
	Categories     datatypes.JSON `gorm:"type:jsonb"` // ["fashion", "advertising"]
	BarterAccepted bool           `gorm:"default:false"`
	ProfileViews   int            `gorm:"default:0"`
	Rating         float64        `gorm:"default:0"`
	IsPublic       bool           `gorm:"default:true"`

	// Relations
	PortfolioItems []PortfolioItem `gorm:"foreignKey:ModelID"`
	Reviews        []Review        `gorm:"foreignKey:ModelID"`
}

type EmployerProfile struct {
	BaseModel
	UserID        string `gorm:"uniqueIndex;not null"`
	CompanyName   string `gorm:"not null"`
	ContactPerson string
	Phone         string
	Website       string
	City          string
	CompanyType   string
	Description   string
	IsVerified    bool    `gorm:"default:false"`
	Rating        float64 `gorm:"default:0"`
}

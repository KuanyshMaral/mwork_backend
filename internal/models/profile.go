package models

import (
	"encoding/json"
	"gorm.io/datatypes"
)

type ModelProfile struct {
	BaseModel
	UserID         string `gorm:"uniqueIndex;not null"`
	Name           string `gorm:"not null"`
	Age            int    `gorm:"not null"`
	Height         int    `gorm:"not null"` // Изменено с float64 на int
	Weight         int    `gorm:"not null"` // Изменено с float64 на int
	Gender         string `gorm:"not null"`
	Experience     int    // years
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

// GetCategories возвращает категории модели как slice строк
func (m *ModelProfile) GetCategories() []string {
	var categories []string
	if len(m.Categories) > 0 {
		_ = json.Unmarshal(m.Categories, &categories)
	}
	return categories
}

// GetLanguages возвращает языки модели как slice строк
func (m *ModelProfile) GetLanguages() []string {
	var languages []string
	if len(m.Languages) > 0 {
		_ = json.Unmarshal(m.Languages, &languages)
	}
	return languages
}

// SetCategories устанавливает категории модели
func (m *ModelProfile) SetCategories(categories []string) {
	data, _ := json.Marshal(categories)
	m.Categories = datatypes.JSON(data)
}

// SetLanguages устанавливает языки модели
func (m *ModelProfile) SetLanguages(languages []string) {
	data, _ := json.Marshal(languages)
	m.Languages = datatypes.JSON(data)
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

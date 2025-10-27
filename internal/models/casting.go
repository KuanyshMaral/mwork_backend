package models

import (
	"encoding/json"
	"gorm.io/datatypes"
	"time"
)

type Casting struct {
	BaseModel
	EmployerID      string         `gorm:"not null;index" json:"employer_id"`
	Title           string         `gorm:"not null" json:"title"`
	Description     string         `json:"description,omitempty"`
	PaymentMin      float64        `json:"payment_min"`
	PaymentMax      float64        `json:"payment_max"`
	CastingDate     *time.Time     `json:"casting_date,omitempty"`
	CastingTime     *string        `json:"casting_time,omitempty"`
	Address         *string        `json:"address,omitempty"`
	City            string         `gorm:"not null" json:"city"`
	Categories      datatypes.JSON `gorm:"type:jsonb" json:"categories,omitempty"`
	Gender          string         `json:"gender,omitempty"`
	AgeMin          *int           `json:"age_min,omitempty"`
	AgeMax          *int           `json:"age_max,omitempty"`
	HeightMin       *float64       `json:"height_min,omitempty"`
	HeightMax       *float64       `json:"height_max,omitempty"`
	WeightMin       *float64       `json:"weight_min,omitempty"`
	WeightMax       *float64       `json:"weight_max,omitempty"`
	ClothingSize    *string        `json:"clothing_size,omitempty"`
	ShoeSize        *string        `json:"shoe_size,omitempty"`
	ExperienceLevel *string        `json:"experience_level,omitempty"`
	Languages       datatypes.JSON `gorm:"type:jsonb" json:"languages,omitempty"`
	JobType         string         `json:"job_type"` // "one_time", "permanent"
	Status          CastingStatus  `gorm:"default:'draft'" json:"status"`
	Views           int            `gorm:"default:0" json:"views"`

	// Relations
	Employer  EmployerProfile   `gorm:"foreignKey:EmployerID" json:"employer,omitempty"`
	Responses []CastingResponse `gorm:"foreignKey:CastingID" json:"responses,omitempty"`
}

// Методы для удобного получения категорий и языков
func (c *Casting) GetCategories() []string {
	var cats []string
	if len(c.Categories) > 0 {
		_ = json.Unmarshal(c.Categories, &cats)
	}
	return cats
}

func (c *Casting) GetLanguages() []string {
	var langs []string
	if len(c.Languages) > 0 {
		_ = json.Unmarshal(c.Languages, &langs)
	}
	return langs
}

// Методы для установки категорий и языков
func (c *Casting) SetCategories(categories []string) {
	data, _ := json.Marshal(categories)
	c.Categories = datatypes.JSON(data)
}

func (c *Casting) SetLanguages(languages []string) {
	data, _ := json.Marshal(languages)
	c.Languages = datatypes.JSON(data)
}

type CastingResponse struct {
	BaseModel
	CastingID string         `gorm:"not null;index" json:"casting_id"`
	ModelID   string         `gorm:"not null;index" json:"model_id"`
	Message   *string        `json:"message,omitempty"`
	Status    ResponseStatus `gorm:"default:'pending'" json:"status"`

	// Relations
	Model   ModelProfile `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	Casting Casting      `gorm:"foreignKey:CastingID" json:"casting,omitempty"`
}

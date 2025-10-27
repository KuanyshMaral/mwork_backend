package models

import (
	"gorm.io/gorm"
	"time"
)

type BaseModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type BaseModelWithDeleted struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

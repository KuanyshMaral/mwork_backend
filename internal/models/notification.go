package models

import (
	"gorm.io/datatypes"
	"time"
)

type Notification struct {
	BaseModel
	UserID  string `gorm:"not null;index"`
	Type    string `gorm:"not null"` // "new_response", "new_message", "casting_match"
	Title   string `gorm:"not null"`
	Message string
	Data    datatypes.JSON `gorm:"type:jsonb"` // {"casting_id": "...", "model_id": "..."}
	IsRead  bool           `gorm:"default:false"`
	ReadAt  *time.Time
}

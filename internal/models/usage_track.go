package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Константы для типов событий
const (
	EventUserLogin      = "USER_LOGIN"
	EventUserRegister   = "USER_REGISTER"
	EventCastingCreate  = "CASTING_CREATE"
	EventCastingView    = "CASTING_VIEW"
	EventResponseCreate = "RESPONSE_CREATE"
	EventProfileView    = "PROFILE_VIEW"
)

// UsageTrack представляет запись об одном событии пользователя
type UsageTrack struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primary_key"`
	UserID    *uuid.UUID     `gorm:"type:uuid;index"` // Может быть nil для анонимных событий
	EventType string         `gorm:"type:varchar(100);index;not null"`
	Metadata  datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP;index"`
}

// TableName указывает GORM имя таблицы
func (UsageTrack) TableName() string {
	return "usage_tracking"
}

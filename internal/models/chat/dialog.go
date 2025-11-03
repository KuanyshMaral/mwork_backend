package chat

import "time"

type Dialog struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	IsGroup       bool   `gorm:"default:false"`
	Title         *string
	ImageURL      *string
	CastingID     *string `gorm:"index"`
	LastMessageID *string `gorm:"index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Participants []DialogParticipant `gorm:"foreignKey:DialogID"`
	LastMessage  *Message            `gorm:"foreignKey:LastMessageID"`
}

// ✅ ИСПРАВЛЕНИЕ: Указываем схему "chat"
func (Dialog) TableName() string {
	return "chat.dialogs"
}

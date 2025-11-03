package chat

import "time"

type DialogParticipant struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DialogID    string `gorm:"index;not null"`
	UserID      string `gorm:"index;not null"`
	Role        string `gorm:"default:'member'"` // member, admin, owner
	JoinedAt    time.Time
	LastSeenAt  time.Time
	IsMuted     bool
	TypingUntil *time.Time
	LeftAt      *time.Time
}

// ✅ ИСПРАВЛЕНИЕ: Указываем схему "chat"
func (DialogParticipant) TableName() string {
	return "chat.dialog_participants"
}

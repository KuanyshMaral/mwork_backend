package chat

import "time"

type MessageReaction struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MessageID string `gorm:"index;not null"`
	UserID    string `gorm:"index;not null"`
	Emoji     string `gorm:"type:varchar(10);not null"`
	CreatedAt time.Time
}

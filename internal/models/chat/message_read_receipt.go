package chat

import "time"

type MessageReadReceipt struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MessageID string `gorm:"index;not null"`
	UserID    string `gorm:"index;not null"`
	ReadAt    time.Time
}

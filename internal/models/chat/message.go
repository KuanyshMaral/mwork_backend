package chat

import "time"

type Message struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DialogID       string `gorm:"index;not null"`
	SenderID       string `gorm:"index;not null"`
	Type           string `gorm:"default:'text'"` // text, image, video, file, system
	Content        string `gorm:"type:text"`
	AttachmentURL  *string
	AttachmentName *string
	ForwardFromID  *string `gorm:"index"`
	ReplyToID      *string `gorm:"index"`
	Status         string  `gorm:"default:'sent'"` // sent, delivered, read, deleted
	DeletedAt      *time.Time
	CreatedAt      time.Time

	ForwardFrom  *Message             `gorm:"foreignKey:ForwardFromID"`
	ReplyTo      *Message             `gorm:"foreignKey:ReplyToID"`
	Reactions    []MessageReaction    `gorm:"foreignKey:MessageID"`
	ReadReceipts []MessageReadReceipt `gorm:"foreignKey:MessageID"`
	Attachments  []MessageAttachment  `gorm:"foreignKey:MessageID"`
}

// ✅ ИСПРАВЛЕНИЕ: Указываем схему "chat"
func (Message) TableName() string {
	return "chat.messages"
}

package chat

import "time"

type MessageAttachment struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MessageID  string `gorm:"index;not null"`
	UploaderID string `gorm:"index"`
	FileType   string // image, video, file
	MimeType   string
	FileName   string
	URL        string
	Size       int64
	CreatedAt  time.Time
}

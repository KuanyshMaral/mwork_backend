package models

import "time"

type Upload struct {
	ID         string `gorm:"primaryKey"` // UUID или ULID
	UserID     string `gorm:"index"`      // кто загрузил
	EntityType string // "model_profile", "casting", "employer_profile", "message", etc.
	EntityID   string // ID сущности, к которой привязан файл
	FileType   string // "image", "video", "document"
	Usage      string // "avatar", "portfolio", "casting_attachment", "logo", etc.
	Path       string // "/uploads/models/abc.jpg"
	MimeType   string
	Size       int64
	IsPublic   bool
	CreatedAt  time.Time
}

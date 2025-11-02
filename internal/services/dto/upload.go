package dto

import (
	"mime/multipart"
	"time"
)

// ============================================
// REQUEST STRUCTURES
// ============================================

// ▼▼▼ ИЗМЕНЕНО (Проблема 6) ▼▼▼
// UniversalUploadRequest - универсальный запрос для загрузки
type UniversalUploadRequest struct {
	UserID     string                `json:"-"`                              // Из контекста
	Module     string                `form:"module" binding:"required"`      // portfolio, chat, casting, profile
	EntityType string                `form:"entity_type" binding:"required"` // model_profile, dialog, message, casting
	EntityID   string                `form:"entity_id"`                      // ID сущности (опционально)
	Usage      string                `form:"usage" binding:"required"`       // avatar, message_attachment, portfolio_photo
	IsPublic   bool                  `form:"is_public"`                      // Публичный доступ
	Metadata   map[string]string     `form:"metadata"`                       // Дополнительные метаданные
	File       *multipart.FileHeader `json:"-"`                              // Сам файл (не биндится из формы)
}

// ▲▲▲ ИЗМЕНЕНО (Проблема 6) ▲▲▲

// ============================================
// RESPONSE STRUCTURES
// ============================================

// UploadResponse - ответ с информацией о файле
type UploadResponse struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	Module     string            `json:"module"`
	EntityType string            `json:"entity_type"`
	EntityID   string            `json:"entity_id,omitempty"`
	FileType   string            `json:"file_type"` // image, video, document, file
	Usage      string            `json:"usage"`
	URL        string            `json:"url"`
	MimeType   string            `json:"mime_type"`
	Size       int64             `json:"size"`
	IsPublic   bool              `json:"is_public"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// StorageUsageResponse - информация об использовании хранилища
type StorageUsageResponse struct {
	Used       int64   `json:"used"`
	Limit      int64   `json:"limit"`
	Percentage float64 `json:"percentage"`
}

// UploadStats - статистика платформы
type UploadStats struct {
	TotalUploads int64            `json:"total_uploads"`
	TotalSize    int64            `json:"total_size"`
	ByModule     map[string]int64 `json:"by_module"`
	ByFileType   map[string]int64 `json:"by_file_type"`
	ActiveUsers  int64            `json:"active_users"`
	StorageUsed  int64            `json:"storage_used"`
	StorageLimit int64            `json:"storage_limit"`
}

// ============================================
// СТАРЫЕ DTO (ДЛЯ ОБРАТНОЙ СОВМЕСТИМОСТИ)
// ============================================

// UploadRequest - старый формат (deprecated, используйте UniversalUploadRequest)
type UploadRequest struct {
	EntityType string `json:"entity_type" binding:"required"`
	EntityID   string `json:"entity_id"`
	Usage      string `json:"usage" binding:"required"`
	IsPublic   bool   `json:"is_public"`
}

// FileConfigPortfolio - конфигурация для portfolio (deprecated)
type FileConfigPortfolio struct {
	MaxSize        int64
	AllowedTypes   []string
	AllowedUsages  map[string][]string
	StoragePath    string
	MaxUserStorage int64
}

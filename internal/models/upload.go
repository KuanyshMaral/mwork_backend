package models

import (
	"database/sql/driver" // <-- 1. ДОБАВИТЬ ИМПОРТ
	"encoding/json"       // <-- 2. ДОБАВИТЬ ИМПОРТ
	"errors"              // <-- 3. ДОБАВИТЬ ИМПОРТ
	"time"

	"gorm.io/gorm"
)

// ▼▼▼ НАЧАЛО: ДОБАВЬТЕ ЭТОТ БЛОК ▼▼▼

// JSONMap представляет тип map[string]interface{} для хранения в JSONB
type JSONMap map[string]interface{}

// Value реализует интерфейс driver.Valuer
// (Говорит GORM, как сохранить этот тип в БД)
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	// Преобразуем map в JSON []byte
	return json.Marshal(m)
}

// Scan реализует интерфейс sql.Scanner
// (Говорит GORM, как прочитать этот тип из БД)
func (m *JSONMap) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	if b == nil {
		*m = nil
		return nil
	}

	// Преобразуем JSON []byte обратно в map
	return json.Unmarshal(b, m)
}

// ▲▲▲ КОНЕЦ: ДОБАВЬТЕ ЭТОТ БЛОК ▲▲▲

// Upload представляет загруженный файл в системе
type Upload struct {
	ID     string `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID string `gorm:"type:uuid;not null;index:idx_uploads_user" json:"user_id"`

	// Новое поле для модуля
	Module string `gorm:"type:varchar(50);not null;index:idx_uploads_module" json:"module"`

	// Сущность, к которой привязан файл
	EntityType string `gorm:"type:varchar(50);not null;index:idx_uploads_entity" json:"entity_type"`
	EntityID   string `gorm:"type:uuid;index:idx_uploads_entity" json:"entity_id,omitempty"`

	// Информация о файле
	FileType string `gorm:"type:varchar(20);not null" json:"file_type"` // image, video, document, file
	Usage    string `gorm:"type:varchar(50);not null" json:"usage"`     // avatar, portfolio_photo, message_attachment
	Path     string `gorm:"type:text;not null" json:"path"`
	MimeType string `gorm:"type:varchar(100);not null" json:"mime_type"`
	Size     int64  `gorm:"not null" json:"size"`

	// Настройки доступа
	IsPublic bool `gorm:"default:false" json:"is_public"`

	// Дополнительные метаданные (JSON)
	Metadata JSONMap `gorm:"type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName указывает имя таблицы для GORM
func (Upload) TableName() string {
	return "uploads"
}

// Indexes для оптимизации запросов
func (Upload) Indexes() []string {
	return []string{
		"idx_uploads_user",
		"idx_uploads_module",
		"idx_uploads_entity",
		"idx_uploads_created_at",
	}
}

// BeforeCreate hook для валидации перед созданием
func (u *Upload) BeforeCreate(tx *gorm.DB) error {
	// Можно добавить валидацию
	return nil
}

// GetURL возвращает URL файла (метод-хелпер)
func (u *Upload) GetURL(baseURL string) string {
	if u.IsPublic {
		return baseURL + "/" + u.Path
	}
	return baseURL + "/api/v1/files/" + u.ID
}

// IsImage проверяет, является ли файл изображением
func (u *Upload) IsImage() bool {
	return u.FileType == "image"
}

// IsVideo проверяет, является ли файл видео
func (u *Upload) IsVideo() bool {
	return u.FileType == "video"
}

// IsDocument проверяет, является ли файл документом
func (u *Upload) IsDocument() bool {
	return u.FileType == "document"
}

package dto

import "time"

// ---------------- Requests ----------------

type CreateNotificationRequest struct {
	UserID  string                 `json:"user_id" validate:"-"` // Set by server
	Type    string                 `json:"type" validate:"required"`
	Title   string                 `json:"title" validate:"required,max=100"`
	Message string                 `json:"message" validate:"omitempty,max=1000"`
	Data    map[string]interface{} `json:"data"`
}

type CreateBulkNotificationsRequest struct {
	// 'dive' validates each element in the slice
	Notifications []*CreateNotificationRequest `json:"notifications" validate:"required,min=1,dive"`
}

type CreateTemplateRequest struct {
	Type      string   `json:"type" validate:"required"`
	Title     string   `json:"title" validate:"required,max=100"`
	Message   string   `json:"message" validate:"required,max=2000"`
	Variables []string `json:"variables"`
	IsActive  bool     `json:"is_active"`
}

type UpdateTemplateRequest struct {
	Title     *string  `json:"title,omitempty" validate:"omitempty,max=100"`
	Message   *string  `json:"message,omitempty" validate:"omitempty,max=2000"`
	Variables []string `json:"variables,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
}

type SendBulkNotificationRequest struct {
	UserIDs []string               `json:"user_ids" validate:"required,min=1"`
	Type    string                 `json:"type" validate:"required"`
	Title   string                 `json:"title" validate:"required,max=100"`
	Message string                 `json:"message" validate:"required,max=2100"`
	Data    map[string]interface{} `json:"data"`
}

// ---------------- Responses ----------------

type NotificationResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	IsRead    bool                   `json:"is_read"`
	ReadAt    *time.Time             `json:"read_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type NotificationListResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
	Total         int64                   `json:"total"`
	Page          int                     `json:"page"`
	PageSize      int                     `json:"page_size"`
	TotalPages    int                     `json:"total_pages"`
}

// ---------------- Criteria ----------------

// Для поиска уведомлений пользователя
type NotificationCriteria struct {
	Page     int
	PageSize int
	Filters  map[string]interface{}
}

// Для поиска уведомлений админа
type AdminNotificationCriteria struct {
	Page     int
	PageSize int
	Filters  map[string]interface{}
}

// Данные для batch-уведомлений
type ResponseNotificationData struct {
	CastingID  string
	ResponseID string
	ModelName  string
}

type CastingMatchNotificationData struct {
	ModelID      string
	CastingTitle string
	Score        float64
}

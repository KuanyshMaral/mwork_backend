package dto

import "time"

// ---------------- Requests ----------------

type CreateNotificationRequest struct {
	UserID  string                 `json:"user_id" binding:"required"`
	Type    string                 `json:"type" binding:"required"`
	Title   string                 `json:"title" binding:"required"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type CreateBulkNotificationsRequest struct {
	Notifications []*CreateNotificationRequest `json:"notifications" binding:"required"`
}

type CreateTemplateRequest struct {
	Type      string   `json:"type" binding:"required"`
	Title     string   `json:"title" binding:"required"`
	Message   string   `json:"message" binding:"required"`
	Variables []string `json:"variables"`
	IsActive  bool     `json:"is_active"`
}

type UpdateTemplateRequest struct {
	Title     *string  `json:"title,omitempty"`
	Message   *string  `json:"message,omitempty"`
	Variables []string `json:"variables,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
}

type SendBulkNotificationRequest struct {
	UserIDs []string               `json:"user_ids" binding:"required"`
	Type    string                 `json:"type" binding:"required"`
	Title   string                 `json:"title" binding:"required"`
	Message string                 `json:"message" binding:"required"`
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

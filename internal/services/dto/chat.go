package dto

import (
	"time"

	"mwork_backend/internal/repositories"
)

// Request/Response structures

type CreateDialogRequest struct {
	IsGroup   bool     `json:"is_group"`
	Title     *string  `json:"title,omitempty"`
	ImageURL  *string  `json:"image_url,omitempty"`
	CastingID *string  `json:"casting_id,omitempty"`
	UserIDs   []string `json:"user_ids" binding:"required,min=1"`
}

type UpdateDialogRequest struct {
	Title    *string `json:"title,omitempty"`
	ImageURL *string `json:"image_url,omitempty"`
}

type SendMessageRequest struct {
	DialogID      string  `json:"dialog_id" binding:"required"`
	Type          string  `json:"type" binding:"required"` // text, image, video, file, system
	Content       string  `json:"content"`
	ReplyToID     *string `json:"reply_to_id,omitempty"`
	ForwardFromID *string `json:"forward_from_id,omitempty"`
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type ForwardMessageRequest struct {
	MessageID string   `json:"message_id" binding:"required"`
	DialogIDs []string `json:"dialog_ids" binding:"required,min=1"`
}

type DialogResponse struct {
	ID           string                 `json:"id"`
	IsGroup      bool                   `json:"is_group"`
	Title        *string                `json:"title,omitempty"`
	ImageURL     *string                `json:"image_url,omitempty"`
	CastingID    *string                `json:"casting_id,omitempty"`
	LastMessage  *MessageResponse       `json:"last_message,omitempty"`
	Participants []*ParticipantResponse `json:"participants"`
	UnreadCount  int64                  `json:"unread_count"`
	IsMuted      bool                   `json:"is_muted"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type MessageResponse struct {
	ID             string                `json:"id"`
	DialogID       string                `json:"dialog_id"`
	SenderID       string                `json:"sender_id"`
	SenderName     string                `json:"sender_name,omitempty"`
	Type           string                `json:"type"`
	Content        string                `json:"content"`
	AttachmentURL  *string               `json:"attachment_url,omitempty"`
	AttachmentName *string               `json:"attachment_name,omitempty"`
	ReplyTo        *MessageResponse      `json:"reply_to,omitempty"`
	ForwardFrom    *MessageResponse      `json:"forward_from,omitempty"`
	Status         string                `json:"status"`
	Reactions      []*ReactionResponse   `json:"reactions"`
	Attachments    []*AttachmentResponse `json:"attachments"`
	ReadBy         []string              `json:"read_by"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	Message        string                `json:"message"`
}

type ParticipantResponse struct {
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	Role       string    `json:"role"`
	LastSeenAt time.Time `json:"last_seen_at"`
	IsMuted    bool      `json:"is_muted"`
	IsOnline   bool      `json:"is_online"`
}

type AttachmentResponse struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	FileType  string    `json:"file_type"`
	MimeType  string    `json:"mime_type"`
	FileName  string    `json:"file_name"`
	URL       string    `json:"url"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type ReactionResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type ReadReceiptResponse struct {
	UserID   string    `json:"user_id"`
	UserName string    `json:"user_name"`
	ReadAt   time.Time `json:"read_at"`
}

type MessageListResponse struct {
	Messages   []*MessageResponse `json:"messages"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
	HasMore    bool               `json:"has_more"`
}

type CriteriaPage struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"offset"`
}

type DialogListResponse struct {
	Dialogs    []*DialogResponse `json:"dialogs"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

type DialogWithMessagesResponse struct {
	Dialog   *DialogResponse    `json:"dialog"`
	Messages []*MessageResponse `json:"messages"`
	Total    int64              `json:"total"`
	HasMore  bool               `json:"has_more"`
}

// Re-export repository types
type MessageCriteria = repositories.MessageCriteria
type DialogCriteria = repositories.DialogCriteria

// File configuration
type FileConfig struct {
	MaxSize      int64
	AllowedTypes []string
	StoragePath  string
}

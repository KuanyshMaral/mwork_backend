package models

import "time"

type CastingResponse struct {
	ID        string
	CastingID string
	ModelID   string
	Message   *string
	Status    string // "pending" | "accepted" | "rejected"
	CreatedAt time.Time
}

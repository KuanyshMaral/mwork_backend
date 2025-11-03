package models

type PortfolioItem struct {
	BaseModel
	ModelID string `gorm:"not null;index" json:"model_id"`
	// ❗️ ИСПРАВЛЕНО: UploadID должен быть *string или NullUUID
	UploadID    *string `json:"upload_id,omitempty"` // <-- ИСПОЛЬЗУЙ *string
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	OrderIndex  int     `json:"order_index"`

	// Relations
	Upload *Upload      `gorm:"foreignKey:UploadID" json:"upload"`
	Model  ModelProfile `gorm:"foreignKey:ModelID" json:"model"`
}

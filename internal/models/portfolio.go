package models

type PortfolioItem struct {
	BaseModel
	ModelID     string `gorm:"not null;index"`
	UploadID    string `gorm:"not null;index"`
	Title       string
	Description string
	OrderIndex  int `gorm:"default:0"`

	// Relations
	Upload *Upload `gorm:"foreignKey:UploadID"`
}

type Upload struct {
	BaseModel
	UserID     string `gorm:"not null;index"`
	EntityType string // "model_profile", "portfolio", "casting"
	EntityID   string
	FileType   string // "image", "video", "document"
	Usage      string // "avatar", "portfolio_photo", "casting_attachment"
	Path       string `gorm:"not null"`
	MimeType   string
	Size       int64
	IsPublic   bool `gorm:"default:true"`
}

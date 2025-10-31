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

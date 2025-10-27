package config

import "mwork_backend/internal/services/dto"

var PortfolioFileConfig = dto.FileConfigPortfolio{
	MaxSize:      50 * 1024 * 1024, // 50MB
	AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "video/mp4", "application/pdf"},
	AllowedUsages: map[string][]string{
		"model_profile": {"avatar", "portfolio_photo"},
		"portfolio":     {"portfolio_photo", "portfolio_video"},
		"casting":       {"casting_attachment"},
	},
	StoragePath:    "./uploads",
	MaxUserStorage: 100 * 1024 * 1024, // 100MB per user
}

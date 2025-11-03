package services

import (
	"mwork_backend/internal/email"
	"mwork_backend/internal/storage"
)

// ServiceContainer содержит все сервисы приложения.
type ServiceContainer struct {
	UserService         UserService
	AuthService         AuthService
	ProfileService      ProfileService
	CastingService      CastingService
	ResponseService     ResponseService
	ReviewService       ReviewService
	PortfolioService    PortfolioService
	MatchingService     MatchingService
	NotificationService NotificationService
	SubscriptionService SubscriptionService
	SearchService       SearchService
	AnalyticsService    AnalyticsService
	ChatService         ChatService
	UploadService       UploadService
	EmailService        email.Provider
	storage             storage.Storage // (Можно сделать приватным, если он нужен только внутри других сервисов)
}

package handlers

// AppHandlers содержит все хэндлеры приложения.
type AppHandlers struct {
	AuthHandler         *AuthHandler
	UserHandler         *UserHandler
	ProfileHandler      *ProfileHandler
	CastingHandler      *CastingHandler
	ResponseHandler     *ResponseHandler
	ReviewHandler       *ReviewHandler
	PortfolioHandler    *PortfolioHandler
	MatchingHandler     *MatchingHandler
	NotificationHandler *NotificationHandler
	SubscriptionHandler *SubscriptionHandler
	SearchHandler       *SearchHandler
	AnalyticsHandler    *AnalyticsHandler
	ChatHandler         *ChatHandler
	FileHandler         *FileHandler
	UploadHandler       *UploadHandler
}

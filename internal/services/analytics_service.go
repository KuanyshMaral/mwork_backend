package services

import (
	"context" // <-- Добавлено
	"errors"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

var (
	ErrInvalidDateRange = errors.New("invalid date range")
)

type AnalyticsService interface {
	GetPlatformOverview(dateFrom, dateTo time.Time) (*dto.PlatformOverview, error)
	GetPlatformGrowthMetrics(days int) (*dto.GrowthMetrics, error)
	GetUserAnalytics(dateFrom, dateTo time.Time) (*dto.UserAnalytics, error)
	GetUserAcquisitionMetrics(dateFrom, dateTo time.Time) (*dto.UserAcquisitionMetrics, error)
	GetCastingAnalytics(dateFrom, dateTo time.Time) (*dto.CastingAnalytics, error)
	GetMatchingAnalytics(dateFrom, dateTo time.Time) (*dto.MatchingAnalytics, error)
	GetFinancialAnalytics(dateFrom, dateTo time.Time) (*dto.FinancialAnalytics, error)
	GetGeographicAnalytics() (*dto.GeographicAnalytics, error)
	GetPerformanceMetrics(dateFrom, dateTo time.Time) (*dto.PerformanceMetrics, error)
	GetRealTimeMetrics() (*dto.RealTimeMetrics, error)
	GetActiveUsersCount() (int64, error)
	GetAdminDashboard(adminID string) (*dto.AdminDashboard, error)
	GetUserRetentionMetrics(days int) (*dto.UserRetentionMetrics, error)
	GetCastingPerformanceMetrics(employerID string, dateFrom, dateTo time.Time) (*dto.CastingPerformanceMetrics, error)
	GetMatchingEfficiencyMetrics(days int) (*dto.MatchingEfficiencyMetrics, error)
	GetCityPerformanceMetrics(topN int) ([]*dto.CityPerformance, error)
	GetCategoryAnalytics() (*dto.CategoryAnalytics, error)
	GetPopularCategories(days int, limit int) ([]*dto.CategoryStats, error)
	GetPlatformHealthMetrics() (*dto.PlatformHealthMetrics, error)
	GenerateCustomReport(req *dto.CustomReportRequest) (*dto.CustomReport, error)
	GetPredefinedReports() ([]*dto.PredefinedReport, error)
	GetSystemHealthMetrics() (*dto.SystemHealth, error)
}

type analyticsService struct {
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	castingRepo      repositories.CastingRepository
	reviewRepo       repositories.ReviewRepository
	notificationRepo repositories.NotificationRepository
	portfolioRepo    repositories.PortfolioRepository
	subscriptionRepo repositories.SubscriptionRepository
	chatRepo         repositories.ChatRepository
	analyticsRepo    repositories.AnalyticsRepository // <-- Добавлено
}

func NewAnalyticsService(
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	castingRepo repositories.CastingRepository,
	reviewRepo repositories.ReviewRepository,
	notificationRepo repositories.NotificationRepository,
	portfolioRepo repositories.PortfolioRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	chatRepo repositories.ChatRepository,
	analyticsRepo repositories.AnalyticsRepository, // <-- Добавлено
) AnalyticsService {
	return &analyticsService{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		castingRepo:      castingRepo,
		reviewRepo:       reviewRepo,
		notificationRepo: notificationRepo,
		portfolioRepo:    portfolioRepo,
		subscriptionRepo: subscriptionRepo,
		chatRepo:         chatRepo,
		analyticsRepo:    analyticsRepo, // <-- Добавлено
	}
}

// Platform Overview
func (s *analyticsService) GetPlatformOverview(dateFrom, dateTo time.Time) (*dto.PlatformOverview, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	growthMetrics, err := s.GetPlatformGrowthMetrics(30)
	if err != nil {
		return nil, err
	}

	userAnalytics, err := s.GetUserAnalytics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	castingAnalytics, err := s.GetCastingAnalytics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	matchingAnalytics, err := s.GetMatchingAnalytics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	financialAnalytics, err := s.GetFinancialAnalytics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	geographicAnalytics, err := s.GetGeographicAnalytics()
	if err != nil {
		return nil, err
	}

	categoryAnalytics, err := s.GetCategoryAnalytics()
	if err != nil {
		return nil, err
	}

	performanceMetrics, err := s.GetPerformanceMetrics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	platformHealth, err := s.GetPlatformHealthMetrics()
	if err != nil {
		return nil, err
	}

	realTimeMetrics, err := s.GetRealTimeMetrics()
	if err != nil {
		return nil, err
	}

	return &dto.PlatformOverview{
		GrowthMetrics:       *growthMetrics,
		UserAnalytics:       *userAnalytics,
		CastingAnalytics:    *castingAnalytics,
		MatchingAnalytics:   *matchingAnalytics,
		FinancialAnalytics:  *financialAnalytics,
		GeographicAnalytics: *geographicAnalytics,
		CategoryAnalytics:   *categoryAnalytics,
		PerformanceMetrics:  *performanceMetrics,
		PlatformHealth:      *platformHealth,
		RealTimeMetrics:     *realTimeMetrics,
		Reports:             []dto.CustomReport{},
	}, nil
}

func (s *analyticsService) GetPlatformGrowthMetrics(days int) (*dto.GrowthMetrics, error) {
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -days)

	userStats, err := s.userRepo.GetUserStats(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	// Calculate monthly growth
	monthStart := time.Now().AddDate(0, -1, 0)
	monthStats, _ := s.userRepo.GetUserStats(monthStart, dateTo)

	previousMonthStart := monthStart.AddDate(0, -1, 0)
	previousMonthStats, _ := s.userRepo.GetUserStats(previousMonthStart, monthStart)

	monthlyGrowthRate := 0.0
	if previousMonthStats.TotalUsers > 0 {
		monthlyGrowthRate = (float64(monthStats.NewUsers) / float64(previousMonthStats.TotalUsers)) * 100
	}

	// Generate historical trends
	var historicalTrends []dto.DataPoint
	ctx := context.Background() // <-- Используем context
	for i := 0; i < days; i++ {
		currentDate := dateFrom.AddDate(0, 0, i)

		// Используем GetDAU из analyticsRepo для получения активных пользователей
		// или GetUserStats для новых пользователей. Выберем GetUserStats для "NewUsers"
		dailyStats, err := s.userRepo.GetUserStats(currentDate, currentDate.AddDate(0, 0, 1))
		if err != nil {
			// Попробуем получить DAU, если GetUserStats не удался (или наоборот)
			dau, dauErr := s.analyticsRepo.GetDAU(ctx, currentDate)
			if dauErr != nil {
				continue // Пропускаем день, если обе статистики не удались
			}
			dailyStats.NewUsers = dau // Используем DAU как запасной вариант, хотя это разные метрики
			// В идеале GetUserStats должен быть надежным, или мы должны
			// отслеживать "USER_REGISTER" в analyticsRepo
		}

		historicalTrends = append(historicalTrends, dto.DataPoint{
			Timestamp: currentDate.Format("2006-01-02"),
			Value:     float64(dailyStats.NewUsers),
		})
	}

	// Получаем средний DAU за период
	avgDAU, _ := s.calculateAverageDAU(dateFrom, dateTo)

	return &dto.GrowthMetrics{
		TotalUsers:        int(userStats.TotalUsers),
		NewUsersThisMonth: int(monthStats.NewUsers),
		MonthlyGrowthRate: monthlyGrowthRate,
		ActiveUsers:       int(avgDAU), // <-- Используем средний DAU
		ChurnRate:         s.calculateChurnRate(dateFrom, dateTo),
		HistoricalTrends:  historicalTrends,
	}, nil
}

// User Analytics
func (s *analyticsService) GetUserAnalytics(dateFrom, dateTo time.Time) (*dto.UserAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	acquisition, err := s.GetUserAcquisitionMetrics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	retention, err := s.GetUserRetentionMetrics(30)
	if err != nil {
		return nil, err
	}

	// --- Обновленная логика ---
	// Рассчитываем средний DAU, WAU, MAU за период
	avgDAU, err := s.calculateAverageDAU(dateFrom, dateTo)
	if err != nil {
		// Не фатально, можем просто установить в 0
		avgDAU = 0
	}

	// WAU и MAU все еще упрощены, но основаны на более точной метрике DAU
	activity := dto.UserActivity{
		DailyActiveUsers:   int(avgDAU),
		WeeklyActiveUsers:  int(avgDAU * 5),  // Упрощенное предположение (5 рабочих дней)
		MonthlyActiveUsers: int(avgDAU * 22), // Упрощенное предположение (22 рабочих дня)
	}
	// --- Конец обновленной логики ---

	churn := dto.ChurnAnalysis{
		Rate: s.calculateChurnRate(dateFrom, dateTo),
		Reasons: []dto.DataPoint{
			{Timestamp: "Inactivity", Value: 45.0},
			{Timestamp: "Price", Value: 25.0},
			{Timestamp: "Competition", Value: 20.0},
			{Timestamp: "Other", Value: 10.0},
		},
	}

	return &dto.UserAnalytics{
		Acquisition: *acquisition,
		Retention:   *retention,
		Activity:    activity,
		Churn:       churn,
	}, nil
}

func (s *analyticsService) GetUserAcquisitionMetrics(dateFrom, dateTo time.Time) (*dto.UserAcquisitionMetrics, error) {
	userStats, err := s.userRepo.GetUserStats(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	// Получаем DAU, чтобы рассчитать ReturningUsers
	avgDAU, _ := s.calculateAverageDAU(dateFrom, dateTo)

	sources := []dto.DataPoint{
		{Timestamp: "Organic", Value: 500},
		{Timestamp: "Direct", Value: 300},
		{Timestamp: "Referral", Value: 150},
		{Timestamp: "Social", Value: 200},
		{Timestamp: "Paid Ads", Value: 100},
	}

	returningUsers := int(avgDAU) - int(userStats.NewUsers)
	if returningUsers < 0 {
		returningUsers = 0 // Не может быть отрицательным
	}

	return &dto.UserAcquisitionMetrics{
		NewUsers:       int(userStats.NewUsers),
		ReturningUsers: returningUsers, // <-- Используем DAU
		ConversionRate: 0.05,
		Sources:        sources,
	}, nil
}

// Casting Analytics
func (s *analyticsService) GetCastingAnalytics(dateFrom, dateTo time.Time) (*dto.CastingAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	castingStats, err := s.castingRepo.GetPlatformCastingStats(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	popularCategories, err := s.GetPopularCategories(30, 10)
	if err != nil {
		return nil, err
	}

	performance := dto.CastingPerformanceMetrics{
		CompletionRate: castingStats.SuccessRate,
		EngagementRate: castingStats.AvgResponseRate,
	}

	return &dto.CastingAnalytics{
		TotalCastings:       int(castingStats.TotalCastings),
		ActiveCastings:      int(castingStats.ActiveCastings),
		SuccessRate:         castingStats.SuccessRate,
		AverageResponseTime: castingStats.AvgResponseTime,
		Performance:         performance,
		Categories:          popularCategories,
	}, nil
}

// Matching Analytics
func (s *analyticsService) GetMatchingAnalytics(dateFrom, dateTo time.Time) (*dto.MatchingAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	matchingStats, err := s.castingRepo.GetMatchingStats(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	matchQuality := dto.MatchQuality{
		AverageScore:     matchingStats.AvgMatchScore,
		UserSatisfaction: matchingStats.AvgSatisfaction,
	}

	algorithm := dto.AlgorithmPerformance{
		Accuracy:  matchingStats.MatchRate,
		LatencyMS: 120,
		Version:   "v2.1.0",
	}

	efficiency := dto.MatchingEfficiencyMetrics{
		AverageMatchTime: matchingStats.TimeToMatch,
		MatchRate:        matchingStats.MatchRate,
		DropOffRate:      1.0 - matchingStats.ResponseRate,
	}

	return &dto.MatchingAnalytics{
		TotalMatches: int(matchingStats.TotalMatches),
		MatchQuality: matchQuality,
		Algorithm:    algorithm,
		Efficiency:   efficiency,
	}, nil
}

// Financial Analytics
func (s *analyticsService) GetFinancialAnalytics(dateFrom, dateTo time.Time) (*dto.FinancialAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	subscriptionMetrics, err := s.subscriptionRepo.GetSubscriptionMetrics(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	// Generate monthly revenue data
	var monthlyRevenue []dto.DataPoint
	for i := 0; i < 12; i++ {
		month := dateTo.AddDate(0, -i, 0)
		monthlyRevenue = append(monthlyRevenue, dto.DataPoint{
			Timestamp: month.Format("2006-01"),
			Value:     subscriptionMetrics.MRR * float64(12-i) / 12.0,
		})
	}

	subscriptionData := dto.SubscriptionMetrics{
		ActiveSubscriptions: int(subscriptionMetrics.TotalSubscribers),
		Cancellations:       int(float64(subscriptionMetrics.TotalSubscribers) * subscriptionMetrics.ChurnRate / 100.0),
		RevenueShare:        subscriptionMetrics.MRR / subscriptionMetrics.TotalRevenue,
	}

	return &dto.FinancialAnalytics{
		TotalRevenue:     subscriptionMetrics.TotalRevenue,
		MonthlyRevenue:   monthlyRevenue,
		ARPU:             subscriptionMetrics.ARPU,
		SubscriptionData: subscriptionData,
	}, nil
}

// Geographic Analytics
func (s *analyticsService) GetGeographicAnalytics() (*dto.GeographicAnalytics, error) {
	userDistribution, err := s.userRepo.GetUserDistributionByCity()
	if err != nil {
		return nil, err
	}

	castingDistribution, err := s.castingRepo.GetCastingDistributionByCity()
	if err != nil {
		return nil, err
	}

	var countries []dto.CityStats
	var cities []dto.CityPerformance

	for city, userCount := range userDistribution {
		countries = append(countries, dto.CityStats{
			Name:      city,
			UserCount: int(userCount),
			Revenue:   float64(userCount) * 25.0, // Simplified calculation
		})

		// Ensure userCount is not zero to avoid division by zero
		responseTime := 0.5
		if userCount > 0 {
			responseTime += (float64(castingDistribution[city]) / float64(userCount))
		}

		cities = append(cities, dto.CityPerformance{
			Name:           city,
			EngagementRate: 0.65 + (float64(len(city)%100) / 100.0 * 0.3),
			ResponseTime:   responseTime,
		})
	}

	return &dto.GeographicAnalytics{
		Countries: countries,
		Cities:    cities,
	}, nil
}

// Performance Analytics
func (s *analyticsService) GetPerformanceMetrics(dateFrom, dateTo time.Time) (*dto.PerformanceMetrics, error) {
	return &dto.PerformanceMetrics{
		ResponseTimes: dto.ResponseTimes{
			AverageMS: 150.0,
			MaxMS:     1200.0,
		},
		Throughput: dto.Throughput{
			RequestsPerSecond: 150.5,
		},
	}, nil
}

// Real-time Analytics
func (s *analyticsService) GetRealTimeMetrics() (*dto.RealTimeMetrics, error) {
	// Эта метрика (активные за 15 мин) хороша для "Real-time"
	activeUsers, err := s.userRepo.GetActiveUsersCount(15)
	if err != nil {
		return nil, err
	}

	activeCastings, err := s.castingRepo.GetActiveCastingsCount()
	if err != nil {
		return nil, err
	}

	keyMetrics := dto.KeyMetrics{
		ActiveUsers: int(activeUsers), // Активные за 15 мин
		NewUsers:    0,
		Revenue:     0.0,
	}

	// --- Обновленная логика ---
	// Получаем DAU (активные за *сегодня*)
	ctx := context.Background()
	dau, err := s.analyticsRepo.GetDAU(ctx, time.Now())
	if err != nil {
		dau = 0 // Не фатально
	}

	activity := dto.UserActivity{
		DailyActiveUsers:   int(dau),      // <-- Реальный DAU за сегодня
		WeeklyActiveUsers:  int(dau * 5),  // Упрощение
		MonthlyActiveUsers: int(dau * 20), // Упрощение
	}
	// --- Конец обновленной логики ---

	castingActivity := dto.CastingActivity{
		OpenCastings: int(activeCastings),
		NewCastings:  0,
	}

	matchActivity := dto.MatchActivity{
		ActiveMatches: 0,
		NewMatches:    0,
	}

	return &dto.RealTimeMetrics{
		KeyMetrics:      keyMetrics,
		UserActivity:    activity,
		CastingActivity: castingActivity,
		MatchActivity:   matchActivity,
		SystemEvents:    []dto.SystemEvent{},
	}, nil
}

func (s *analyticsService) GetActiveUsersCount() (int64, error) {
	return s.userRepo.GetActiveUsersCount(15)
}

// Admin Dashboard
func (s *analyticsService) GetAdminDashboard(adminID string) (*dto.AdminDashboard, error) {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return nil, err
	}

	if admin.Role != models.UserRoleAdmin {
		return nil, errors.New("insufficient permissions")
	}

	recentActivity := []dto.RecentActivity{
		{
			Timestamp: time.Now().Add(-time.Hour).Format(time.RFC3339),
			UserID:    "user1",
			Action:    "registration",
		},
	}

	systemHealth := dto.SystemHealth{
		CPUUsage: 45.2,
		RAMUsage: 67.8,
	}

	alerts := []dto.Alert{
		{
			Severity: "warning",
			Message:  "High database load detected",
		},
	}

	return &dto.AdminDashboard{
		RecentActivity: recentActivity,
		SystemHealth:   systemHealth,
		Alerts:         alerts,
	}, nil
}

// ==============================
// Helper methods
// ==============================

// calculateAverageDAU рассчитывает средний DAU за период
func (s *analyticsService) calculateAverageDAU(from, to time.Time) (float64, error) {
	ctx := context.Background()
	totalDAU := int64(0)
	numDays := 0

	// Нормализуем даты до начала дня
	currentDay := from.Truncate(24 * time.Hour)
	endDate := to.Truncate(24 * time.Hour)

	// Итерируем по каждому дню в диапазоне
	for ; !currentDay.After(endDate); currentDay = currentDay.AddDate(0, 0, 1) {
		dau, err := s.analyticsRepo.GetDAU(ctx, currentDay)
		if err != nil {
			// Если один день не удался, мы можем либо пропустить его, либо вернуть ошибку
			// Вернем ошибку, чтобы среднее значение было точным
			return 0, err
		}
		totalDAU += dau
		numDays++
	}

	if numDays == 0 {
		return 0, nil // Избегаем деления на ноль
	}

	return float64(totalDAU) / float64(numDays), nil
}

func (s *analyticsService) calculateRetentionRate(dateFrom, dateTo time.Time) float64 {
	// TODO: Реализовать с использованием analyticsRepo (например, когортный анализ)
	return 0.75
}

func (s *analyticsService) calculateChurnRate(dateFrom, dateTo time.Time) float64 {
	// TODO: Реализовать с использованием analyticsRepo
	return 0.08
}

// ==============================
// Stub implementations
// ==============================

func (s *analyticsService) GetUserRetentionMetrics(days int) (*dto.UserRetentionMetrics, error) {
	return &dto.UserRetentionMetrics{
		RetentionRate:   0.75,
		AverageLifespan: 365.0,
		Cohorts: []dto.CohortData{
			{CohortName: "2024-Q1", Retention: 0.80},
			{CohortName: "2024-Q2", Retention: 0.75},
			{CohortName: "2024-Q3", Retention: 0.70},
		},
	}, nil
}

func (s *analyticsService) GetCastingPerformanceMetrics(employerID string, dateFrom, dateTo time.Time) (*dto.CastingPerformanceMetrics, error) {
	return &dto.CastingPerformanceMetrics{
		CompletionRate: 0.85,
		EngagementRate: 0.72,
	}, nil
}

func (s *analyticsService) GetMatchingEfficiencyMetrics(days int) (*dto.MatchingEfficiencyMetrics, error) {
	return &dto.MatchingEfficiencyMetrics{
		AverageMatchTime: 2.5,
		MatchRate:        0.65,
		DropOffRate:      0.15,
	}, nil
}

func (s *analyticsService) GetCityPerformanceMetrics(topN int) ([]*dto.CityPerformance, error) {
	return []*dto.CityPerformance{
		{Name: "Almaty", EngagementRate: 0.75, ResponseTime: 1.2},
		{Name: "Astana", EngagementRate: 0.70, ResponseTime: 1.5},
	}, nil
}

func (s *analyticsService) GetCategoryAnalytics() (*dto.CategoryAnalytics, error) {
	categories, err := s.GetPopularCategories(30, 10)
	if err != nil {
		return nil, err
	}

	return &dto.CategoryAnalytics{
		Categories: categories,
	}, nil
}

func (s *analyticsService) GetPopularCategories(days int, limit int) ([]*dto.CategoryStats, error) {
	popularCategories, err := s.castingRepo.GetPopularCategories(limit)
	if err != nil {
		return nil, err
	}

	var result []*dto.CategoryStats
	for _, cat := range popularCategories {
		result = append(result, &dto.CategoryStats{
			Name:   cat.Name,
			Count:  int(cat.Count),
			Rating: 4.5,
		})
	}

	return result, nil
}

func (s *analyticsService) GetPlatformHealthMetrics() (*dto.PlatformHealthMetrics, error) {
	return &dto.PlatformHealthMetrics{
		SystemHealth: dto.SystemHealth{},
		ResourceUsage: dto.ResourceUsage{
			CPUUsage: 45.2,
			RAMUsage: 67.8,
		},
		DatabaseHealth: dto.DatabaseHealth{
			Status:       "healthy",
			QueryLatency: 0.08,
		},
		APIPerformance: dto.APIPerformance{
			AverageLatency: 0.15,
			ErrorRate:      0.02,
		},
		StorageUsage: dto.StorageUsage{
			UsedGB:  250.5,
			TotalGB: 1000.0,
		},
	}, nil
}

func (s *analyticsService) GenerateCustomReport(req *dto.CustomReportRequest) (*dto.CustomReport, error) {
	return &dto.CustomReport{
		ID:          "report-" + time.Now().Format("20060102150405"),
		Name:        req.Type,
		Description: "Custom report for " + req.Period,
		Data:        map[string]any{},
	}, nil
}

func (s *analyticsService) GetPredefinedReports() ([]*dto.PredefinedReport, error) {
	return []*dto.PredefinedReport{
		{Name: "User Growth Report", Description: "Monthly user growth analysis", Category: "Users"},
		{Name: "Revenue Report", Description: "Financial performance overview", Category: "Finance"},
		{Name: "Casting Performance", Description: "Casting success metrics", Category: "Castings"},
	}, nil
}

func (s *analyticsService) GetSystemHealthMetrics() (*dto.SystemHealth, error) {
	return &dto.SystemHealth{}, nil
}

package services

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

var (
	ErrInvalidDateRange = errors.New("invalid date range")
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type AnalyticsService interface {
	GetPlatformOverview(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.PlatformOverview, error)
	GetPlatformGrowthMetrics(db *gorm.DB, ctx context.Context, days int) (*dto.GrowthMetrics, error)
	GetUserAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.UserAnalytics, error)
	GetUserAcquisitionMetrics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.UserAcquisitionMetrics, error)
	GetCastingAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.CastingAnalytics, error)
	GetMatchingAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.MatchingAnalytics, error)
	GetFinancialAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.FinancialAnalytics, error)
	GetGeographicAnalytics(db *gorm.DB, ctx context.Context) (*dto.GeographicAnalytics, error)
	GetPerformanceMetrics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.PerformanceMetrics, error)
	GetRealTimeMetrics(db *gorm.DB, ctx context.Context) (*dto.RealTimeMetrics, error)
	GetActiveUsersCount(db *gorm.DB, ctx context.Context) (int64, error)
	GetAdminDashboard(db *gorm.DB, ctx context.Context, adminID string) (*dto.AdminDashboard, error)
	GetUserRetentionMetrics(db *gorm.DB, ctx context.Context, days int) (*dto.UserRetentionMetrics, error)
	GetCastingPerformanceMetrics(db *gorm.DB, ctx context.Context, employerID string, dateFrom, dateTo time.Time) (*dto.CastingPerformanceMetrics, error)
	GetMatchingEfficiencyMetrics(db *gorm.DB, ctx context.Context, days int) (*dto.MatchingEfficiencyMetrics, error)
	GetCityPerformanceMetrics(db *gorm.DB, ctx context.Context, topN int) ([]*dto.CityPerformance, error)
	GetCategoryAnalytics(db *gorm.DB, ctx context.Context) (*dto.CategoryAnalytics, error)
	GetPopularCategories(db *gorm.DB, ctx context.Context, days int, limit int) ([]*dto.CategoryStats, error)
	GetPlatformHealthMetrics(db *gorm.DB, ctx context.Context) (*dto.PlatformHealthMetrics, error)
	GenerateCustomReport(db *gorm.DB, ctx context.Context, req *dto.CustomReportRequest) (*dto.CustomReport, error)
	GetPredefinedReports(db *gorm.DB, ctx context.Context) ([]*dto.PredefinedReport, error)
	GetSystemHealthMetrics(db *gorm.DB, ctx context.Context) (*dto.SystemHealth, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type analyticsService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	castingRepo      repositories.CastingRepository
	reviewRepo       repositories.ReviewRepository
	notificationRepo repositories.NotificationRepository
	portfolioRepo    repositories.PortfolioRepository
	subscriptionRepo repositories.SubscriptionRepository
	chatRepo         repositories.ChatRepository
	analyticsRepo    repositories.AnalyticsRepository
}

// ✅ Конструктор обновлен (db убран)
func NewAnalyticsService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	castingRepo repositories.CastingRepository,
	reviewRepo repositories.ReviewRepository,
	notificationRepo repositories.NotificationRepository,
	portfolioRepo repositories.PortfolioRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	chatRepo repositories.ChatRepository,
	analyticsRepo repositories.AnalyticsRepository,
) AnalyticsService {
	return &analyticsService{
		// ❌ 'db: db,' УДАЛЕНО
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		castingRepo:      castingRepo,
		reviewRepo:       reviewRepo,
		notificationRepo: notificationRepo,
		portfolioRepo:    portfolioRepo,
		subscriptionRepo: subscriptionRepo,
		chatRepo:         chatRepo,
		analyticsRepo:    analyticsRepo,
	}
}

// Platform Overview
// GetPlatformOverview - 'db' добавлен
func (s *analyticsService) GetPlatformOverview(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.PlatformOverview, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	// ✅ 'db' пробрасывается во все внутренние вызовы
	growthMetrics, err := s.GetPlatformGrowthMetrics(db, ctx, 30)
	if err != nil {
		return nil, err
	}

	userAnalytics, err := s.GetUserAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	castingAnalytics, err := s.GetCastingAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	matchingAnalytics, err := s.GetMatchingAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	financialAnalytics, err := s.GetFinancialAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	geographicAnalytics, err := s.GetGeographicAnalytics(db, ctx)
	if err != nil {
		return nil, err
	}

	categoryAnalytics, err := s.GetCategoryAnalytics(db, ctx)
	if err != nil {
		return nil, err
	}

	performanceMetrics, err := s.GetPerformanceMetrics(db, ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	platformHealth, err := s.GetPlatformHealthMetrics(db, ctx)
	if err != nil {
		return nil, err
	}

	realTimeMetrics, err := s.GetRealTimeMetrics(db, ctx)
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

// GetPlatformGrowthMetrics - 'db' добавлен
func (s *analyticsService) GetPlatformGrowthMetrics(db *gorm.DB, ctx context.Context, days int) (*dto.GrowthMetrics, error) {
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -days)

	// ✅ Используем 'db' из параметра
	userStats, err := s.analyticsRepo.GetUserStats(db, dateFrom, dateTo)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	monthStart := time.Now().AddDate(0, -1, 0)
	// ✅ Используем 'db' из параметра
	monthStats, _ := s.analyticsRepo.GetUserStats(db, monthStart, dateTo)

	previousMonthStart := monthStart.AddDate(0, -1, 0)
	// ✅ Используем 'db' из параметра
	previousMonthStats, _ := s.analyticsRepo.GetUserStats(db, previousMonthStart, monthStart)

	monthlyGrowthRate := 0.0
	if previousMonthStats.TotalUsers > 0 {
		monthlyGrowthRate = (float64(monthStats.NewUsers) / float64(previousMonthStats.TotalUsers)) * 100
	}

	var historicalTrends []dto.DataPoint
	for i := 0; i < days; i++ {
		currentDate := dateFrom.AddDate(0, 0, i)

		// ✅ Используем 'db' из параметра
		dailyStats, err := s.analyticsRepo.GetUserStats(db, currentDate, currentDate.AddDate(0, 0, 1))
		if err != nil {
			// ✅ Используем 'db' из параметра
			dau, dauErr := s.analyticsRepo.GetDAU(db, ctx, currentDate)
			if dauErr != nil {
				continue
			}
			dailyStats.NewUsers = dau
		}

		historicalTrends = append(historicalTrends, dto.DataPoint{
			Timestamp: currentDate.Format("2006-01-02"),
			Value:     float64(dailyStats.NewUsers),
		})
	}

	// ✅ Передаем 'db'
	avgDAU, _ := s.calculateAverageDAU(db, ctx, dateFrom, dateTo)

	return &dto.GrowthMetrics{
		TotalUsers:        int(userStats.TotalUsers),
		NewUsersThisMonth: int(monthStats.NewUsers),
		MonthlyGrowthRate: monthlyGrowthRate,
		ActiveUsers:       int(avgDAU),
		ChurnRate:         s.calculateChurnRate(db, ctx, dateFrom, dateTo), // ✅ Передаем 'db'
		HistoricalTrends:  historicalTrends,
	}, nil
}

// User Analytics
// GetUserAnalytics - 'db' добавлен
func (s *analyticsService) GetUserAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.UserAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	// ✅ Передаем 'db'
	acquisition, err := s.GetUserAcquisitionMetrics(db, ctx, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	// ✅ Передаем 'db'
	retention, err := s.GetUserRetentionMetrics(db, ctx, 30)
	if err != nil {
		return nil, err
	}

	// ✅ Передаем 'db'
	avgDAU, err := s.calculateAverageDAU(db, ctx, dateFrom, dateTo)
	if err != nil {
		avgDAU = 0
	}

	activity := dto.UserActivity{
		DailyActiveUsers:   int(avgDAU),
		WeeklyActiveUsers:  int(avgDAU * 5),
		MonthlyActiveUsers: int(avgDAU * 22),
	}

	churn := dto.ChurnAnalysis{
		Rate: s.calculateChurnRate(db, ctx, dateFrom, dateTo), // ✅ Передаем 'db'
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

// GetUserAcquisitionMetrics - 'db' добавлен
func (s *analyticsService) GetUserAcquisitionMetrics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.UserAcquisitionMetrics, error) {
	// ✅ Используем 'db' из параметра
	userStats, err := s.analyticsRepo.GetUserStats(db, dateFrom, dateTo)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Передаем 'db'
	avgDAU, _ := s.calculateAverageDAU(db, ctx, dateFrom, dateTo)

	sources := []dto.DataPoint{
		{Timestamp: "Organic", Value: 500},
		{Timestamp: "Direct", Value: 300},
		{Timestamp: "Referral", Value: 150},
		{Timestamp: "Social", Value: 200},
		{Timestamp: "Paid Ads", Value: 100},
	}

	returningUsers := int(avgDAU) - int(userStats.NewUsers)
	if returningUsers < 0 {
		returningUsers = 0
	}

	return &dto.UserAcquisitionMetrics{
		NewUsers:       int(userStats.NewUsers),
		ReturningUsers: returningUsers,
		ConversionRate: 0.05,
		Sources:        sources,
	}, nil
}

// Casting Analytics
// GetCastingAnalytics - 'db' добавлен
func (s *analyticsService) GetCastingAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.CastingAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	// ✅ Используем 'db' из параметра
	castingStats, err := s.castingRepo.GetPlatformCastingStats(db, dateFrom, dateTo)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Передаем 'db'
	popularCategories, err := s.GetPopularCategories(db, ctx, 30, 10)
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
// GetMatchingAnalytics - 'db' добавлен
func (s *analyticsService) GetMatchingAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.MatchingAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	// ✅ Используем 'db' из параметра
	matchingStats, err := s.castingRepo.GetMatchingStats(db, dateFrom, dateTo)
	if err != nil {
		return nil, apperrors.InternalError(err)
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
// GetFinancialAnalytics - 'db' добавлен
func (s *analyticsService) GetFinancialAnalytics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.FinancialAnalytics, error) {
	if dateFrom.After(dateTo) {
		return nil, ErrInvalidDateRange
	}

	// ✅ Используем 'db' из параметра
	subscriptionMetrics, err := s.subscriptionRepo.GetSubscriptionMetrics(db, dateFrom, dateTo)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

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
// GetGeographicAnalytics - 'db' добавлен
func (s *analyticsService) GetGeographicAnalytics(db *gorm.DB, ctx context.Context) (*dto.GeographicAnalytics, error) {
	// ✅ Используем 'db' из параметра
	userDistribution, err := s.analyticsRepo.GetUserDistributionByCity(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Используем 'db' из параметра
	castingDistribution, err := s.analyticsRepo.GetCastingDistributionByCity(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var countries []dto.CityStats
	var cities []dto.CityPerformance

	for city, userCount := range userDistribution {
		countries = append(countries, dto.CityStats{
			Name:      city,
			UserCount: int(userCount),
			Revenue:   float64(userCount) * 25.0,
		})

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
// GetPerformanceMetrics - 'db' добавлен
func (s *analyticsService) GetPerformanceMetrics(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) (*dto.PerformanceMetrics, error) {
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
// GetRealTimeMetrics - 'db' добавлен
func (s *analyticsService) GetRealTimeMetrics(db *gorm.DB, ctx context.Context) (*dto.RealTimeMetrics, error) {
	// ✅ Используем 'db' из параметра
	activeUsers, err := s.analyticsRepo.GetActiveUsersCount(db, 15)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Используем 'db' из параметра
	activeCastings, err := s.castingRepo.GetActiveCastingsCount(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	keyMetrics := dto.KeyMetrics{
		ActiveUsers: int(activeUsers),
		NewUsers:    0,
		Revenue:     0.0,
	}

	// ✅ Используем 'db' из параметра
	dau, err := s.analyticsRepo.GetDAU(db, ctx, time.Now())
	if err != nil {
		dau = 0
	}

	activity := dto.UserActivity{
		DailyActiveUsers:   int(dau),
		WeeklyActiveUsers:  int(dau * 5),
		MonthlyActiveUsers: int(dau * 20),
	}

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

// GetActiveUsersCount - 'db' добавлен
func (s *analyticsService) GetActiveUsersCount(db *gorm.DB, ctx context.Context) (int64, error) {
	// ✅ Используем 'db' из параметра
	count, err := s.analyticsRepo.GetActiveUsersCount(db, 15)
	if err != nil {
		return 0, apperrors.InternalError(err)
	}
	return count, nil
}

// Admin Dashboard
// GetAdminDashboard - 'db' добавлен
func (s *analyticsService) GetAdminDashboard(db *gorm.DB, ctx context.Context, adminID string) (*dto.AdminDashboard, error) {
	// ✅ Используем 'db' из параметра
	admin, err := s.userRepo.FindByID(db, adminID)
	if err != nil {
		return nil, handleRepositoryError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return nil, apperrors.ErrInsufficientPermissions
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

// calculateAverageDAU - 'db' добавлен
func (s *analyticsService) calculateAverageDAU(db *gorm.DB, ctx context.Context, from, to time.Time) (float64, error) {
	totalDAU := int64(0)
	numDays := 0

	currentDay := from.Truncate(24 * time.Hour)
	endDate := to.Truncate(24 * time.Hour)

	for ; !currentDay.After(endDate); currentDay = currentDay.AddDate(0, 0, 1) {
		// ✅ Используем 'db' из параметра
		dau, err := s.analyticsRepo.GetDAU(db, ctx, currentDay)
		if err != nil {
			return 0, apperrors.InternalError(err)
		}
		totalDAU += dau
		numDays++
	}

	if numDays == 0 {
		return 0, nil
	}

	return float64(totalDAU) / float64(numDays), nil
}

// calculateRetentionRate - 'db' добавлен
func (s *analyticsService) calculateRetentionRate(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) float64 {
	// TODO: Реализовать с использованием analyticsRepo(db, ...)
	return 0.75
}

// calculateChurnRate - 'db' добавлен
func (s *analyticsService) calculateChurnRate(db *gorm.DB, ctx context.Context, dateFrom, dateTo time.Time) float64 {
	// TODO: Реализовать с использованием analyticsRepo(db, ...)
	return 0.08
}

// ==============================
// Stub implementations
// ==============================

// GetUserRetentionMetrics - 'db' добавлен
func (s *analyticsService) GetUserRetentionMetrics(db *gorm.DB, ctx context.Context, days int) (*dto.UserRetentionMetrics, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
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

// GetCastingPerformanceMetrics - 'db' добавлен
func (s *analyticsService) GetCastingPerformanceMetrics(db *gorm.DB, ctx context.Context, employerID string, dateFrom, dateTo time.Time) (*dto.CastingPerformanceMetrics, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return &dto.CastingPerformanceMetrics{
		CompletionRate: 0.85,
		EngagementRate: 0.72,
	}, nil
}

// GetMatchingEfficiencyMetrics - 'db' добавлен
func (s *analyticsService) GetMatchingEfficiencyMetrics(db *gorm.DB, ctx context.Context, days int) (*dto.MatchingEfficiencyMetrics, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return &dto.MatchingEfficiencyMetrics{
		AverageMatchTime: 2.5,
		MatchRate:        0.65,
		DropOffRate:      0.15,
	}, nil
}

// GetCityPerformanceMetrics - 'db' добавлен
func (s *analyticsService) GetCityPerformanceMetrics(db *gorm.DB, ctx context.Context, topN int) ([]*dto.CityPerformance, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return []*dto.CityPerformance{
		{Name: "Almaty", EngagementRate: 0.75, ResponseTime: 1.2},
		{Name: "Astana", EngagementRate: 0.70, ResponseTime: 1.5},
	}, nil
}

// GetCategoryAnalytics - 'db' добавлен
func (s *analyticsService) GetCategoryAnalytics(db *gorm.DB, ctx context.Context) (*dto.CategoryAnalytics, error) {
	// ✅ Передаем 'db'
	categories, err := s.GetPopularCategories(db, ctx, 30, 10)
	if err != nil {
		return nil, err
	}

	return &dto.CategoryAnalytics{
		Categories: categories,
	}, nil
}

// GetPopularCategories - 'db' добавлен
func (s *analyticsService) GetPopularCategories(db *gorm.DB, ctx context.Context, days int, limit int) ([]*dto.CategoryStats, error) {
	// ✅ Используем 'db' из параметра
	popularCategories, err := s.castingRepo.GetPopularCategories(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
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

// GetPlatformHealthMetrics - 'db' добавлен
func (s *analyticsService) GetPlatformHealthMetrics(db *gorm.DB, ctx context.Context) (*dto.PlatformHealthMetrics, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
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

// GenerateCustomReport - 'db' добавлен
func (s *analyticsService) GenerateCustomReport(db *gorm.DB, ctx context.Context, req *dto.CustomReportRequest) (*dto.CustomReport, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return &dto.CustomReport{
		ID:          "report-" + time.Now().Format("20060102150405"),
		Name:        req.Type,
		Description: "Custom report for " + req.Period,
		Data:        map[string]any{},
	}, nil
}

// GetPredefinedReports - 'db' добавлен
func (s *analyticsService) GetPredefinedReports(db *gorm.DB, ctx context.Context) ([]*dto.PredefinedReport, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return []*dto.PredefinedReport{ // <-- Исправлено на Report (единственное число)
		// ✅ Добавлены &dto.PredefinedReport{} для создания указателей
		&dto.PredefinedReport{Name: "User Growth Report", Description: "Monthly user growth analysis", Category: "Users"},
		&dto.PredefinedReport{Name: "Revenue Report", Description: "Financial performance overview", Category: "Finance"},
		&dto.PredefinedReport{Name: "Casting Performance", Description: "Casting success metrics", Category: "Castings"},
	}, nil
}

// GetSystemHealthMetrics - 'db' добавлен
func (s *analyticsService) GetSystemHealthMetrics(db *gorm.DB, ctx context.Context) (*dto.SystemHealth, error) {
	// ✅ Используем 'db' из параметра (для будущей логики)
	return &dto.SystemHealth{}, nil
}

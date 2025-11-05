package repositories

import (
	"context"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

// AnalyticsRepository отвечает за операции с данными отслеживания использования и аналитики
type AnalyticsRepository interface {
	// ============================================
	// Usage Tracking (уже было)
	// ============================================

	// TrackEvent записывает новое событие отслеживания в базу данных
	TrackEvent(db *gorm.DB, ctx context.Context, event *models.UsageTrack) error

	// GetEventCountsByType возвращает количество событий, сгруппированных по типу, за период
	GetEventCountsByType(db *gorm.DB, ctx context.Context, from, to time.Time) (map[string]int64, error)

	// GetUniqueActiveUsersByEvent возвращает количество уникальных пользователей,
	// совершивших определенное событие за период
	GetUniqueActiveUsersByEvent(db *gorm.DB, ctx context.Context, eventType string, from, to time.Time) (int64, error)

	// GetDAU (Daily Active Users) возвращает количество уникальных пользователей
	// с любым событием за указанный день
	GetDAU(db *gorm.DB, ctx context.Context, date time.Time) (int64, error)

	// ============================================
	// User Analytics (из user_repository.go)
	// ============================================

	// GetActiveUsersCount возвращает количество активных пользователей за последние N минут
	GetActiveUsersCount(db *gorm.DB, minutes int) (int64, error)

	// GetUserDistributionByCity возвращает распределение пользователей по городам
	GetUserDistributionByCity(db *gorm.DB) (map[string]int64, error)

	// GetUserStats возвращает статистику пользователей за период
	GetUserStats(db *gorm.DB, dateFrom, dateTo time.Time) (*UserStats, error)

	// GetRegistrationStats возвращает статистику регистраций за последние N дней
	GetRegistrationStats(db *gorm.DB, days int) (*RegistrationStats, error)

	// ============================================
	// Casting Analytics (из casting_repository.go)
	// ============================================

	// GetPlatformCastingStats возвращает статистику по кастингам платформы
	GetPlatformCastingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*PlatformCastingStats, error)

	// GetMatchingStats возвращает статистику по матчингу
	GetMatchingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*MatchingStats, error)

	// GetCastingDistributionByCity возвращает распределение кастингов по городам
	GetCastingDistributionByCity(db *gorm.DB) (map[string]int64, error)

	// GetActiveCastingsCount возвращает количество активных кастингов
	GetActiveCastingsCount(db *gorm.DB) (int64, error)

	// GetPopularCategories возвращает популярные категории кастингов
	GetPopularCategories(db *gorm.DB, limit int) ([]CategoryCount, error)

	// ============================================
	// Combined Platform Analytics (новое)
	// ============================================

	// GetPlatformOverview возвращает общую сводку по платформе
	GetPlatformOverview(db *gorm.DB) (*PlatformOverview, error)

	// GetActivityTrend возвращает тренд активности за период
	GetActivityTrend(db *gorm.DB, dateFrom, dateTo time.Time, interval string) ([]ActivityPoint, error)
}

type analyticsRepository struct {
	// ✅ Пустая структура - db больше не хранится здесь
}

// NewAnalyticsRepository создает новый экземпляр AnalyticsRepository
func NewAnalyticsRepository() AnalyticsRepository {
	return &analyticsRepository{}
}

// ============================================
// Usage Tracking Implementation
// ============================================

// TrackEvent записывает новое событие отслеживания в базу данных
func (r *analyticsRepository) TrackEvent(db *gorm.DB, ctx context.Context, event *models.UsageTrack) error {
	return db.WithContext(ctx).Create(event).Error
}

// GetEventCountsByType возвращает количество событий, сгруппированных по типу, за период
func (r *analyticsRepository) GetEventCountsByType(db *gorm.DB, ctx context.Context, from, to time.Time) (map[string]int64, error) {
	var results []struct {
		EventType string `gorm:"column:event_type"`
		Count     int64  `gorm:"column:count"`
	}

	err := db.WithContext(ctx).
		Model(&models.UsageTrack{}).
		Select("event_type, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", from, to).
		Group("event_type").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.EventType] = res.Count
	}

	return counts, nil
}

// GetUniqueActiveUsersByEvent возвращает количество уникальных пользователей,
// совершивших определенное событие за период
func (r *analyticsRepository) GetUniqueActiveUsersByEvent(db *gorm.DB, ctx context.Context, eventType string, from, to time.Time) (int64, error) {
	var count int64

	err := db.WithContext(ctx).
		Model(&models.UsageTrack{}).
		Where("event_type = ?", eventType).
		Where("created_at BETWEEN ? AND ?", from, to).
		Distinct("user_id").
		Count(&count).Error

	return count, err
}

// GetDAU (Daily Active Users) возвращает количество уникальных пользователей
// с любым событием за указанный день
func (r *analyticsRepository) GetDAU(db *gorm.DB, ctx context.Context, date time.Time) (int64, error) {
	var count int64
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24*time.Hour - time.Nanosecond)

	err := db.WithContext(ctx).
		Model(&models.UsageTrack{}).
		Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Distinct("user_id").
		Count(&count).Error

	return count, err
}

// ============================================
// User Analytics Implementation
// ============================================

// GetActiveUsersCount возвращает количество активных пользователей за последние N минут
func (r *analyticsRepository) GetActiveUsersCount(db *gorm.DB, minutes int) (int64, error) {
	var count int64

	// Если есть поле last_active_at, используем его
	if db.Migrator().HasColumn(&models.User{}, "last_active_at") {
		activeSince := time.Now().Add(-time.Duration(minutes) * time.Minute)
		err := db.Model(&models.User{}).
			Where("last_active_at >= ?", activeSince).
			Count(&count).Error
		return count, err
	}

	// Альтернатива: считаем пользователей, которые были активны сегодня
	today := time.Now().Truncate(24 * time.Hour)
	err := db.Model(&models.User{}).
		Where("created_at >= ? OR updated_at >= ?", today, today).
		Count(&count).Error

	return count, err
}

// GetUserDistributionByCity возвращает распределение пользователей по городам
func (r *analyticsRepository) GetUserDistributionByCity(db *gorm.DB) (map[string]int64, error) {
	type CityCount struct {
		City  string
		Count int64
	}

	var cityCounts []CityCount
	result := make(map[string]int64)

	// Сначала проверяем, есть ли город в профиле модели
	if db.Migrator().HasColumn(&models.ModelProfile{}, "city") {
		err := db.Model(&models.ModelProfile{}).
			Select("city, COUNT(*) as count").
			Where("city IS NOT NULL AND city != ''").
			Group("city").
			Find(&cityCounts).Error

		if err == nil && len(cityCounts) > 0 {
			for _, cc := range cityCounts {
				result[cc.City] = cc.Count
			}
			return result, nil
		}
	}

	// Проверяем профили работодателей
	if db.Migrator().HasColumn(&models.EmployerProfile{}, "city") {
		err := db.Model(&models.EmployerProfile{}).
			Select("city, COUNT(*) as count").
			Where("city IS NOT NULL AND city != ''").
			Group("city").
			Find(&cityCounts).Error

		if err == nil && len(cityCounts) > 0 {
			for _, cc := range cityCounts {
				result[cc.City] += cc.Count // Добавляем к существующим
			}
			return result, nil
		}
	}

	return result, nil
}

// UserStats структура для статистики пользователей
type UserStats struct {
	TotalUsers  int64 `json:"total_users"`
	NewUsers    int64 `json:"new_users"`
	ActiveUsers int64 `json:"active_users"`
}

// GetUserStats возвращает статистику пользователей за период
func (r *analyticsRepository) GetUserStats(db *gorm.DB, dateFrom, dateTo time.Time) (*UserStats, error) {
	var stats UserStats

	// Count total users
	if err := db.Model(&models.User{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	// Count new users within the range
	if err := db.Model(&models.User{}).
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).
		Count(&stats.NewUsers).Error; err != nil {
		return nil, err
	}

	// Count active users – assumes you track last_active_at or similar
	if db.Migrator().HasColumn(&models.User{}, "last_active_at") {
		if err := db.Model(&models.User{}).
			Where("last_active_at BETWEEN ? AND ?", dateFrom, dateTo).
			Count(&stats.ActiveUsers).Error; err != nil {
			return nil, err
		}
	}

	return &stats, nil
}

// RegistrationStats структура для статистики регистраций
type RegistrationStats struct {
	Total           int64            `json:"total"`
	Today           int64            `json:"today"`
	ThisWeek        int64            `json:"this_week"`
	ThisMonth       int64            `json:"this_month"`
	ByRole          map[string]int64 `json:"by_role"`
	VerifiedCount   int64            `json:"verified_count"`
	UnverifiedCount int64            `json:"unverified_count"`
}

// GetRegistrationStats возвращает статистику регистраций за последние N дней
func (r *analyticsRepository) GetRegistrationStats(db *gorm.DB, days int) (*RegistrationStats, error) {
	var stats RegistrationStats
	now := time.Now()

	// Total count
	if err := db.Model(&models.User{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Today
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := db.Model(&models.User{}).Where("created_at >= ?", todayStart).Count(&stats.Today).Error; err != nil {
		return nil, err
	}

	// This week
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))
	if err := db.Model(&models.User{}).Where("created_at >= ?", weekStart).Count(&stats.ThisWeek).Error; err != nil {
		return nil, err
	}

	// This month
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := db.Model(&models.User{}).Where("created_at >= ?", monthStart).Count(&stats.ThisMonth).Error; err != nil {
		return nil, err
	}

	// By role
	stats.ByRole = make(map[string]int64)
	roles := []models.UserRole{models.UserRoleModel, models.UserRoleEmployer, models.UserRoleAdmin}

	for _, role := range roles {
		var count int64
		if err := db.Model(&models.User{}).Where("role = ?", role).Count(&count).Error; err != nil {
			return nil, err
		}
		stats.ByRole[string(role)] = count
	}

	// Verified counts
	if err := db.Model(&models.User{}).Where("is_verified = ?", true).Count(&stats.VerifiedCount).Error; err != nil {
		return nil, err
	}
	stats.UnverifiedCount = stats.Total - stats.VerifiedCount

	return &stats, nil
}

// ============================================
// Casting Analytics Implementation
// ============================================

// PlatformCastingStats структура для статистики кастингов платформы
type PlatformCastingStats struct {
	TotalCastings     int64   `json:"totalCastings"`
	ActiveCastings    int64   `json:"activeCastings"`
	SuccessRate       float64 `json:"successRate"`
	AvgResponseRate   float64 `json:"avgResponseRate"`
	AvgResponseTime   float64 `json:"avgResponseTime"`
	AcceptedResponses int64   `json:"acceptedResponses"`
	ClosedCastings    int64   `json:"closedCastings"`
}

// GetPlatformCastingStats возвращает статистику по кастингам платформы
func (r *analyticsRepository) GetPlatformCastingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*PlatformCastingStats, error) {
	var stats PlatformCastingStats

	// Total castings in period
	if err := db.Model(&models.Casting{}).
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).
		Count(&stats.TotalCastings).Error; err != nil {
		return nil, err
	}

	// Active castings
	if err := db.Model(&models.Casting{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", models.CastingStatusActive, dateFrom, dateTo).
		Count(&stats.ActiveCastings).Error; err != nil {
		return nil, err
	}

	// Calculate success rate (castings with responses)
	var castingsWithResponses int64
	subquery := db.Model(&models.CastingResponse{}).
		Select("DISTINCT casting_id").
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo)

	if err := db.Model(&models.Casting{}).
		Where("id IN (?) AND created_at BETWEEN ? AND ?", subquery, dateFrom, dateTo).
		Count(&castingsWithResponses).Error; err != nil {
		return nil, err
	}

	if stats.TotalCastings > 0 {
		stats.SuccessRate = float64(castingsWithResponses) / float64(stats.TotalCastings) * 100
	}

	// Average response rate calculation would require more complex business logic
	// For now, setting placeholder values
	stats.AvgResponseRate = 0.0
	stats.AvgResponseTime = 0.0

	return &stats, nil
}

// MatchingStats структура для статистики матчинга
type MatchingStats struct {
	TotalMatches    int64   `json:"totalMatches"`
	AvgMatchScore   float64 `json:"avgMatchScore"`
	AvgSatisfaction float64 `json:"avgSatisfaction"`
	MatchRate       float64 `json:"matchRate"`
	ResponseRate    float64 `json:"responseRate"`
	TimeToMatch     float64 `json:"timeToMatch"` // in hours
}

// GetMatchingStats возвращает статистику по матчингу
func (r *analyticsRepository) GetMatchingStats(db *gorm.DB, dateFrom, dateTo time.Time) (*MatchingStats, error) {
	var stats MatchingStats

	// Total matches (responses in period)
	if err := db.Model(&models.CastingResponse{}).
		Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).
		Count(&stats.TotalMatches).Error; err != nil {
		return nil, err
	}

	// Placeholder values for complex calculations
	// These would require additional tables/fields for match scoring and satisfaction
	stats.AvgMatchScore = 0.0
	stats.AvgSatisfaction = 0.0
	stats.MatchRate = 0.0
	stats.ResponseRate = 0.0
	stats.TimeToMatch = 0.0

	return &stats, nil
}

// GetCastingDistributionByCity возвращает распределение кастингов по городам
func (r *analyticsRepository) GetCastingDistributionByCity(db *gorm.DB) (map[string]int64, error) {
	type CityCount struct {
		City  string
		Count int64
	}

	var cityCounts []CityCount
	result := make(map[string]int64)

	err := db.Model(&models.Casting{}).
		Select("city, COUNT(*) as count").
		Where("status = ?", models.CastingStatusActive).
		Group("city").
		Find(&cityCounts).Error

	if err != nil {
		return nil, err
	}

	for _, cc := range cityCounts {
		result[cc.City] = cc.Count
	}

	return result, nil
}

// GetActiveCastingsCount возвращает количество активных кастингов
func (r *analyticsRepository) GetActiveCastingsCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&models.Casting{}).
		Where("status = ?", models.CastingStatusActive).
		Count(&count).Error
	return count, err
}

// CategoryCount структура для подсчета категорий
type CategoryCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// GetPopularCategories возвращает популярные категории кастингов
func (r *analyticsRepository) GetPopularCategories(db *gorm.DB, limit int) ([]CategoryCount, error) {
	var categories []CategoryCount

	// Using raw SQL for JSONB array extraction (PostgreSQL specific)
	query := `
		SELECT category as name, COUNT(*) as count
		FROM (
			SELECT jsonb_array_elements_text(categories) as category
			FROM castings
			WHERE status = ? AND categories IS NOT NULL
		) as extracted_categories
		GROUP BY category
		ORDER BY count DESC
		LIMIT ?
	`

	err := db.Raw(query, models.CastingStatusActive, limit).Scan(&categories).Error
	return categories, err
}

// ============================================
// Combined Platform Analytics (новое)
// ============================================

// PlatformOverview структура для общей сводки по платформе
type PlatformOverview struct {
	TotalUsers        int64              `json:"total_users"`
	ActiveUsers       int64              `json:"active_users"`
	TotalCastings     int64              `json:"total_castings"`
	ActiveCastings    int64              `json:"active_castings"`
	TotalResponses    int64              `json:"total_responses"`
	UsersByCity       map[string]int64   `json:"users_by_city"`
	CastingsByCity    map[string]int64   `json:"castings_by_city"`
	PopularCategories []CategoryCount    `json:"popular_categories"`
	RegistrationStats *RegistrationStats `json:"registration_stats"`
}

// GetPlatformOverview возвращает общую сводку по платформе
func (r *analyticsRepository) GetPlatformOverview(db *gorm.DB) (*PlatformOverview, error) {
	overview := &PlatformOverview{}

	// Total users
	if err := db.Model(&models.User{}).Count(&overview.TotalUsers).Error; err != nil {
		return nil, err
	}

	// Active users (last 30 days)
	monthAgo := time.Now().AddDate(0, -1, 0)
	if db.Migrator().HasColumn(&models.User{}, "last_active_at") {
		if err := db.Model(&models.User{}).
			Where("last_active_at >= ?", monthAgo).
			Count(&overview.ActiveUsers).Error; err != nil {
			return nil, err
		}
	}

	// Total castings
	if err := db.Model(&models.Casting{}).Count(&overview.TotalCastings).Error; err != nil {
		return nil, err
	}

	// Active castings
	if err := db.Model(&models.Casting{}).
		Where("status = ?", models.CastingStatusActive).
		Count(&overview.ActiveCastings).Error; err != nil {
		return nil, err
	}

	// Total responses
	if err := db.Model(&models.CastingResponse{}).Count(&overview.TotalResponses).Error; err != nil {
		return nil, err
	}

	// Users by city
	usersByCity, err := r.GetUserDistributionByCity(db)
	if err != nil {
		return nil, err
	}
	overview.UsersByCity = usersByCity

	// Castings by city
	castingsByCity, err := r.GetCastingDistributionByCity(db)
	if err != nil {
		return nil, err
	}
	overview.CastingsByCity = castingsByCity

	// Popular categories
	categories, err := r.GetPopularCategories(db, 10)
	if err != nil {
		return nil, err
	}
	overview.PopularCategories = categories

	// Registration stats
	regStats, err := r.GetRegistrationStats(db, 30)
	if err != nil {
		return nil, err
	}
	overview.RegistrationStats = regStats

	return overview, nil
}

// ActivityPoint точка активности для графика
type ActivityPoint struct {
	Timestamp time.Time `json:"timestamp"`
	UserCount int64     `json:"user_count"`
	Castings  int64     `json:"castings"`
	Responses int64     `json:"responses"`
}

// GetActivityTrend возвращает тренд активности за период
func (r *analyticsRepository) GetActivityTrend(db *gorm.DB, dateFrom, dateTo time.Time, interval string) ([]ActivityPoint, error) {
	var points []ActivityPoint

	// Определяем формат группировки в зависимости от интервала
	var dateFormat string
	switch interval {
	case "hour":
		dateFormat = "YYYY-MM-DD HH24:00:00"
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "YYYY-\"W\"IW" // ISO week
	case "month":
		dateFormat = "YYYY-MM"
	default:
		dateFormat = "YYYY-MM-DD"
	}

	// Используем подзапросы для эффективности
	query := `
		WITH time_series AS (
			SELECT 
				TO_CHAR(created_at, ?) as period,
				created_at as timestamp
			FROM users
			WHERE created_at BETWEEN ? AND ?
			GROUP BY period, timestamp
		)
		SELECT 
			MIN(ts.timestamp) as timestamp,
			COUNT(DISTINCT u.id) as user_count,
			COUNT(DISTINCT c.id) as castings,
			COUNT(DISTINCT cr.id) as responses
		FROM time_series ts
		LEFT JOIN users u ON TO_CHAR(u.created_at, ?) = ts.period
		LEFT JOIN castings c ON TO_CHAR(c.created_at, ?) = ts.period
		LEFT JOIN casting_responses cr ON TO_CHAR(cr.created_at, ?) = ts.period
		GROUP BY ts.period
		ORDER BY MIN(ts.timestamp)
	`

	err := db.Raw(query, dateFormat, dateFrom, dateTo, dateFormat, dateFormat, dateFormat).
		Scan(&points).Error

	return points, err
}

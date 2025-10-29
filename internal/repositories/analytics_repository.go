package repositories

import (
	"context"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

// AnalyticsRepository отвечает за операции с данными отслеживания использования
type AnalyticsRepository interface {
	// TrackEvent записывает новое событие отслеживания в базу данных
	TrackEvent(ctx context.Context, event *models.UsageTrack) error

	// GetEventCountsByType возвращает количество событий, сгруппированных по типу, за период
	GetEventCountsByType(ctx context.Context, from, to time.Time) (map[string]int64, error)

	// GetUniqueActiveUsersByEvent возвращает количество уникальных пользователей,
	// совершивших определенное событие за период
	GetUniqueActiveUsersByEvent(ctx context.Context, eventType string, from, to time.Time) (int64, error)

	// GetDAU (Daily Active Users) возвращает количество уникальных пользователей
	// с любым событием за указанный день
	GetDAU(ctx context.Context, date time.Time) (int64, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

// NewAnalyticsRepository создает новый экземпляр AnalyticsRepository
func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// TrackEvent записывает новое событие отслеживания в базу данных
func (r *analyticsRepository) TrackEvent(ctx context.Context, event *models.UsageTrack) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// GetEventCountsByType возвращает количество событий, сгруппированных по типу, за период
func (r *analyticsRepository) GetEventCountsByType(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	var results []struct {
		EventType string `gorm:"column:event_type"`
		Count     int64  `gorm:"column:count"`
	}

	err := r.db.WithContext(ctx).
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
func (r *analyticsRepository) GetUniqueActiveUsersByEvent(ctx context.Context, eventType string, from, to time.Time) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.UsageTrack{}).
		Where("event_type = ?", eventType).
		Where("created_at BETWEEN ? AND ?", from, to).
		Distinct("user_id").
		Count(&count).Error

	return count, err
}

// GetDAU (Daily Active Users) возвращает количество уникальных пользователей
// с любым событием за указанный день
func (r *analyticsRepository) GetDAU(ctx context.Context, date time.Time) (int64, error) {
	var count int64
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24*time.Hour - time.Nanosecond)

	err := r.db.WithContext(ctx).
		Model(&models.UsageTrack{}).
		Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Distinct("user_id").
		Count(&count).Error

	return count, err
}

package repositories

import (
	"database/sql"
)

type AnalyticsRepository interface {
	GetProfileViews(modelID string) (int, error)
	GetRating(modelID string) (float64, error)
	GetIncome(modelID string) (float64, error)
	GetResponseCount(modelID string) (int, error)
}

type analyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) GetProfileViews(modelID string) (int, error) {
	var views int
	err := r.db.QueryRow(`
        SELECT profile_views FROM model_profiles WHERE id = $1
    `, modelID).Scan(&views)
	return views, err
}

func (r *analyticsRepository) GetRating(modelID string) (float64, error) {
	var avg sql.NullFloat64
	err := r.db.QueryRow(`
        SELECT AVG(score) FROM ratings WHERE model_id = $1
    `, modelID).Scan(&avg)
	if avg.Valid {
		return avg.Float64, err
	}
	return 0, err
}

func (r *analyticsRepository) GetIncome(modelID string) (float64, error) {
	var income sql.NullFloat64
	err := r.db.QueryRow(`
        SELECT COALESCE(SUM(c.payment_max), 0)
        FROM castings c
        JOIN casting_responses r ON r.casting_id = c.id
        WHERE r.model_id = $1 AND r.status = 'accepted'
    `, modelID).Scan(&income)
	if income.Valid {
		return income.Float64, err
	}
	return 0, err
}

func (r *analyticsRepository) GetResponseCount(modelID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
        SELECT COUNT(*) FROM casting_responses WHERE model_id = $1
    `, modelID).Scan(&count)
	return count, err
}

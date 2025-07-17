package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/lib/pq"
	"mwork_front_fn/backend/models"
)

// ✅ Экспортируемый интерфейс
type ModelProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.ModelProfile, error)
	Create(ctx context.Context, profile *models.ModelProfile) error
}

// ✅ Приватная реализация
type modelProfileRepositoryImpl struct {
	db *sql.DB
}

// ✅ Конструктор, возвращает интерфейс
func NewModelProfileRepository(db *sql.DB) ModelProfileRepository {
	return &modelProfileRepositoryImpl{db: db}
}

func (r *modelProfileRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*models.ModelProfile, error) {
	var profile models.ModelProfile
	var languages, categories []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, age, height, weight, gender, experience,
		       hourly_rate, description, clothing_size, shoe_size, city,
		       languages, categories, barter_accepted, profile_views,
		       rating, is_public, created_at, updated_at
		FROM model_profiles WHERE user_id = $1
	`, userID).Scan(
		&profile.ID, &profile.UserID, &profile.Name, &profile.Age, &profile.Height,
		&profile.Weight, &profile.Gender, &profile.Experience, &profile.HourlyRate,
		&profile.Description, &profile.ClothingSize, &profile.ShoeSize, &profile.City,
		&languages, &categories, &profile.BarterAccepted, &profile.ProfileViews,
		&profile.Rating, &profile.IsPublic, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(languages, &profile.Languages)
	json.Unmarshal(categories, &profile.Categories)

	return &profile, nil
}

func (r *modelProfileRepositoryImpl) Create(ctx context.Context, profile *models.ModelProfile) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO model_profiles (
			user_id, name, age, height, weight, gender, experience,
			hourly_rate, description, clothing_size, shoe_size,
			city, languages, categories, barter_accepted, is_public
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id, created_at, updated_at
	`,
		profile.UserID, profile.Name, profile.Age, profile.Height, profile.Weight,
		profile.Gender, profile.Experience, profile.HourlyRate, profile.Description,
		profile.ClothingSize, profile.ShoeSize, profile.City,
		pq.StringArray(profile.Languages), pq.StringArray(profile.Categories),
		profile.BarterAccepted, profile.IsPublic,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

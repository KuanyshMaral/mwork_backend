package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"mwork_backend/internal/models"
)

type CastingRepository struct {
	db *sql.DB
}

func NewCastingRepository(db *sql.DB) *CastingRepository {
	return &CastingRepository{db: db}
}

func (r *CastingRepository) Create(ctx context.Context, c *models.Casting) error {
	categoriesJSON, _ := json.Marshal(c.Categories)
	languagesJSON, _ := json.Marshal(c.Languages)

	return r.db.QueryRowContext(ctx, `
		INSERT INTO castings (
			employer_id, title, description, payment_min, payment_max, 
			casting_date, casting_time, address, city, categories, gender, 
			age_min, age_max, height_min, height_max, weight_min, weight_max, 
			clothing_size, shoe_size, experience_level, languages, 
			job_type, status
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23
		)
		RETURNING id, created_at, updated_at
	`,
		c.EmployerID, c.Title, c.Description, c.PaymentMin, c.PaymentMax,
		c.CastingDate, c.CastingTime, c.Address, c.City, categoriesJSON, c.Gender,
		c.AgeMin, c.AgeMax, c.HeightMin, c.HeightMax, c.WeightMin, c.WeightMax,
		c.ClothingSize, c.ShoeSize, c.ExperienceLevel, languagesJSON,
		c.JobType, c.Status,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CastingRepository) GetByID(ctx context.Context, id string) (*models.Casting, error) {
	var c models.Casting
	var categoriesJSON, languagesJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, employer_id, title, description, payment_min, payment_max, 
			casting_date, casting_time, address, city, categories, gender, 
			age_min, age_max, height_min, height_max, weight_min, weight_max, 
			clothing_size, shoe_size, experience_level, languages, 
			job_type, status, views, created_at, updated_at
		FROM castings WHERE id = $1
	`, id).Scan(
		&c.ID, &c.EmployerID, &c.Title, &c.Description, &c.PaymentMin, &c.PaymentMax,
		&c.CastingDate, &c.CastingTime, &c.Address, &c.City, &categoriesJSON, &c.Gender,
		&c.AgeMin, &c.AgeMax, &c.HeightMin, &c.HeightMax, &c.WeightMin, &c.WeightMax,
		&c.ClothingSize, &c.ShoeSize, &c.ExperienceLevel, &languagesJSON,
		&c.JobType, &c.Status, &c.Views, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(categoriesJSON, &c.Categories)
	_ = json.Unmarshal(languagesJSON, &c.Languages)

	return &c, nil
}

func (r *CastingRepository) Update(ctx context.Context, c *models.Casting) error {
	categoriesJSON, _ := json.Marshal(c.Categories)
	languagesJSON, _ := json.Marshal(c.Languages)

	_, err := r.db.ExecContext(ctx, `
		UPDATE castings SET
			title = $1, description = $2, payment_min = $3, payment_max = $4,
			casting_date = $5, casting_time = $6, address = $7, city = $8,
			categories = $9, gender = $10, age_min = $11, age_max = $12,
			height_min = $13, height_max = $14, weight_min = $15, weight_max = $16,
			clothing_size = $17, shoe_size = $18, experience_level = $19, languages = $20,
			job_type = $21, status = $22, updated_at = now()
		WHERE id = $23
	`,
		c.Title, c.Description, c.PaymentMin, c.PaymentMax,
		c.CastingDate, c.CastingTime, c.Address, c.City,
		categoriesJSON, c.Gender, c.AgeMin, c.AgeMax,
		c.HeightMin, c.HeightMax, c.WeightMin, c.WeightMax,
		c.ClothingSize, c.ShoeSize, c.ExperienceLevel, languagesJSON,
		c.JobType, c.Status, c.ID,
	)
	return err
}

func (r *CastingRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM castings WHERE id = $1`, id)
	return err
}

func (r *CastingRepository) ListByEmployer(ctx context.Context, employerID string) ([]*models.Casting, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id, employer_id, title, description, payment_min, payment_max,
			casting_date, casting_time, address, city, categories, gender,
			age_min, age_max, height_min, height_max, weight_min, weight_max,
			clothing_size, shoe_size, experience_level, languages,
			job_type, status, views, created_at, updated_at
		FROM castings WHERE employer_id = $1 ORDER BY created_at DESC
	`, employerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var castings []*models.Casting

	for rows.Next() {
		var c models.Casting
		var categoriesJSON, languagesJSON []byte

		err := rows.Scan(
			&c.ID, &c.EmployerID, &c.Title, &c.Description, &c.PaymentMin, &c.PaymentMax,
			&c.CastingDate, &c.CastingTime, &c.Address, &c.City, &categoriesJSON, &c.Gender,
			&c.AgeMin, &c.AgeMax, &c.HeightMin, &c.HeightMax, &c.WeightMin, &c.WeightMax,
			&c.ClothingSize, &c.ShoeSize, &c.ExperienceLevel, &languagesJSON,
			&c.JobType, &c.Status, &c.Views, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			continue
		}

		_ = json.Unmarshal(categoriesJSON, &c.Categories)
		_ = json.Unmarshal(languagesJSON, &c.Languages)

		castings = append(castings, &c)
	}

	return castings, nil
}

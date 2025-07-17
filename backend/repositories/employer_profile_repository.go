package repositories

import (
	"context"
	"database/sql"
	"mwork_front_fn/backend/models"
)

type EmployerProfileRepository struct {
	db *sql.DB
}

func NewEmployerProfileRepository(db *sql.DB) *EmployerProfileRepository {
	return &EmployerProfileRepository{db: db}
}

func (r *EmployerProfileRepository) Create(ctx context.Context, profile *models.EmployerProfile) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO employer_profiles (
			user_id, company_name, contact_person, phone, website,
			city, company_type, description, is_verified
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at
	`,
		profile.UserID, profile.CompanyName, profile.ContactPerson, profile.Phone,
		profile.Website, profile.City, profile.CompanyType, profile.Description,
		profile.IsVerified,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

func (r *EmployerProfileRepository) GetByUserID(ctx context.Context, userID string) (*models.EmployerProfile, error) {
	var profile models.EmployerProfile

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, company_name, contact_person, phone, website,
		       city, company_type, description, is_verified, created_at, updated_at
		FROM employer_profiles WHERE user_id = $1
	`, userID).Scan(
		&profile.ID, &profile.UserID, &profile.CompanyName, &profile.ContactPerson,
		&profile.Phone, &profile.Website, &profile.City, &profile.CompanyType,
		&profile.Description, &profile.IsVerified, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &profile, nil
}

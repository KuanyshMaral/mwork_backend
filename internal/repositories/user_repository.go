package repositories

import (
	"context"
	"database/sql"
	"mwork_backend/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users WHERE email = $1 LIMIT 1
	`, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role,
		&user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users WHERE id = $1 LIMIT 1
	`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role,
		&user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, user.Email, user.PasswordHash, user.Role, user.Status).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET email = $1, password_hash = $2, role = $3, status = $4, updated_at = NOW()
		WHERE id = $5
	`, user.Email, user.PasswordHash, user.Role, user.Status, user.ID)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, role, status, is_verified, verification_token, created_at, updated_at
		FROM users
		WHERE verification_token = $1
	`

	row := r.db.QueryRowContext(ctx, query, token)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.IsVerified,
		&user.VerificationToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByResetToken(ctx context.Context, token string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, role, status, is_verified, verification_token, reset_token, reset_token_exp, created_at, updated_at
		FROM users WHERE reset_token = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.IsVerified,
		&user.VerificationToken,
		&user.ResetToken,
		&user.ResetTokenExp,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

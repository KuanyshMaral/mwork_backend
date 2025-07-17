package repositories

import (
	"context"
	"database/sql"
	"errors"
	"mwork_front_fn/backend/models"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Сохраняет токен
func (r *RefreshTokenRepository) Save(ctx context.Context, token *models.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.ID, token.UserID, token.Token, token.CreatedAt, token.ExpiresAt)
	return err
}

// Удаляет токен
func (r *RefreshTokenRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token)
	return err
}

// Проверяет наличие токена
func (r *RefreshTokenRepository) Exists(ctx context.Context, token string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM refresh_tokens WHERE token = $1)
	`, token).Scan(&exists)
	return exists, err
}

// Возвращает userID по токену
func (r *RefreshTokenRepository) GetUserIDByToken(ctx context.Context, token string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id FROM refresh_tokens WHERE token = $1
	`, token).Scan(&userID)
	return userID, err
}

// Возвращает всю структуру токена
func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, created_at, expires_at
		FROM refresh_tokens WHERE token = $1
	`, token).Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.CreatedAt, &rt.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

package repositories

import (
	"context"
	"database/sql"
	"mwork_backend/internal/models"
)

type ResponseRepository struct {
	db *sql.DB
}

func NewResponseRepository(db *sql.DB) *ResponseRepository {
	return &ResponseRepository{db: db}
}

func (r *ResponseRepository) Create(ctx context.Context, res *models.CastingResponse) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO casting_responses (casting_id, model_id, message, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, res.CastingID, res.ModelID, res.Message, res.Status).Scan(
		&res.ID, &res.CreatedAt,
	)
}

func (r *ResponseRepository) GetByID(ctx context.Context, id string) (*models.CastingResponse, error) {
	var res models.CastingResponse
	err := r.db.QueryRowContext(ctx, `
		SELECT id, casting_id, model_id, message, status, created_at
		FROM casting_responses WHERE id = $1
	`, id).Scan(
		&res.ID, &res.CastingID, &res.ModelID, &res.Message, &res.Status, &res.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *ResponseRepository) ListByCasting(ctx context.Context, castingID string) ([]models.CastingResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, casting_id, model_id, message, status, created_at
		FROM casting_responses WHERE casting_id = $1
	`, castingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []models.CastingResponse
	for rows.Next() {
		var res models.CastingResponse
		err := rows.Scan(&res.ID, &res.CastingID, &res.ModelID, &res.Message, &res.Status, &res.CreatedAt)
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}
	return responses, nil
}

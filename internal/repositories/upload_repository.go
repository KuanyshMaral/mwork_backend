package repositories

import (
	"context"
	"database/sql"
	"mwork_front_fn/internal/models"
)

type UploadRepository interface {
	Save(ctx context.Context, upload *models.Upload) error
	FindByID(ctx context.Context, id string) (*models.Upload, error)
	FindByEntity(ctx context.Context, entityType, entityID string) ([]models.Upload, error)
	DeleteByID(ctx context.Context, id string) error
}

type uploadRepository struct {
	db *sql.DB
}

func NewUploadRepository(db *sql.DB) UploadRepository {
	return &uploadRepository{db: db}
}

func (r *uploadRepository) Save(ctx context.Context, u *models.Upload) error {
	query := `
        INSERT INTO uploads (
            id, user_id, entity_type, entity_id, file_type,
            usage, path, mime_type, size, is_public, created_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `
	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.UserID, u.EntityType, u.EntityID, u.FileType,
		u.Usage, u.Path, u.MimeType, u.Size, u.IsPublic, u.CreatedAt,
	)
	return err
}

func (r *uploadRepository) FindByID(ctx context.Context, id string) (*models.Upload, error) {
	query := `SELECT id, user_id, entity_type, entity_id, file_type, usage, path, mime_type, size, is_public, created_at
              FROM uploads WHERE id = $1 LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, id)

	var u models.Upload
	err := row.Scan(
		&u.ID, &u.UserID, &u.EntityType, &u.EntityID, &u.FileType,
		&u.Usage, &u.Path, &u.MimeType, &u.Size, &u.IsPublic, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *uploadRepository) FindByEntity(ctx context.Context, entityType, entityID string) ([]models.Upload, error) {
	query := `SELECT id, user_id, entity_type, entity_id, file_type, usage, path, mime_type, size, is_public, created_at
              FROM uploads WHERE entity_type = $1 AND entity_id = $2`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uploads []models.Upload
	for rows.Next() {
		var u models.Upload
		if err := rows.Scan(
			&u.ID, &u.UserID, &u.EntityType, &u.EntityID, &u.FileType,
			&u.Usage, &u.Path, &u.MimeType, &u.Size, &u.IsPublic, &u.CreatedAt,
		); err != nil {
			return nil, err
		}
		uploads = append(uploads, u)
	}

	return uploads, nil
}

func (r *uploadRepository) DeleteByID(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM uploads WHERE id = $1`, id)
	return err
}

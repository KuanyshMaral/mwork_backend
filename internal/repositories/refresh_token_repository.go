package repositories

import (
	"errors"
	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

var (
	// ErrRefreshTokenNotFound возвращается, когда refresh-токен не найден в БД
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// RefreshTokenRepository определяет интерфейс для операций с refresh-токенами
type RefreshTokenRepository interface {
	// Create cоздает новую запись о refresh-токене
	Create(token *models.RefreshToken) error
	// FindByToken находит refresh-токен по его строковому значению
	FindByToken(tokenString string) (*models.RefreshToken, error)
	// DeleteByToken удаляет refresh-токен по его строковому значению
	DeleteByToken(tokenString string) error
	// DeleteByUserID удаляет все refresh-токены, связанные с пользователем
	DeleteByUserID(userID string) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository создает новый экземпляр RefreshTokenRepository
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// Create cоздает новую запись о refresh-токене
func (r *refreshTokenRepository) Create(token *models.RefreshToken) error {
	return r.db.Create(token).Error
}

// FindByToken находит refresh-токен по его строковому значению
func (r *refreshTokenRepository) FindByToken(tokenString string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.Where("token = ?", tokenString).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// DeleteByToken удаляет refresh-токен по его строковому значению
func (r *refreshTokenRepository) DeleteByToken(tokenString string) error {
	// Не возвращаем ошибку, если токен не найден, т.к. цель - его отсутствие
	return r.db.Where("token = ?", tokenString).Delete(&models.RefreshToken{}).Error
}

// DeleteByUserID удаляет все refresh-токены, связанные с пользователем
func (r *refreshTokenRepository) DeleteByUserID(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

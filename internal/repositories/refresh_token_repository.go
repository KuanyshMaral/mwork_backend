package repositories

import (
	"errors"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

var (
	// ErrRefreshTokenNotFound возвращается, когда refresh-токен не найден в БД
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// RefreshTokenRepository определяет интерфейс для операций с refresh-токенами
type RefreshTokenRepository interface {
	// Create создает новую запись о refresh-токене
	Create(db *gorm.DB, token *models.RefreshToken) error

	// FindByToken находит refresh-токен по его строковому значению
	FindByToken(db *gorm.DB, tokenString string) (*models.RefreshToken, error)

	// DeleteByToken удаляет refresh-токен по его строковому значению
	DeleteByToken(db *gorm.DB, tokenString string) error

	// DeleteByUserID удаляет все refresh-токены, связанные с пользователем
	DeleteByUserID(db *gorm.DB, userID string) error

	// CleanExpiredRefreshTokens удаляет все истекшие токены
	CleanExpiredRefreshTokens(db *gorm.DB) error

	// CountByUserID возвращает количество активных токенов пользователя
	CountByUserID(db *gorm.DB, userID string) (int64, error)

	// FindByUserID находит все токены пользователя (для администрирования)
	FindByUserID(db *gorm.DB, userID string) ([]models.RefreshToken, error)
}

type refreshTokenRepository struct {
	// ✅ Пустая структура - db больше не хранится здесь
}

// NewRefreshTokenRepository создает новый экземпляр RefreshTokenRepository
func NewRefreshTokenRepository() RefreshTokenRepository {
	return &refreshTokenRepository{}
}

// Create создает новую запись о refresh-токене
func (r *refreshTokenRepository) Create(db *gorm.DB, token *models.RefreshToken) error {
	return db.Create(token).Error
}

// FindByToken находит refresh-токен по его строковому значению
func (r *refreshTokenRepository) FindByToken(db *gorm.DB, tokenString string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := db.Where("token = ?", tokenString).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// DeleteByToken удаляет refresh-токен по его строковому значению
func (r *refreshTokenRepository) DeleteByToken(db *gorm.DB, tokenString string) error {
	result := db.Where("token = ?", tokenString).Delete(&models.RefreshToken{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Возвращаем ошибку, чтобы сервис мог ее обработать
		return ErrRefreshTokenNotFound
	}
	return nil
}

// DeleteByUserID удаляет все refresh-токены, связанные с пользователем
func (r *refreshTokenRepository) DeleteByUserID(db *gorm.DB, userID string) error {
	return db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

// CleanExpiredRefreshTokens удаляет все истекшие токены
func (r *refreshTokenRepository) CleanExpiredRefreshTokens(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error
}

// CountByUserID возвращает количество активных токенов пользователя
func (r *refreshTokenRepository) CountByUserID(db *gorm.DB, userID string) (int64, error) {
	var count int64
	err := db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Count(&count).Error
	return count, err
}

// FindByUserID находит все токены пользователя (для администрирования)
func (r *refreshTokenRepository) FindByUserID(db *gorm.DB, userID string) ([]models.RefreshToken, error) {
	var tokens []models.RefreshToken
	err := db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}

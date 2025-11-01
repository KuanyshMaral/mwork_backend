package repositories

import (
	"errors"
	"mwork_backend/internal/models"
	"time"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type UserRepository interface {
	// User operations
	FindByID(db *gorm.DB, id string) (*models.User, error)
	FindByEmail(db *gorm.DB, email string) (*models.User, error)
	Create(db *gorm.DB, user *models.User) error
	Update(db *gorm.DB, user *models.User) error
	UpdateStatus(db *gorm.DB, userID string, status models.UserStatus) error
	VerifyUser(db *gorm.DB, userID string) error
	Delete(db *gorm.DB, userID string) error
	FindByRole(db *gorm.DB, role models.UserRole, limit, offset int) ([]models.User, error)
	CountByRole(db *gorm.DB, role models.UserRole) (int64, error)

	// Admin operations
	FindAll(db *gorm.DB, limit, offset int) ([]models.User, error)
	CountAll(db *gorm.DB) (int64, error)
	FindWithFilter(db *gorm.DB, criteria UserFilter) ([]models.User, int64, error)

	FindByVerificationToken(db *gorm.DB, token string) (*models.User, error)
	FindByResetToken(db *gorm.DB, token string) (*models.User, error)
	UpdateLastActive(db *gorm.DB, userID string) error
}

type UserRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
}

type UserFilter struct {
	Role       models.UserRole
	Status     models.UserStatus
	IsVerified *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	Search     string
	Page       int
	PageSize   int
}

// ✅ Конструктор не принимает db
func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{}
}

// User operations

func (r *UserRepositoryImpl) FindByID(db *gorm.DB, id string) (*models.User, error) {
	var user models.User
	// ✅ Используем 'db' из параметра
	err := db.Preload("ModelProfile").Preload("EmployerProfile").Preload("Subscription").
		First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) FindByEmail(db *gorm.DB, email string) (*models.User, error) {
	var user models.User
	// ✅ Используем 'db' из параметра
	err := db.Preload("ModelProfile").Preload("EmployerProfile").
		First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) Create(db *gorm.DB, user *models.User) error {
	// Check if user already exists
	var existing models.User
	// ✅ Используем 'db' из параметра
	if err := db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
		return ErrUserAlreadyExists
	}

	// ✅ Используем 'db' из параметра
	return db.Create(user).Error
}

func (r *UserRepositoryImpl) Update(db *gorm.DB, user *models.User) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(user).Updates(map[string]interface{}{
		"email":              user.Email,
		"role":               user.Role,
		"status":             user.Status,
		"is_verified":        user.IsVerified,
		"verification_token": user.VerificationToken,
		"reset_token":        user.ResetToken,
		"reset_token_exp":    user.ResetTokenExp,
		"updated_at":         time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepositoryImpl) UpdateStatus(db *gorm.DB, userID string, status models.UserStatus) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepositoryImpl) VerifyUser(db *gorm.DB, userID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_verified":        true,
		"verification_token": "",
		"updated_at":         time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepositoryImpl) Delete(db *gorm.DB, userID string) error {
	// ✅ Вложенная транзакция удалена.
	// Мы используем переданный 'db', который может быть (или не быть) транзакцией.
	// Логика удаления связанных refresh-токенов удалена,
	// т.к. она относится к RefreshTokenRepository и должна быть в слое сервиса.

	// Delete user
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", userID).Delete(&models.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepositoryImpl) FindByRole(db *gorm.DB, role models.UserRole, limit, offset int) ([]models.User, error) {
	var users []models.User
	// ✅ Используем 'db' из параметра
	err := db.Where("role = ?", role).Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

func (r *UserRepositoryImpl) CountByRole(db *gorm.DB, role models.UserRole) (int64, error) {
	var count int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.User{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

// ❌❌❌ Удален раздел RefreshToken operations ❌❌❌
// (CreateRefreshToken, FindRefreshToken, DeleteRefreshToken, DeleteUserRefreshTokens, CleanExpiredRefreshTokens)
// Они перенесены в refresh_token_repository.go

// Admin operations

func (r *UserRepositoryImpl) FindAll(db *gorm.DB, limit, offset int) ([]models.User, error) {
	var users []models.User
	// ✅ Используем 'db' из параметра
	err := db.Preload("ModelProfile").Preload("EmployerProfile").
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

func (r *UserRepositoryImpl) CountAll(db *gorm.DB) (int64, error) {
	var count int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *UserRepositoryImpl) FindWithFilter(db *gorm.DB, criteria UserFilter) ([]models.User, int64, error) {
	var users []models.User
	// ✅ Используем 'db' из параметра
	query := db.Model(&models.User{})

	// Apply filters
	if criteria.Role != "" {
		query = query.Where("role = ?", criteria.Role)
	}
	if criteria.Status != "" {
		query = query.Where("status = ?", criteria.Status)
	}
	if criteria.IsVerified != nil {
		query = query.Where("is_verified = ?", *criteria.IsVerified)
	}
	if criteria.DateFrom != nil {
		query = query.Where("created_at >= ?", criteria.DateFrom)
	}
	if criteria.DateTo != nil {
		query = query.Where("created_at <= ?", criteria.DateTo)
	}
	if criteria.Search != "" {
		search := "%" + criteria.Search + "%"
		query = query.Where("email ILIKE ?", search)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and get results
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Preload("ModelProfile").Preload("EmployerProfile").
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error

	return users, total, err
}

// ❌❌❌ Удален раздел Analytics operations ❌❌❌
// (GetRegistrationStats, GetUserStats, GetActiveUsersCount, GetUserDistributionByCity)
// Они перенесены в analytics_repository.go

func (r *UserRepositoryImpl) FindByVerificationToken(db *gorm.DB, token string) (*models.User, error) {
	var user models.User
	// ✅ Используем 'db' из параметра
	err := db.Where("verification_token = ? AND verification_token != ''", token).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByResetToken находит пользователя по токену сброса пароля
func (r *UserRepositoryImpl) FindByResetToken(db *gorm.DB, token string) (*models.User, error) {
	var user models.User
	// ✅ Используем 'db' из параметра
	err := db.Where("reset_token = ? AND reset_token_exp > ?", token, time.Now()).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateLastActive обновляет время последней активности пользователя
func (r *UserRepositoryImpl) UpdateLastActive(db *gorm.DB, userID string) error {
	// ✅ Используем 'db' из параметра
	// Проверяем наличие колонки
	if !db.Migrator().HasColumn(&models.User{}, "last_active_at") {
		// Если колонки нет, ничего не делаем
		return nil
	}

	// ✅ Используем 'db' из параметра
	result := db.Model(&models.User{}).Where("id = ?", userID).Update("last_active_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

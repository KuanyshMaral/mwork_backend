package services

import (
	"errors"
	"math"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
	"time" // 👈 Добавлен импорт для GetRegistrationStats

	"gorm.io/gorm"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type UserService interface {
	GetProfile(db *gorm.DB, userID string) (*dto.UserResponse, error)
	UpdateProfile(db *gorm.DB, userID string, req *dto.UpdateProfileRequest) error
	GetUsers(db *gorm.DB, filter dto.AdminUserFilter) ([]*dto.UserResponse, int64, error)
	UpdateUserStatus(db *gorm.DB, adminID, userID string, status models.UserStatus) error
	VerifyEmployer(db *gorm.DB, adminID, employerID string) error
	GetRegistrationStats(db *gorm.DB, days int) (interface{}, error)
	// ❗️ ДОБАВЛЕН МЕТОД УДАЛЕНИЯ ПОЛЬЗОВАТЕЛЯ
	DeleteUser(db *gorm.DB, adminID, userID string) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type UserServiceImpl struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	userRepo    repositories.UserRepository
	profileRepo repositories.ProfileRepository
	// ❗️ Тебе, вероятно, понадобится репозиторий аналитики, который ты выделил
	// analyticsRepo repositories.AnalyticsRepository
}

// Конструктор больше не принимает 'db'
func NewUserService(
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	// analyticsRepo repositories.AnalyticsRepository, // 👈 Добавь сюда репо аналитики
) UserService {
	return &UserServiceImpl{
		// ❌ 'db: db,' УДАЛЕНО
		userRepo:    userRepo,
		profileRepo: profileRepo,
		// analyticsRepo: analyticsRepo, // 👈 И сюда
	}
}

// =======================
// Profile operations
// =======================

// GetProfile - 'db' добавлен
func (s *UserServiceImpl) GetProfile(db *gorm.DB, userID string) (*dto.UserResponse, error) {
	// ✅ Используем 'db' из параметра
	user, err := s.userRepo.FindByID(db, userID)
	if err != nil {
		return nil, handleRepositoryError(err)
	}

	// ✅ Передаем 'db' в хелпер
	return s.buildUserResponse(db, user)
}

// UpdateProfile - 'db' добавлен
func (s *UserServiceImpl) UpdateProfile(db *gorm.DB, userID string, req *dto.UpdateProfileRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db' (Unit of Work)
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback() // Rollback в случае паники или ошибки

	// ✅ Передаем 'tx'
	user, err := s.userRepo.FindByID(tx, userID)
	if err != nil {
		return handleRepositoryError(err)
	}

	if user.Role == models.UserRoleModel {
		// ✅ Передаем 'tx'
		profile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
		if err != nil {
			return handleRepositoryError(err)
		}
		updateModelProfile(profile, req) // хелпер не трогает db
		// ✅ Передаем 'tx'
		if err := s.profileRepo.UpdateModelProfile(tx, profile); err != nil {
			return apperrors.InternalError(err)
		}

	} else if user.Role == models.UserRoleEmployer {
		// ✅ Передаем 'tx'
		profile, err := s.profileRepo.FindEmployerProfileByUserID(tx, userID)
		if err != nil {
			return handleRepositoryError(err)
		}
		updateEmployerProfile(profile, req) // хелпер не трогает db
		// ✅ Передаем 'tx'
		if err := s.profileRepo.UpdateEmployerProfile(tx, profile); err != nil {
			return apperrors.InternalError(err)
		}
	} else {
		return apperrors.ErrInvalidUserRole
	}

	// ✅ Коммитим транзакцию (Unit of Work)
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}
	return nil
}

// =======================
// Admin operations
// =======================

// GetUsers - 'db' добавлен
func (s *UserServiceImpl) GetUsers(db *gorm.DB, filter dto.AdminUserFilter) ([]*dto.UserResponse, int64, error) {
	repoFilter := repositories.UserFilter{
		Role:       filter.Role,
		Status:     filter.Status,
		IsVerified: filter.IsVerified,
		DateFrom:   filter.DateFrom,
		DateTo:     filter.DateTo,
		Search:     filter.Search,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}

	// ✅ Используем 'db' из параметра
	users, total, err := s.userRepo.FindWithFilter(db, repoFilter)
	if err != nil {
		return nil, 0, apperrors.InternalError(err)
	}

	var userResponses []*dto.UserResponse
	for i := range users {
		// ✅ Передаем 'db'
		userResponse, err := s.buildUserResponse(db, &users[i])
		if err != nil {
			// Логгируем ошибку, но продолжаем
			continue
		}
		userResponses = append(userResponses, userResponse)
	}

	return userResponses, total, nil
}

// UpdateUserStatus - 'db' добавлен
func (s *UserServiceImpl) UpdateUserStatus(db *gorm.DB, adminID, userID string, status models.UserStatus) error {
	if adminID == userID {
		return apperrors.ErrCannotModifySelf
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем 'tx'
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleRepositoryError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	// ✅ Передаем 'tx'
	if err := s.userRepo.UpdateStatus(tx, userID, status); err != nil {
		return handleRepositoryError(err)
	}

	// ✅ Коммитим
	return tx.Commit().Error
}

// VerifyEmployer - 'db' добавлен
func (s *UserServiceImpl) VerifyEmployer(db *gorm.DB, adminID, employerID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем 'tx'
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleRepositoryError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	// ✅ Передаем 'tx'
	if err := s.profileRepo.VerifyEmployerProfile(tx, employerID); err != nil {
		return handleRepositoryError(err)
	}

	// ✅ Коммитим
	return tx.Commit().Error
}

// GetRegistrationStats - 'db' добавлен
func (s *UserServiceImpl) GetRegistrationStats(db *gorm.DB, days int) (interface{}, error) {
	// ✅ Используем 'db' из параметра
	// ❗️ Здесь тебе нужно будет вызвать твой 'analyticsRepo', который ты выделил
	// return s.analyticsRepo.GetRegistrationStats(db, days)

	// Временная заглушка, пока нет 'analyticsRepo'
	// (В твоем старом репо этот метод был, так что логика у тебя есть)
	// Этот код - просто пример
	dateFrom := time.Now().AddDate(0, 0, -days)
	var stats []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	err := db.Model(&models.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", dateFrom).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&stats).Error

	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return stats, nil
}

// DeleteUser - 'db' добавлен
func (s *UserServiceImpl) DeleteUser(db *gorm.DB, adminID, userID string) error {
	if adminID == userID {
		return apperrors.ErrCannotModifySelf // Администратор не может удалить сам себя
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// 1. Проверяем, что запрос исходит от Администратора
	// ✅ Передаем 'tx'
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleRepositoryError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	// 2. Удаляем пользователя
	// Логика полного удаления (например, токенов) должна быть здесь,
	// но пока просто вызываем репозиторий.
	// ✅ Передаем 'tx'
	if err := s.userRepo.Delete(tx, userID); err != nil {
		return handleRepositoryError(err)
	}

	// 3. Коммитим
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}
	return nil
}

// =======================
// Helper methods
// =======================

// buildUserResponse - 'db' добавлен
func (s *UserServiceImpl) buildUserResponse(db *gorm.DB, user *models.User) (*dto.UserResponse, error) {
	userResponse := &dto.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Status:     user.Status,
		IsVerified: user.IsVerified,
	}

	if user.Role == models.UserRoleModel {
		if user.ModelProfile != nil {
			userResponse.Profile = user.ModelProfile
		} else {
			// ✅ Передаем 'db'
			profile, err := s.profileRepo.FindModelProfileByUserID(db, user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	} else if user.Role == models.UserRoleEmployer {
		if user.EmployerProfile != nil {
			userResponse.Profile = user.EmployerProfile
		} else {
			// ✅ Передаем 'db'
			profile, err := s.profileRepo.FindEmployerProfileByUserID(db, user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	}

	return userResponse, nil
}

// Хелперы 'updateModelProfile' и 'updateEmployerProfile' не меняются,
// так как они не взаимодействуют с базой данных.
func updateModelProfile(profile *models.ModelProfile, req *dto.UpdateProfileRequest) {
	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.Age != nil {
		profile.Age = *req.Age
	}
	if req.Height != nil {
		profile.Height = float64(math.Round(*req.Height))
	}
	if req.Weight != nil {
		profile.Weight = float64(math.Round(*req.Weight))
	}
	if req.Gender != nil {
		profile.Gender = *req.Gender
	}
	if req.Experience != nil {
		profile.Experience = *req.Experience
	}
	if req.HourlyRate != nil {
		profile.HourlyRate = *req.HourlyRate
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	if req.ClothingSize != nil {
		profile.ClothingSize = *req.ClothingSize
	}
	if req.ShoeSize != nil {
		profile.ShoeSize = *req.ShoeSize
	}
	if req.BarterAccepted != nil {
		profile.BarterAccepted = *req.BarterAccepted
	}
	if req.IsPublic != nil {
		profile.IsPublic = *req.IsPublic
	}
	if req.Languages != nil {
		profile.SetLanguages(req.Languages)
	}
	if req.Categories != nil {
		profile.SetCategories(req.Categories)
	}
}

func updateEmployerProfile(profile *models.EmployerProfile, req *dto.UpdateProfileRequest) {
	if req.CompanyName != nil {
		profile.CompanyName = *req.CompanyName
	}
	if req.ContactPerson != nil {
		profile.ContactPerson = *req.ContactPerson
	}
	if req.Phone != nil {
		profile.Phone = *req.Phone
	}
	if req.Website != nil {
		profile.Website = *req.Website
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.CompanyType != nil {
		profile.CompanyType = *req.CompanyType
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
}

// handleRepositoryError не меняется
func handleRepositoryError(err error) error {
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserAlreadyExists) {
		return apperrors.ErrAlreadyExists(err)
	}
	return apperrors.InternalError(err)
}

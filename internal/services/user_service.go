package services

import (
	"strconv"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

type UserService interface {
	GetProfile(userID string) (*dto.UserResponse, error)
	UpdateProfile(userID string, req *dto.UpdateProfileRequestUser) error
	GetUsers(filter dto.AdminUserFilter) ([]*dto.UserResponse, int64, error)
	UpdateUserStatus(adminID, userID string, status models.UserStatus) error
	VerifyEmployer(adminID, employerID string) error
	GetRegistrationStats(days int) (*repositories.RegistrationStats, error)
}

type UserServiceImpl struct {
	userRepo    repositories.UserRepository
	profileRepo repositories.ProfileRepository
}

func NewUserService(
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
) UserService {
	return &UserServiceImpl{
		userRepo:    userRepo,
		profileRepo: profileRepo,
	}
}

// =======================
// Profile operations
// =======================

// GetProfile возвращает профиль пользователя
func (s *UserServiceImpl) GetProfile(userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}

	return s.buildUserResponse(user)
}

// UpdateProfile обновляет профиль пользователя
func (s *UserServiceImpl) UpdateProfile(userID string, req *dto.UpdateProfileRequestUser) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if user.Role == models.UserRoleModel {
		profile, err := s.profileRepo.FindModelProfileByUserID(userID)
		if err != nil {
			return appErrors.InternalError(err)
		}

		updateModelProfile(profile, req)
		return s.profileRepo.UpdateModelProfile(profile)
	} else if user.Role == models.UserRoleEmployer {
		profile, err := s.profileRepo.FindEmployerProfileByUserID(userID)
		if err != nil {
			return appErrors.InternalError(err)
		}

		updateEmployerProfile(profile, req)
		return s.profileRepo.UpdateEmployerProfile(profile)
	}

	return appErrors.ErrInvalidUserRole
}

// =======================
// Admin operations
// =======================

// GetUsers возвращает список пользователей с фильтрацией (для админов)
func (s *UserServiceImpl) GetUsers(filter dto.AdminUserFilter) ([]*dto.UserResponse, int64, error) {
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

	users, total, err := s.userRepo.FindWithFilter(repoFilter)
	if err != nil {
		return nil, 0, appErrors.InternalError(err)
	}

	var userResponses []*dto.UserResponse
	for i := range users {
		userResponse, err := s.buildUserResponse(&users[i])
		if err != nil {
			continue
		}
		userResponses = append(userResponses, userResponse)
	}

	return userResponses, total, nil
}

// UpdateUserStatus обновляет статус пользователя (админ-функция)
func (s *UserServiceImpl) UpdateUserStatus(adminID, userID string, status models.UserStatus) error {
	if adminID == userID {
		return appErrors.ErrCannotModifySelf
	}

	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	return s.userRepo.UpdateStatus(userID, status)
}

// VerifyEmployer верифицирует работодателя (админ-функция)
func (s *UserServiceImpl) VerifyEmployer(adminID, employerID string) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return appErrors.InternalError(err)
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	return s.profileRepo.VerifyEmployerProfile(employerID)
}

// GetRegistrationStats возвращает статистику регистраций (админ-функция)
func (s *UserServiceImpl) GetRegistrationStats(days int) (*repositories.RegistrationStats, error) {
	stats, err := s.userRepo.GetRegistrationStats(days)
	if err != nil {
		return nil, appErrors.InternalError(err)
	}
	return stats, nil
}

// =======================
// Helper methods
// =======================

// buildUserResponse строит ответ с данными пользователя и профилем
func (s *UserServiceImpl) buildUserResponse(user *models.User) (*dto.UserResponse, error) {
	userResponse := &dto.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Role:       user.Role,
		Status:     user.Status,
		IsVerified: user.IsVerified,
	}

	// Загрузка профиля
	if user.Role == models.UserRoleModel {
		if user.ModelProfile != nil {
			userResponse.Profile = user.ModelProfile
		} else {
			profile, err := s.profileRepo.FindModelProfileByUserID(user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	} else if user.Role == models.UserRoleEmployer {
		if user.EmployerProfile != nil {
			userResponse.Profile = user.EmployerProfile
		} else {
			profile, err := s.profileRepo.FindEmployerProfileByUserID(user.ID)
			if err == nil {
				userResponse.Profile = profile
			}
		}
	}

	return userResponse, nil
}

// updateFieldWithConversion обновляет поле с конвертацией типов
func updateFieldWithConversion(dst interface{}, src interface{}) {
	if src == nil {
		return
	}
	switch d := dst.(type) {
	case *string:
		*d = *(src.(*string))
	case *int:
		val, _ := strconv.Atoi(*(src.(*string))) // конвертация string -> int
		*d = val
	case *float64:
		*d = *(src.(*float64))
	case *bool:
		*d = *(src.(*bool))
	}
}

// updateModelProfile обновляет поля профиля модели
func updateModelProfile(profile *models.ModelProfile, req *dto.UpdateProfileRequestUser) {
	updateFieldWithConversion(&profile.Name, req.Name)
	updateFieldWithConversion(&profile.City, req.City)
	updateFieldWithConversion(&profile.Age, req.Age)
	updateFieldWithConversion(&profile.Height, req.Height)
	updateFieldWithConversion(&profile.Weight, req.Weight)
	updateFieldWithConversion(&profile.Gender, req.Gender)
	updateFieldWithConversion(&profile.Experience, req.Experience)
	updateFieldWithConversion(&profile.HourlyRate, req.HourlyRate)
	updateFieldWithConversion(&profile.Description, req.Description)
	updateFieldWithConversion(&profile.ClothingSize, req.ClothingSize)
	updateFieldWithConversion(&profile.ShoeSize, req.ShoeSize)
	updateFieldWithConversion(&profile.BarterAccepted, req.BarterAccepted)
	updateFieldWithConversion(&profile.IsPublic, req.IsPublic)
}

// updateEmployerProfile обновляет поля профиля работодателя
func updateEmployerProfile(profile *models.EmployerProfile, req *dto.UpdateProfileRequestUser) {
	updateFieldWithConversion(&profile.CompanyName, req.CompanyName)
	updateFieldWithConversion(&profile.ContactPerson, req.ContactPerson)
	updateFieldWithConversion(&profile.Phone, req.Phone)
	updateFieldWithConversion(&profile.Website, req.Website)
	updateFieldWithConversion(&profile.City, req.City)
	updateFieldWithConversion(&profile.CompanyType, req.CompanyType)
	updateFieldWithConversion(&profile.Description, req.Description)
}

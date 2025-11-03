package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"

	"gorm.io/datatypes"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type ProfileService interface {
	CreateModelProfile(db *gorm.DB, req *dto.CreateModelProfileRequest) error
	CreateEmployerProfile(db *gorm.DB, req *dto.CreateEmployerProfileRequest) error
	GetProfile(db *gorm.DB, userID, requesterID string) (*dto.ProfileResponse, error)
	UpdateProfile(db *gorm.DB, userID string, req *dto.UpdateProfileRequest) error
	SearchModels(db *gorm.DB, criteria *dto.SearchModelsRequest) (*dto.PaginatedResponse, error)
	SearchEmployers(db *gorm.DB, criteria *dto.SearchEmployersRequest) (*dto.PaginatedResponse, error)
	GetModelStats(db *gorm.DB, modelID string) (*dto.ModelProfileStats, error)
	ToggleProfileVisibility(db *gorm.DB, userID string, isPublic bool) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type ProfileServiceImpl struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	profileRepo      repositories.ProfileRepository
	userRepo         repositories.UserRepository
	portfolioRepo    repositories.PortfolioRepository
	reviewRepo       repositories.ReviewRepository
	notificationRepo repositories.NotificationRepository
}

// ✅ Конструктор обновлен (db убран)
func NewProfileService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	profileRepo repositories.ProfileRepository,
	userRepo repositories.UserRepository,
	portfolioRepo repositories.PortfolioRepository,
	reviewRepo repositories.ReviewRepository,
	notificationRepo repositories.NotificationRepository,
) ProfileService {
	return &ProfileServiceImpl{
		// ❌ 'db: db,' УДАЛЕНО
		profileRepo:      profileRepo,
		userRepo:         userRepo,
		portfolioRepo:    portfolioRepo,
		reviewRepo:       reviewRepo,
		notificationRepo: notificationRepo,
	}
}

// ==========================
// Profile Creation
// ==========================
// CreateModelProfile - 'db' добавлен
func (s *ProfileServiceImpl) CreateModelProfile(db *gorm.DB, req *dto.CreateModelProfileRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, req.UserID)
	if err != nil {
		return handleProfileError(err)
	}
	if user.Role != models.UserRoleModel {
		return apperrors.ErrInvalidUserRole
	}

	if err := s.validateModelProfileData(req); err != nil {
		return err
	}

	languagesJSON, err := json.Marshal(req.Languages)
	if err != nil {
		return fmt.Errorf("failed to marshal languages: %w", err)
	}
	categoriesJSON, err := json.Marshal(req.Categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	profile := &models.ModelProfile{
		UserID:         req.UserID,
		Name:           req.Name,
		Age:            req.Age,
		Height:         float64(req.Height),
		Weight:         float64(req.Weight),
		Gender:         req.Gender,
		Experience:     req.Experience,
		HourlyRate:     req.HourlyRate,
		Description:    req.Description,
		ClothingSize:   req.ClothingSize,
		ShoeSize:       req.ShoeSize,
		City:           req.City,
		Languages:      datatypes.JSON(languagesJSON),
		Categories:     datatypes.JSON(categoriesJSON),
		BarterAccepted: req.BarterAccepted,
		IsPublic:       req.IsPublic,
	}

	// ✅ Передаем tx
	if err := s.profileRepo.CreateModelProfile(tx, profile); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// CreateEmployerProfile - 'db' добавлен
func (s *ProfileServiceImpl) CreateEmployerProfile(db *gorm.DB, req *dto.CreateEmployerProfileRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, req.UserID)
	if err != nil {
		return handleProfileError(err)
	}
	if user.Role != models.UserRoleEmployer {
		return apperrors.ErrInvalidUserRole
	}

	profile := &models.EmployerProfile{
		UserID:        req.UserID,
		CompanyName:   req.CompanyName,
		ContactPerson: req.ContactPerson,
		Phone:         req.Phone,
		Website:       req.Website,
		City:          req.City,
		CompanyType:   req.CompanyType,
		Description:   req.Description,
		IsVerified:    false,
	}

	// ✅ Передаем tx
	if err := s.profileRepo.CreateEmployerProfile(tx, profile); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ==========================
// Profile Retrieval
// ==========================
// GetProfile - 'db' добавлен
func (s *ProfileServiceImpl) GetProfile(db *gorm.DB, userID, requesterID string) (*dto.ProfileResponse, error) {
	// ✅ Используем 'db' из параметра
	user, err := s.userRepo.FindByID(db, userID)
	if err != nil {
		return nil, handleProfileError(err)
	}

	var profileData interface{}
	var profileType string
	var stats interface{}

	switch user.Role {
	case models.UserRoleModel:
		// ✅ Передаем db
		profile, err := s.profileRepo.FindModelProfileByUserID(db, userID)
		if err != nil {
			return nil, handleProfileError(err)
		}
		if !profile.IsPublic && requesterID != userID {
			return nil, apperrors.ErrProfileNotPublic
		}
		profileData = profile
		profileType = "model"

		// ✅ Передаем db
		if modelStats, err := s.profileRepo.GetModelStats(db, profile.ID); err == nil {
			stats = modelStats
		}
		if requesterID != userID {
			// ✅ Передаем 'db' (пул) в go рутину
			go s.profileRepo.IncrementModelProfileViews(db, profile.ID)
		}

	case models.UserRoleEmployer:
		// ✅ Передаем db
		profile, err := s.profileRepo.FindEmployerProfileByUserID(db, userID)
		if err != nil {
			return nil, handleProfileError(err)
		}
		profileData = profile
		profileType = "employer"

		// ✅ Передаем db
		if employerStats, err := s.getEmployerStats(db, profile.ID); err == nil {
			stats = employerStats
		}

	default:
		return nil, apperrors.ErrInvalidUserRole
	}

	return &dto.ProfileResponse{
		ID:        userID,
		Type:      profileType,
		UserID:    userID,
		Data:      profileData,
		Stats:     stats,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// ==========================
// Profile Update
// ==========================
// UpdateProfile - 'db' добавлен
func (s *ProfileServiceImpl) UpdateProfile(db *gorm.DB, userID string, req *dto.UpdateProfileRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, userID)
	if err != nil {
		return handleProfileError(err)
	}

	switch user.Role {
	case models.UserRoleModel:
		// ✅ Передаем tx
		if err := s.updateModelProfile(tx, userID, req); err != nil {
			return err
		}
	case models.UserRoleEmployer:
		// ✅ Передаем tx
		if err := s.updateEmployerProfile(tx, userID, req); err != nil {
			return err
		}
	default:
		return apperrors.ErrInvalidUserRole
	}
	return tx.Commit().Error
}

// updateModelProfile - 'db' добавлен
func (s *ProfileServiceImpl) updateModelProfile(db *gorm.DB, userID string, req *dto.UpdateProfileRequest) error {
	// ✅ Используем 'db' из параметра
	profile, err := s.profileRepo.FindModelProfileByUserID(db, userID)
	if err != nil {
		return handleProfileError(err)
	}

	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.Height != nil {
		profile.Height = float64(*req.Height)
	}
	if req.Weight != nil {
		profile.Weight = float64(*req.Weight)
	}
	// ... (другие поля)
	if req.Languages != nil {
		languagesJSON, err := json.Marshal(req.Languages)
		if err != nil {
			return fmt.Errorf("failed to marshal languages: %w", err)
		}
		profile.Languages = datatypes.JSON(languagesJSON)
	}
	if req.Categories != nil {
		categoriesJSON, err := json.Marshal(req.Categories)
		if err != nil {
			return fmt.Errorf("failed to marshal categories: %w", err)
		}
		profile.Categories = datatypes.JSON(categoriesJSON)
	}

	// ✅ Используем 'db' из параметра
	return s.profileRepo.UpdateModelProfile(db, profile)
}

// updateEmployerProfile - 'db' добавлен
func (s *ProfileServiceImpl) updateEmployerProfile(db *gorm.DB, userID string, req *dto.UpdateProfileRequest) error {
	// ✅ Используем 'db' из параметра
	profile, err := s.profileRepo.FindEmployerProfileByUserID(db, userID)
	if err != nil {
		return handleProfileError(err)
	}

	if req.CompanyName != nil {
		profile.CompanyName = *req.CompanyName
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	// ... (другие поля)

	// ✅ Используем 'db' из параметра
	return s.profileRepo.UpdateEmployerProfile(db, profile)
}

// ==========================
// Search and Discovery
// ==========================
// SearchModels - 'db' добавлен
func (s *ProfileServiceImpl) SearchModels(db *gorm.DB, criteria *dto.SearchModelsRequest) (*dto.PaginatedResponse, error) {
	searchCriteria := repositories.ModelSearchCriteria{
		Query:         criteria.Query,
		City:          criteria.City,
		Categories:    criteria.Categories,
		Gender:        criteria.Gender,
		MinAge:        criteria.MinAge,
		MaxAge:        criteria.MaxAge,
		MinHeight:     criteria.MinHeight,
		MaxHeight:     criteria.MaxHeight,
		MinWeight:     criteria.MinWeight,
		MaxWeight:     criteria.MaxWeight,
		MinPrice:      criteria.MinPrice,
		MaxPrice:      criteria.MaxPrice,
		MinExperience: criteria.MinExperience,
		Languages:     criteria.Languages,
		AcceptsBarter: criteria.AcceptsBarter,
		MinRating:     criteria.MinRating,
		IsPublic:      &[]bool{true}[0],
		Page:          criteria.Page,
		PageSize:      criteria.PageSize,
		SortBy:        criteria.SortBy,
		SortOrder:     criteria.SortOrder,
	}

	// ✅ Используем 'db' из параметра
	models, total, err := s.profileRepo.SearchModelProfiles(db, searchCriteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return buildPaginatedResponse(models, total, criteria.Page, criteria.PageSize), nil
}

// SearchEmployers - 'db' добавлен
func (s *ProfileServiceImpl) SearchEmployers(db *gorm.DB, criteria *dto.SearchEmployersRequest) (*dto.PaginatedResponse, error) {
	repoCriteria := repositories.EmployerSearchCriteria{
		Query:       criteria.Query,
		City:        criteria.City,
		CompanyType: criteria.CompanyType,
		IsVerified:  criteria.IsVerified,
		Page:        criteria.Page,
		PageSize:    criteria.PageSize,
	}

	// ✅ Используем 'db' из параметра
	employers, total, err := s.profileRepo.SearchEmployerProfiles(db, repoCriteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return buildPaginatedResponse(employers, total, criteria.Page, criteria.PageSize), nil
}

// ==========================
// Helper Methods
// ==========================
// (validateModelProfileData - чистая функция, без изменений)
func (s *ProfileServiceImpl) validateModelProfileData(req *dto.CreateModelProfileRequest) error {
	if req.Age < 16 || req.Age > 70 {
		return errors.New("age must be between 16 and 70")
	}
	if req.Height < 100 || req.Height > 250 {
		return errors.New("height must be between 100 and 250 cm")
	}
	if req.HourlyRate < 0 {
		return errors.New("hourly rate cannot be negative")
	}
	return nil
}

// getEmployerStats - 'db' добавлен
func (s *ProfileServiceImpl) getEmployerStats(db *gorm.DB, employerID string) (*dto.EmployerProfileStats, error) {
	// ✅ Используем 'db' из параметра
	// TODO: Реализовать s.profileRepo.GetEmployerStats(db, employerID)
	return &dto.EmployerProfileStats{
		TotalCastings:  0,
		ActiveCastings: 0,
		CompletedJobs:  0,
		TotalResponses: 0,
		AverageRating:  0.0,
	}, nil
}

// GetModelStats - 'db' добавлен
func (s *ProfileServiceImpl) GetModelStats(db *gorm.DB, modelID string) (*dto.ModelProfileStats, error) {
	// ✅ Используем 'db' из параметра
	stats, err := s.profileRepo.GetModelStats(db, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &dto.ModelProfileStats{
		TotalViews:      stats.TotalViews,
		AverageRating:   stats.AverageRating,
		TotalReviews:    stats.TotalReviews,
		PortfolioItems:  stats.PortfolioItems,
		ActiveResponses: stats.ActiveResponses,
		CompletedJobs:   stats.CompletedJobs,
	}, nil
}

// ToggleProfileVisibility - 'db' добавлен
func (s *ProfileServiceImpl) ToggleProfileVisibility(db *gorm.DB, userID string, isPublic bool) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	user, err := s.userRepo.FindByID(tx, userID)
	if err != nil {
		return handleProfileError(err)
	}
	if user.Role != models.UserRoleModel {
		return errors.New("only model profiles can toggle visibility")
	}

	// ✅ Передаем tx
	profile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil {
		return handleProfileError(err)
	}

	profile.IsPublic = isPublic
	// ✅ Передаем tx
	if err := s.profileRepo.UpdateModelProfile(tx, profile); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// (buildPaginatedResponse - чистая функция, без изменений)
func buildPaginatedResponse(data interface{}, total int64, page, pageSize int) *dto.PaginatedResponse {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = int(total / int64(pageSize))
		if total%int64(pageSize) != 0 {
			totalPages++
		}
	}
	return &dto.PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

// (handleProfileError - хелпер, без изменений)
func handleProfileError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrProfileNotFound) ||
		errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

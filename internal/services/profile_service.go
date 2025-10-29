package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"

	"gorm.io/datatypes"
)

// ProfileService - это интерфейс для профильных операций.
type ProfileService interface {
	CreateModelProfile(req *dto.CreateModelProfileRequest) error
	CreateEmployerProfile(req *dto.CreateEmployerProfileRequest) error
	GetProfile(userID, requesterID string) (*dto.ProfileResponse, error)
	UpdateProfile(userID string, req *dto.UpdateProfileRequest) error
	SearchModels(criteria dto.ProfileSearchCriteria) ([]*dto.ProfileResponse, int64, error)
	SearchEmployers(criteria repositories.EmployerSearchCriteria) ([]*dto.ProfileResponse, int64, error)
	GetModelStats(modelID string) (*dto.ModelProfileStats, error)
	ToggleProfileVisibility(userID string, isPublic bool) error
}

// ProfileServiceImpl - конкретная реализация интерфейса ProfileService.
// ПЕРЕИМЕНОВАНО: Было ProfileService, стало ProfileServiceImpl.
type ProfileServiceImpl struct {
	profileRepo      repositories.ProfileRepository
	userRepo         repositories.UserRepository
	portfolioRepo    repositories.PortfolioRepository
	reviewRepo       repositories.ReviewRepository
	notificationRepo repositories.NotificationRepository
}

// NewProfileService - конструктор, возвращающий тип ИНТЕРФЕЙСА.
func NewProfileService(
	profileRepo repositories.ProfileRepository,
	userRepo repositories.UserRepository,
	portfolioRepo repositories.PortfolioRepository,
	reviewRepo repositories.ReviewRepository,
	notificationRepo repositories.NotificationRepository,
) ProfileService { // <--- ИЗМЕНЕНО: теперь возвращает интерфейс ProfileService
	return &ProfileServiceImpl{ // Возвращает указатель на реализацию, который удовлетворяет интерфейсу
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
// Методы теперь привязаны к ProfileServiceImpl
func (s *ProfileServiceImpl) CreateModelProfile(req *dto.CreateModelProfileRequest) error {
	user, err := s.userRepo.FindByID(req.UserID)
	if err != nil {
		return err
	}
	if user.Role != models.UserRoleModel {
		return appErrors.ErrInvalidUserRole
	}
	// ... (остальной код метода)

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
		Height:         int(req.Height), // <--- ИСПРАВЛЕНО: Добавлено int()
		Weight:         int(req.Weight), // <--- ИСПРАВЛЕНО: Добавлено int()
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

	return s.profileRepo.CreateModelProfile(profile)
}

func (s *ProfileServiceImpl) CreateEmployerProfile(req *dto.CreateEmployerProfileRequest) error {
	user, err := s.userRepo.FindByID(req.UserID)
	if err != nil {
		return err
	}
	if user.Role != models.UserRoleEmployer {
		return appErrors.ErrInvalidUserRole
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

	return s.profileRepo.CreateEmployerProfile(profile)
}

// ==========================
// Profile Retrieval
// ==========================
func (s *ProfileServiceImpl) GetProfile(userID, requesterID string) (*dto.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	var profileData interface{}
	var profileType string
	var stats interface{}

	switch user.Role {
	case models.UserRoleModel:
		profile, err := s.profileRepo.FindModelProfileByUserID(userID)
		if err != nil {
			return nil, err
		}
		if !profile.IsPublic && requesterID != userID {
			return nil, appErrors.ErrProfileNotPublic
		}
		profileData = profile
		profileType = "model"

		if modelStats, err := s.profileRepo.GetModelStats(profile.ID); err == nil {
			stats = modelStats
		}
		if requesterID != userID {
			go s.profileRepo.IncrementModelProfileViews(profile.ID)
		}

	case models.UserRoleEmployer:
		profile, err := s.profileRepo.FindEmployerProfileByUserID(userID)
		if err != nil {
			return nil, err
		}
		profileData = profile
		profileType = "employer"

		if employerStats, err := s.getEmployerStats(profile.ID); err == nil {
			stats = employerStats
		}

	default:
		return nil, appErrors.ErrInvalidUserRole
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
func (s *ProfileServiceImpl) UpdateProfile(userID string, req *dto.UpdateProfileRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	switch user.Role {
	case models.UserRoleModel:
		return s.updateModelProfile(userID, req)
	case models.UserRoleEmployer:
		return s.updateEmployerProfile(userID, req)
	default:
		return appErrors.ErrInvalidUserRole
	}
}

func (s *ProfileServiceImpl) updateModelProfile(userID string, req *dto.UpdateProfileRequest) error {
	profile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil {
		return err
	}

	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.City != nil {
		profile.City = *req.City
	}
	if req.Description != nil {
		profile.Description = *req.Description
	}
	if req.Age != nil {
		profile.Age = *req.Age
	}
	if req.Height != nil {
		profile.Height = int(*req.Height) // <--- ИСПРАВЛЕНО: Добавлено int()
	}
	if req.Weight != nil {
		profile.Weight = int(*req.Weight) // <--- ИСПРАВЛЕНО: Добавлено int()
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

	return s.profileRepo.UpdateModelProfile(profile)
}

func (s *ProfileServiceImpl) updateEmployerProfile(userID string, req *dto.UpdateProfileRequest) error {
	profile, err := s.profileRepo.FindEmployerProfileByUserID(userID)
	if err != nil {
		return err
	}

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

	return s.profileRepo.UpdateEmployerProfile(profile)
}

// ==========================
// Search and Discovery
// ==========================
func (s *ProfileServiceImpl) SearchModels(criteria dto.ProfileSearchCriteria) ([]*dto.ProfileResponse, int64, error) {
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

	models, total, err := s.profileRepo.SearchModelProfiles(searchCriteria)
	if err != nil {
		return nil, 0, err
	}

	var responses []*dto.ProfileResponse
	for _, model := range models {
		responses = append(responses, &dto.ProfileResponse{
			ID:        model.ID,
			Type:      "model",
			UserID:    model.UserID,
			Data:      model,
			CreatedAt: model.CreatedAt,
			UpdatedAt: model.UpdatedAt,
		})
	}

	return responses, total, nil
}

func (s *ProfileServiceImpl) SearchEmployers(criteria repositories.EmployerSearchCriteria) ([]*dto.ProfileResponse, int64, error) {
	employers, total, err := s.profileRepo.SearchEmployerProfiles(criteria)
	if err != nil {
		return nil, 0, err
	}

	var responses []*dto.ProfileResponse
	for _, employer := range employers {
		responses = append(responses, &dto.ProfileResponse{
			ID:        employer.ID,
			Type:      "employer",
			UserID:    employer.UserID,
			Data:      employer,
			CreatedAt: employer.CreatedAt,
			UpdatedAt: employer.UpdatedAt,
		})
	}

	return responses, total, nil
}

// ==========================
// Helper Methods
// ==========================
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

func (s *ProfileServiceImpl) getEmployerStats(employerID string) (*dto.EmployerProfileStats, error) {
	return &dto.EmployerProfileStats{
		TotalCastings:  0,
		ActiveCastings: 0,
		CompletedJobs:  0,
		TotalResponses: 0,
		AverageRating:  0.0,
	}, nil
}

// Profile Analytics
func (s *ProfileServiceImpl) GetModelStats(modelID string) (*dto.ModelProfileStats, error) {
	stats, err := s.profileRepo.GetModelStats(modelID)
	if err != nil {
		return nil, err
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

// Profile Visibility Management
func (s *ProfileServiceImpl) ToggleProfileVisibility(userID string, isPublic bool) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	if user.Role != models.UserRoleModel {
		return errors.New("only model profiles can toggle visibility")
	}

	profile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil {
		return err
	}

	profile.IsPublic = isPublic
	return s.profileRepo.UpdateModelProfile(profile)
}

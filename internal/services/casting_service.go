package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
type CastingService interface {
	CreateCasting(db *gorm.DB, req *dto.CreateCastingRequest) error
	GetCasting(db *gorm.DB, castingID string, requesterID string) (*dto.CastingResponse, error)
	UpdateCasting(db *gorm.DB, castingID string, requesterID string, req *dto.UpdateCastingRequest) error
	PublishCasting(db *gorm.DB, castingID string, requesterID string) error
	CloseCasting(db *gorm.DB, castingID string, requesterID string) error
	DeleteCasting(db *gorm.DB, castingID string, requesterID string) error
	SearchCastings(db *gorm.DB, criteria dto.SearchCastingsRequest) ([]*dto.CastingResponse, int64, error)
	GetEmployerCastings(db *gorm.DB, employerID string, requesterID string) ([]*dto.CastingResponse, error)
	GetActiveCastings(db *gorm.DB, limit int) ([]*dto.CastingResponse, error)
	GetCastingsByCity(db *gorm.DB, city string, limit int) ([]*dto.CastingResponse, error)
	GetCastingStats(db *gorm.DB, employerID string, requesterID string) (*repositories.CastingStats, error)
	FindMatchingCastings(db *gorm.DB, modelID string, limit int) ([]*dto.CastingResponse, error)
	UpdateCastingStatus(db *gorm.DB, castingID string, requesterID string, status models.CastingStatus) error
	GetCastingStatsForCasting(db *gorm.DB, castingID string, requesterID string) (*dto.CastingStatsResponse, error)
	CloseExpiredCastings(db *gorm.DB) error
	// ▼▼▼ ДОБАВЛЕНЫ НЕДОСТАЮЩИЕ МЕТОДЫ (ADMIN) ▼▼▼
	GetPlatformCastingStats(db *gorm.DB, dateFrom time.Time, dateTo time.Time) (interface{}, error)
	GetMatchingStats(db *gorm.DB, dateFrom time.Time, dateTo time.Time) (interface{}, error)
	GetCastingDistributionByCity(db *gorm.DB) (interface{}, error)
	GetActiveCastingsCount(db *gorm.DB) (int64, error)
	GetPopularCategories(db *gorm.DB, limit int) (interface{}, error)
	// ▲▲▲ ДОБАВЛЕНЫ НЕДОСТАЮЩИЕ МЕТОДЫ (ADMIN) ▲▲▲
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type CastingServiceImpl struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	castingRepo      repositories.CastingRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	subscriptionRepo repositories.SubscriptionRepository
	notificationRepo repositories.NotificationRepository
	reviewRepo       repositories.ReviewRepository
	responseRepo     repositories.ResponseRepository
}

// ✅ Конструктор обновлен (db убран)
func NewCastingService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	castingRepo repositories.CastingRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	notificationRepo repositories.NotificationRepository,
	reviewRepo repositories.ReviewRepository,
	responseRepo repositories.ResponseRepository,
) CastingService {
	return &CastingServiceImpl{
		// ❌ 'db: db,' УДАЛЕНО
		castingRepo:      castingRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
		reviewRepo:       reviewRepo,
		responseRepo:     responseRepo,
	}
}

// Casting Operations

// CreateCasting - 'db' добавлен
func (s *CastingServiceImpl) CreateCasting(db *gorm.DB, req *dto.CreateCastingRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	employer, err := s.userRepo.FindByID(tx, req.EmployerID)
	if err != nil {
		return handleCastingError(err)
	}

	if employer.Role != models.UserRoleEmployer {
		return apperrors.ErrInsufficientPermissions
	}

	// ✅ Передаем tx
	canPublish, err := s.subscriptionRepo.CanUserPublish(tx, req.EmployerID)
	if err != nil {
		return apperrors.InternalError(err)
	}

	if !canPublish {
		return apperrors.ErrSubscriptionLimit
	}

	categoriesJSON, err := json.Marshal(req.Categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}
	languagesJSON, err := json.Marshal(req.Languages)
	if err != nil {
		return fmt.Errorf("failed to marshal languages: %w", err)
	}
	if req.PaymentMax < req.PaymentMin {
		return errors.New("maximum payment cannot be less than minimum payment")
	}
	if req.AgeMin != nil && req.AgeMax != nil && *req.AgeMin > *req.AgeMax {
		return errors.New("minimum age cannot be greater than maximum age")
	}

	casting := &models.Casting{
		EmployerID:      req.EmployerID,
		Title:           req.Title,
		Description:     req.Description,
		PaymentMin:      req.PaymentMin,
		PaymentMax:      req.PaymentMax,
		CastingDate:     &req.CastingDate,
		CastingTime:     &req.CastingTime,
		Address:         &req.Address,
		City:            req.City,
		Categories:      datatypes.JSON(categoriesJSON),
		Gender:          req.Gender,
		AgeMin:          req.AgeMin,
		AgeMax:          req.AgeMax,
		HeightMin:       req.HeightMin,
		HeightMax:       req.HeightMax,
		WeightMin:       req.WeightMin,
		WeightMax:       req.WeightMax,
		ClothingSize:    &req.ClothingSize,
		ShoeSize:        &req.ShoeSize,
		ExperienceLevel: &req.ExperienceLevel,
		Languages:       datatypes.JSON(languagesJSON),
		JobType:         req.JobType,
		Status:          models.CastingStatusDraft,
	}

	// ✅ Передаем tx
	if err = s.castingRepo.CreateCasting(tx, casting); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.IncrementSubscriptionUsage(tx, req.EmployerID, "publications"); err != nil {
		return apperrors.InternalError(err)
	}

	return tx.Commit().Error
}

// GetCasting - 'db' добавлен
func (s *CastingServiceImpl) GetCasting(db *gorm.DB, castingID string, requesterID string) (*dto.CastingResponse, error) {
	// ✅ Используем 'db' из параметра
	casting, err := s.castingRepo.FindCastingByID(db, castingID)
	if err != nil {
		return nil, handleCastingError(err)
	}

	if requesterID != casting.EmployerID {
		// ✅ Передаем 'db' (пул) в go рутину
		go s.castingRepo.IncrementCastingViews(db, castingID)
	}

	// ✅ Используем 'db' из параметра
	return s.buildCastingResponse(db, casting, requesterID == casting.EmployerID)
}

// UpdateCasting - 'db' добавлен
func (s *CastingServiceImpl) UpdateCasting(db *gorm.DB, castingID string, requesterID string, req *dto.UpdateCastingRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return handleCastingError(err)
	}

	if casting.EmployerID != requesterID {
		return apperrors.ErrInsufficientPermissions
	}
	if casting.Status != models.CastingStatusDraft {
		return apperrors.ErrInvalidCastingStatus
	}

	if req.Title != nil {
		casting.Title = *req.Title
	}
	// ... (другие поля)
	if req.Categories != nil {
		categoriesJSON, err := json.Marshal(req.Categories)
		if err != nil {
			return fmt.Errorf("failed to marshal categories: %w", err)
		}
		casting.Categories = datatypes.JSON(categoriesJSON)
	}
	if req.Languages != nil {
		languagesJSON, err := json.Marshal(req.Languages)
		if err != nil {
			return fmt.Errorf("failed to marshal languages: %w", err)
		}
		casting.Languages = datatypes.JSON(languagesJSON)
	}
	// ... (другие поля)

	// ✅ Передаем tx
	if err := s.castingRepo.UpdateCasting(tx, casting); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// PublishCasting - 'db' добавлен
func (s *CastingServiceImpl) PublishCasting(db *gorm.DB, castingID string, requesterID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return handleCastingError(err)
	}

	if casting.EmployerID != requesterID {
		return apperrors.ErrInsufficientPermissions
	}
	if casting.Status != models.CastingStatusDraft {
		return apperrors.ErrInvalidCastingStatus
	}

	casting.Status = models.CastingStatusActive

	// ✅ Передаем tx
	if err := s.castingRepo.UpdateCasting(tx, casting); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// CloseCasting - 'db' добавлен
func (s *CastingServiceImpl) CloseCasting(db *gorm.DB, castingID string, requesterID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return handleCastingError(err)
	}

	if casting.EmployerID != requesterID {
		return apperrors.ErrInsufficientPermissions
	}
	if casting.Status != models.CastingStatusActive {
		return apperrors.ErrInvalidCastingStatus
	}

	casting.Status = models.CastingStatusClosed
	// ✅ Передаем tx
	if err := s.castingRepo.UpdateCasting(tx, casting); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteCasting - 'db' добавлен
func (s *CastingServiceImpl) DeleteCasting(db *gorm.DB, castingID string, requesterID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return handleCastingError(err)
	}

	if casting.EmployerID != requesterID {
		return apperrors.ErrInsufficientPermissions
	}
	if casting.Status != models.CastingStatusDraft {
		return apperrors.ErrInvalidCastingStatus
	}

	// ✅ Передаем tx
	if err := s.castingRepo.DeleteCasting(tx, castingID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Search and Discovery

// SearchCastings - 'db' добавлен
func (s *CastingServiceImpl) SearchCastings(db *gorm.DB, criteria dto.SearchCastingsRequest) ([]*dto.CastingResponse, int64, error) {
	searchCriteria := repositories.CastingSearchCriteria{
		Query:      criteria.Query,
		City:       criteria.City,
		Categories: criteria.Categories,
		Gender:     criteria.Gender,
		MinAge:     criteria.MinAge,
		MaxAge:     criteria.MaxAge,
		MinSalary:  criteria.MinSalary,
		MaxSalary:  criteria.MaxSalary,
		JobType:    criteria.JobType,
		Status:     criteria.Status,
		EmployerID: criteria.EmployerID,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		SortBy:     criteria.SortBy,
		SortOrder:  criteria.SortOrder,
	}

	// ✅ Используем 'db' из параметра
	castings, total, err := s.castingRepo.SearchCastings(db, searchCriteria)
	if err != nil {
		return nil, 0, apperrors.InternalError(err)
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		// ✅ Используем 'db' из параметра
		response, err := s.buildCastingResponse(db, &casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, total, nil
}

// GetEmployerCastings - 'db' добавлен
func (s *CastingServiceImpl) GetEmployerCastings(db *gorm.DB, employerID string, requesterID string) ([]*dto.CastingResponse, error) {
	if employerID != requesterID {
		return nil, apperrors.ErrInsufficientPermissions
	}

	// ✅ Используем 'db' из параметра
	castings, err := s.castingRepo.FindCastingsByEmployer(db, employerID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		// ✅ Используем 'db' из параметра
		response, err := s.buildCastingResponse(db, &casting, true)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// GetActiveCastings - 'db' добавлен
func (s *CastingServiceImpl) GetActiveCastings(db *gorm.DB, limit int) ([]*dto.CastingResponse, error) {
	// ✅ Используем 'db' из параметра
	castings, err := s.castingRepo.FindActiveCastings(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		// ✅ Используем 'db' из параметра
		response, err := s.buildCastingResponse(db, &casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// GetCastingsByCity - 'db' добавлен
func (s *CastingServiceImpl) GetCastingsByCity(db *gorm.DB, city string, limit int) ([]*dto.CastingResponse, error) {
	// ✅ Используем 'db' из параметра
	castings, err := s.castingRepo.FindCastingsByCity(db, city, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		// ✅ Используем 'db' из параметра
		response, err := s.buildCastingResponse(db, &casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// Stats and Analytics

// GetCastingStats - 'db' добавлен
func (s *CastingServiceImpl) GetCastingStats(db *gorm.DB, employerID string, requesterID string) (*repositories.CastingStats, error) {
	if employerID != requesterID {
		return nil, apperrors.ErrInsufficientPermissions
	}
	// ✅ Используем 'db' из параметра
	stats, err := s.castingRepo.GetCastingStats(db, employerID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return stats, nil
}

// Matching Operations

// FindMatchingCastings - 'db' добавлен
func (s *CastingServiceImpl) FindMatchingCastings(db *gorm.DB, modelID string, limit int) ([]*dto.CastingResponse, error) {
	// ✅ Используем 'db' из параметра
	profile, err := s.profileRepo.FindModelProfileByUserID(db, modelID)
	if err != nil {
		return nil, handleCastingError(err)
	}

	criteria := repositories.MatchingCriteria{
		Limit: limit,
	}
	if profile.Gender != "" {
		criteria.Gender = profile.Gender
	}
	if profile.Height > 0 {
		height := int(profile.Height)
		criteria.MinHeight = &height
		criteria.MaxHeight = &height
	}
	var modelCategories []string
	if len(profile.Categories) > 0 {
		json.Unmarshal(profile.Categories, &modelCategories)
	}
	if len(modelCategories) > 0 {
		criteria.Categories = modelCategories
	}
	// ... (другие критерии)

	// ✅ Используем 'db' из параметра
	castings, err := s.castingRepo.FindCastingsForMatching(db, criteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var matchingCastings []models.Casting
	for _, casting := range castings {
		if s.isModelMatchesCasting(profile, &casting) {
			matchingCastings = append(matchingCastings, casting)
			if len(matchingCastings) >= limit {
				break
			}
		}
	}

	var responses []*dto.CastingResponse
	for _, casting := range matchingCastings {
		// ✅ Используем 'db' из параметра
		response, err := s.buildCastingResponse(db, &casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// Helper Methods

// buildCastingResponse - 'db' добавлен
func (s *CastingServiceImpl) buildCastingResponse(db *gorm.DB, casting *models.Casting, includeResponses bool) (*dto.CastingResponse, error) {
	var categories []string
	var languages []string

	if len(casting.Categories) > 0 {
		json.Unmarshal(casting.Categories, &categories)
	}
	if len(casting.Languages) > 0 {
		json.Unmarshal(casting.Languages, &languages)
	}

	response := &dto.CastingResponse{
		ID:              casting.ID,
		EmployerID:      casting.EmployerID,
		Title:           casting.Title,
		Description:     casting.Description,
		PaymentMin:      casting.PaymentMin,
		PaymentMax:      casting.PaymentMax,
		CastingDate:     casting.CastingDate,
		CastingTime:     casting.CastingTime,
		Address:         casting.Address,
		City:            casting.City,
		Categories:      categories,
		Gender:          casting.Gender,
		AgeMin:          casting.AgeMin,
		AgeMax:          casting.AgeMax,
		HeightMin:       casting.HeightMin,
		HeightMax:       casting.HeightMax,
		WeightMin:       casting.WeightMin,
		WeightMax:       casting.WeightMax,
		ClothingSize:    casting.ClothingSize,
		ShoeSize:        casting.ShoeSize,
		ExperienceLevel: casting.ExperienceLevel,
		Languages:       languages,
		JobType:         casting.JobType,
		Status:          casting.Status,
		Views:           casting.Views,
		Employer:        casting.Employer,
		CreatedAt:       casting.CreatedAt,
		UpdatedAt:       casting.UpdatedAt,
	}

	if includeResponses {
		// ✅ Используем 'db' из параметра
		responses, err := s.responseRepo.FindResponsesByCasting(db, casting.ID)
		if err == nil {
			var responseSummaries []dto.ResponseSummary
			for _, resp := range responses {
				summary := dto.ResponseSummary{
					ID:        resp.ID,
					ModelID:   resp.ModelID,
					ModelName: resp.Model.Name,
					Message:   resp.Message,
					Status:    resp.Status,
					CreatedAt: resp.CreatedAt,
				}
				responseSummaries = append(responseSummaries, summary)
			}
			response.Responses = responseSummaries
		}

		// ✅ Используем 'db' из параметра
		stats, err := s.responseRepo.GetResponseStats(db, casting.ID)
		if err == nil {
			response.Stats = &dto.CastingStatsResponse{
				TotalResponses:    stats.TotalResponses,
				PendingResponses:  stats.PendingResponses,
				AcceptedResponses: stats.AcceptedResponses,
				RejectedResponses: stats.RejectedResponses,
			}
		}
	}

	return response, nil
}

// UpdateCastingStatus - 'db' добавлен
func (s *CastingServiceImpl) UpdateCastingStatus(db *gorm.DB, castingID string, requesterID string, status models.CastingStatus) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return handleCastingError(err)
	}

	if casting.EmployerID != requesterID {
		return apperrors.ErrInsufficientPermissions
	}

	if !isValidStatusTransition(casting.Status, status) {
		return apperrors.ErrInvalidCastingStatus
	}

	// ✅ Передаем tx
	if err := s.castingRepo.UpdateCastingStatus(tx, castingID, status); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetCastingStatsForCasting - 'db' добавлен
func (s *CastingServiceImpl) GetCastingStatsForCasting(db *gorm.DB, castingID string, requesterID string) (*dto.CastingStatsResponse, error) {
	// ✅ Используем 'db' из параметра
	casting, err := s.castingRepo.FindCastingByID(db, castingID)
	if err != nil {
		return nil, handleCastingError(err)
	}

	if casting.EmployerID != requesterID {
		return nil, apperrors.ErrInsufficientPermissions
	}

	// ✅ Используем 'db' из параметра
	stats, err := s.responseRepo.GetResponseStats(db, castingID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &dto.CastingStatsResponse{
		TotalResponses:    stats.TotalResponses,
		PendingResponses:  stats.PendingResponses,
		AcceptedResponses: stats.AcceptedResponses,
		RejectedResponses: stats.RejectedResponses,
	}, nil
}

// CloseExpiredCastings - 'db' добавлен
func (s *CastingServiceImpl) CloseExpiredCastings(db *gorm.DB) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	castings, err := s.castingRepo.FindExpiredCastings(tx)
	if err != nil {
		return apperrors.InternalError(err)
	}

	for _, casting := range castings {
		// ✅ Передаем tx
		if err := s.castingRepo.UpdateCastingStatus(tx, casting.ID, models.CastingStatusClosed); err != nil {
			fmt.Printf("Failed to close casting %s: %v\n", casting.ID, err)
			return apperrors.InternalError(err)
		}
	}
	return tx.Commit().Error
}

// (isValidStatusTransition - чистая функция, без изменений)
func isValidStatusTransition(currentStatus, newStatus models.CastingStatus) bool {
	validTransitions := map[models.CastingStatus][]models.CastingStatus{
		models.CastingStatusDraft: {
			models.CastingStatusActive,
		},
		models.CastingStatusActive: {
			models.CastingStatusClosed,
		},
		models.CastingStatusClosed: {
			models.CastingStatusActive,
		},
	}
	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}
	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

// (isModelMatchesCasting - чистая функция, без изменений)
func (s *CastingServiceImpl) isModelMatchesCasting(profile *models.ModelProfile, casting *models.Casting) bool {
	if casting.Gender != "" && profile.Gender != "" && casting.Gender != profile.Gender {
		return false
	}
	if profile.Age > 0 {
		if casting.AgeMin != nil && profile.Age < *casting.AgeMin {
			return false
		}
		if casting.AgeMax != nil && profile.Age > *casting.AgeMax {
			return false
		}
	}
	if profile.Height > 0 {
		if casting.HeightMin != nil && float64(profile.Height) < *casting.HeightMin {
			return false
		}
		if casting.HeightMax != nil && float64(profile.Height) > *casting.HeightMax {
			return false
		}
	}
	if profile.Weight > 0 {
		if casting.WeightMin != nil && float64(profile.Weight) < *casting.WeightMin {
			return false
		}
		if casting.WeightMax != nil && float64(profile.Weight) > *casting.WeightMax {
			return false
		}
	}
	if len(casting.Categories) > 0 && len(profile.Categories) > 0 {
		var castingCategories []string
		var profileCategories []string
		json.Unmarshal(casting.Categories, &castingCategories)
		json.Unmarshal(profile.Categories, &profileCategories)
		if !hasCommonElements(castingCategories, profileCategories) {
			return false
		}
	}
	return true
}

// (hasCommonElements - чистая функция, без изменений)
func hasCommonElements(slice1, slice2 []string) bool {
	elementMap := make(map[string]bool)
	for _, item := range slice1 {
		elementMap[item] = true
	}
	for _, item := range slice2 {
		if elementMap[item] {
			return true
		}
	}
	return false
}

// ▼▼▼ ДОБАВЛЕНЫ РЕАЛИЗАЦИИ-ЗАГЛУШКИ ДЛЯ ADMIN МЕТОДОВ ▼▼▼

func (s *CastingServiceImpl) GetPlatformCastingStats(db *gorm.DB, dateFrom time.Time, dateTo time.Time) (interface{}, error) {
	// TODO: Implement actual logic by calling repository
	// return s.castingRepo.GetPlatformCastingStats(db, dateFrom, dateTo)
	fmt.Printf("GetPlatformCastingStats called with range: %v to %v\n", dateFrom, dateTo)
	return nil, errors.New("admin statistics: GetPlatformCastingStats not implemented")
}

func (s *CastingServiceImpl) GetMatchingStats(db *gorm.DB, dateFrom time.Time, dateTo time.Time) (interface{}, error) {
	// TODO: Implement actual logic by calling repository
	// return s.castingRepo.GetMatchingStats(db, dateFrom, dateTo)
	fmt.Printf("GetMatchingStats called with range: %v to %v\n", dateFrom, dateTo)
	return nil, errors.New("admin statistics: GetMatchingStats not implemented")
}

func (s *CastingServiceImpl) GetCastingDistributionByCity(db *gorm.DB) (interface{}, error) {
	// TODO: Implement actual logic by calling repository
	// return s.castingRepo.GetCastingDistributionByCity(db)
	return nil, errors.New("admin statistics: GetCastingDistributionByCity not implemented")
}

func (s *CastingServiceImpl) GetActiveCastingsCount(db *gorm.DB) (int64, error) {
	// TODO: Implement actual logic by calling repository
	// return s.castingRepo.GetActiveCastingsCount(db)
	return 0, errors.New("admin statistics: GetActiveCastingsCount not implemented")
}

func (s *CastingServiceImpl) GetPopularCategories(db *gorm.DB, limit int) (interface{}, error) {
	// TODO: Implement actual logic by calling repository
	// return s.castingRepo.GetPopularCategories(db, limit)
	fmt.Printf("GetPopularCategories called with limit: %d\n", limit)
	return nil, errors.New("admin statistics: GetPopularCategories not implemented")
}

// ▲▲▲ ДОБАВЛЕНЫ РЕАЛИЗАЦИИ-ЗАГЛУШКИ ДЛЯ ADMIN МЕТОДОВ ▲▲▲

// (handleCastingError - хелпер, без изменений)
func handleCastingError(err error) error {
	if errors.Is(err, repositories.ErrCastingNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"

	"mwork_backend/internal/appErrors"

	"gorm.io/datatypes"
)

type CastingService struct {
	castingRepo      repositories.CastingRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	subscriptionRepo repositories.SubscriptionRepository
	notificationRepo repositories.NotificationRepository
	reviewRepo       repositories.ReviewRepository
	responseRepo     repositories.ResponseRepository
}

func NewCastingService(
	castingRepo repositories.CastingRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	notificationRepo repositories.NotificationRepository,
	reviewRepo repositories.ReviewRepository,
	responseRepo repositories.ResponseRepository,
) *CastingService {
	return &CastingService{
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

func (s *CastingService) CreateCasting(req *dto.CreateCastingRequest) error {
	employer, err := s.userRepo.FindByID(req.EmployerID)
	if err != nil {
		return err
	}

	if employer.Role != models.UserRoleEmployer {
		return appErrors.ErrInsufficientPermissions
	}

	canPublish, err := s.subscriptionRepo.CanUserPublish(req.EmployerID)
	if err != nil {
		return err
	}

	if !canPublish {
		return appErrors.ErrSubscriptionLimit
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

	err = s.castingRepo.CreateCasting(casting)
	if err != nil {
		return err
	}

	go s.subscriptionRepo.IncrementSubscriptionUsage(req.EmployerID, "publications")

	return nil
}

func (s *CastingService) GetCasting(castingID string, requesterID string) (*dto.CastingResponse, error) {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, err
	}

	if requesterID != casting.EmployerID {
		go s.castingRepo.IncrementCastingViews(castingID)
	}

	return s.buildCastingResponse(casting, requesterID == casting.EmployerID)
}

func (s *CastingService) UpdateCasting(castingID string, requesterID string, req *dto.UpdateCastingRequest) error {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != requesterID {
		return appErrors.ErrInsufficientPermissions
	}

	if casting.Status != models.CastingStatusDraft {
		return appErrors.ErrInvalidCastingStatus
	}

	if req.Title != nil {
		casting.Title = *req.Title
	}
	if req.Description != nil {
		casting.Description = *req.Description
	}
	if req.PaymentMin != nil {
		casting.PaymentMin = *req.PaymentMin
	}
	if req.PaymentMax != nil {
		casting.PaymentMax = *req.PaymentMax
	}
	if req.CastingDate != nil {
		casting.CastingDate = req.CastingDate
	}
	if req.CastingTime != nil {
		casting.CastingTime = req.CastingTime
	}
	if req.Address != nil {
		casting.Address = req.Address
	}
	if req.City != nil {
		casting.City = *req.City
	}
	if req.Gender != nil {
		casting.Gender = *req.Gender
	}
	if req.AgeMin != nil {
		casting.AgeMin = req.AgeMin
	}
	if req.AgeMax != nil {
		casting.AgeMax = req.AgeMax
	}
	if req.HeightMin != nil {
		casting.HeightMin = req.HeightMin
	}
	if req.HeightMax != nil {
		casting.HeightMax = req.HeightMax
	}
	if req.WeightMin != nil {
		casting.WeightMin = req.WeightMin
	}
	if req.WeightMax != nil {
		casting.WeightMax = req.WeightMax
	}
	if req.ClothingSize != nil {
		casting.ClothingSize = req.ClothingSize
	}
	if req.ShoeSize != nil {
		casting.ShoeSize = req.ShoeSize
	}
	if req.ExperienceLevel != nil {
		casting.ExperienceLevel = req.ExperienceLevel
	}
	if req.JobType != nil {
		casting.JobType = *req.JobType
	}

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

	return s.castingRepo.UpdateCasting(casting)
}

func (s *CastingService) PublishCasting(castingID string, requesterID string) error {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != requesterID {
		return appErrors.ErrInsufficientPermissions
	}

	if casting.Status != models.CastingStatusDraft {
		return appErrors.ErrInvalidCastingStatus
	}

	casting.Status = models.CastingStatusActive
	return s.castingRepo.UpdateCasting(casting)
}

func (s *CastingService) CloseCasting(castingID string, requesterID string) error {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != requesterID {
		return appErrors.ErrInsufficientPermissions
	}

	if casting.Status != models.CastingStatusActive {
		return appErrors.ErrInvalidCastingStatus
	}

	casting.Status = models.CastingStatusClosed
	return s.castingRepo.UpdateCasting(casting)
}

func (s *CastingService) DeleteCasting(castingID string, requesterID string) error {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != requesterID {
		return appErrors.ErrInsufficientPermissions
	}

	if casting.Status != models.CastingStatusDraft {
		return appErrors.ErrInvalidCastingStatus
	}

	return s.castingRepo.DeleteCasting(castingID)
}

// Search and Discovery

func (s *CastingService) SearchCastings(criteria dto.CastingSearchCriteria) ([]*dto.CastingResponse, int64, error) {
	searchCriteria := repositories.CastingSearchCriteria{
		Query:      criteria.Query,
		City:       criteria.City,
		Categories: criteria.Categories,
		Gender:     criteria.Gender,
		MinAge:     criteria.MinAge,
		MaxAge:     criteria.MaxAge,
		MinHeight:  criteria.MinHeight,
		MaxHeight:  criteria.MaxHeight,
		MinSalary:  criteria.MinSalary,
		MaxSalary:  criteria.MaxSalary,
		JobType:    criteria.JobType,
		Status:     criteria.Status,
		EmployerID: criteria.EmployerID,
		DateFrom:   criteria.DateFrom,
		DateTo:     criteria.DateTo,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		SortBy:     criteria.SortBy,
		SortOrder:  criteria.SortOrder,
	}

	castings, total, err := s.castingRepo.SearchCastings(searchCriteria)
	if err != nil {
		return nil, 0, err
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		response, err := s.buildCastingResponse(&casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, total, nil
}

func (s *CastingService) GetEmployerCastings(employerID string, requesterID string) ([]*dto.CastingResponse, error) {
	if employerID != requesterID {
		return nil, appErrors.ErrInsufficientPermissions
	}

	castings, err := s.castingRepo.FindCastingsByEmployer(employerID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		response, err := s.buildCastingResponse(&casting, true)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (s *CastingService) GetActiveCastings(limit int) ([]*dto.CastingResponse, error) {
	castings, err := s.castingRepo.FindActiveCastings(limit)
	if err != nil {
		return nil, err
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		response, err := s.buildCastingResponse(&casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (s *CastingService) GetCastingsByCity(city string, limit int) ([]*dto.CastingResponse, error) {
	castings, err := s.castingRepo.FindCastingsByCity(city, limit)
	if err != nil {
		return nil, err
	}

	var responses []*dto.CastingResponse
	for _, casting := range castings {
		response, err := s.buildCastingResponse(&casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// Stats and Analytics

func (s *CastingService) GetCastingStats(employerID string, requesterID string) (*repositories.CastingStats, error) {
	if employerID != requesterID {
		return nil, appErrors.ErrInsufficientPermissions
	}

	return s.castingRepo.GetCastingStats(employerID)
}

// New Analytics Methods

func (s *CastingService) GetPlatformCastingStats(dateFrom, dateTo time.Time) (*dto.PlatformCastingStatsResponse, error) {
	stats, err := s.castingRepo.GetPlatformCastingStats(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	return &dto.PlatformCastingStatsResponse{
		TotalCastings:   stats.TotalCastings,
		ActiveCastings:  stats.ActiveCastings,
		SuccessRate:     stats.SuccessRate,
		AvgResponseRate: stats.AvgResponseRate,
		AvgResponseTime: stats.AvgResponseTime,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
	}, nil
}

func (s *CastingService) GetMatchingStats(dateFrom, dateTo time.Time) (*dto.MatchingStatsResponse, error) {
	stats, err := s.castingRepo.GetMatchingStats(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	return &dto.MatchingStatsResponse{
		TotalMatches:    stats.TotalMatches,
		AvgMatchScore:   stats.AvgMatchScore,
		AvgSatisfaction: stats.AvgSatisfaction,
		MatchRate:       stats.MatchRate,
		ResponseRate:    stats.ResponseRate,
		TimeToMatch:     stats.TimeToMatch,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
	}, nil
}

func (s *CastingService) GetCastingDistributionByCity() (map[string]int64, error) {
	return s.castingRepo.GetCastingDistributionByCity()
}

func (s *CastingService) GetActiveCastingsCount() (int64, error) {
	return s.castingRepo.GetActiveCastingsCount()
}

func (s *CastingService) GetPopularCategories(limit int) ([]dto.CategoryCountResponse, error) {
	categories, err := s.castingRepo.GetPopularCategories(limit)
	if err != nil {
		return nil, err
	}

	var response []dto.CategoryCountResponse
	for _, category := range categories {
		response = append(response, dto.CategoryCountResponse{
			Name:  category.Name,
			Count: category.Count,
		})
	}

	return response, nil
}

// Matching Operations

func (s *CastingService) FindMatchingCastings(modelID string, limit int) ([]*dto.CastingResponse, error) {
	// Получаем профиль модели
	profile, err := s.profileRepo.FindModelProfileByUserID(modelID)
	if err != nil {
		return nil, err
	}

	// Создаем критерии для мэтчинга
	criteria := repositories.MatchingCriteria{
		Limit: limit,
	}

	// Фильтруем по параметрам модели (используем прямое присвоение, так как поля не указатели)
	if profile.Gender != "" {
		criteria.Gender = profile.Gender
	}

	if profile.City != "" {
		criteria.City = profile.City
	}

	// Для числовых полей создаем указатели
	if profile.Age > 0 {
		age := profile.Age
		criteria.MinAge = &age
		criteria.MaxAge = &age
	}

	if profile.Height > 0 {
		height := int(profile.Height)
		criteria.MinHeight = &height
		criteria.MaxHeight = &height
	}

	// Получаем категории модели
	var modelCategories []string
	if len(profile.Categories) > 0 {
		json.Unmarshal(profile.Categories, &modelCategories)
	}
	if len(modelCategories) > 0 {
		criteria.Categories = modelCategories
	}

	// Поиск подходящих кастингов
	castings, err := s.castingRepo.FindCastingsForMatching(criteria)
	if err != nil {
		return nil, err
	}

	// Дополнительная фильтрация по сложным критериям
	var matchingCastings []models.Casting
	for _, casting := range castings {
		if s.isModelMatchesCasting(profile, &casting) {
			matchingCastings = append(matchingCastings, casting)
			if len(matchingCastings) >= limit {
				break
			}
		}
	}

	// Преобразуем в ответы
	var responses []*dto.CastingResponse
	for _, casting := range matchingCastings {
		response, err := s.buildCastingResponse(&casting, false)
		if err != nil {
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// Helper Methods

func (s *CastingService) buildCastingResponse(casting *models.Casting, includeResponses bool) (*dto.CastingResponse, error) {
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
		responses, err := s.responseRepo.FindResponsesByCasting(casting.ID)
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

		stats, err := s.responseRepo.GetResponseStats(casting.ID)
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

// UpdateCastingStatus - обновление статуса кастинга
func (s *CastingService) UpdateCastingStatus(castingID string, requesterID string, status models.CastingStatus) error {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != requesterID {
		return appErrors.ErrInsufficientPermissions
	}

	// Проверка допустимых переходов статусов
	if !isValidStatusTransition(casting.Status, status) {
		return appErrors.ErrInvalidCastingStatus
	}

	return s.castingRepo.UpdateCastingStatus(castingID, status)
}

// GetCastingStatsForCasting - получение статистики по конкретному кастингу
func (s *CastingService) GetCastingStatsForCasting(castingID string, requesterID string) (*dto.CastingStatsResponse, error) {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, err
	}

	if casting.EmployerID != requesterID {
		return nil, appErrors.ErrInsufficientPermissions
	}

	stats, err := s.responseRepo.GetResponseStats(castingID)
	if err != nil {
		return nil, err
	}

	return &dto.CastingStatsResponse{
		TotalResponses:    stats.TotalResponses,
		PendingResponses:  stats.PendingResponses,
		AcceptedResponses: stats.AcceptedResponses,
		RejectedResponses: stats.RejectedResponses,
	}, nil
}

// CloseExpiredCastings - закрытие истекших кастингов
func (s *CastingService) CloseExpiredCastings() error {
	// Получаем все активные кастинги с истекшей датой
	castings, err := s.castingRepo.FindExpiredCastings()
	if err != nil {
		return err
	}

	// Закрываем каждый истекший кастинг
	for _, casting := range castings {
		if err := s.castingRepo.UpdateCastingStatus(casting.ID, models.CastingStatusClosed); err != nil {
			// Логируем ошибку, но продолжаем обработку остальных
			continue
		}
	}

	return nil
}

// Helper functions

// isValidStatusTransition - проверка допустимых переходов статусов
func isValidStatusTransition(currentStatus, newStatus models.CastingStatus) bool {
	validTransitions := map[models.CastingStatus][]models.CastingStatus{
		models.CastingStatusDraft: {
			models.CastingStatusActive,
		},
		models.CastingStatusActive: {
			models.CastingStatusClosed,
		},
		models.CastingStatusClosed: {
			models.CastingStatusActive, // Можно переоткрыть
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

// isModelMatchesCasting - проверка соответствия модели требованиям кастинга
func (s *CastingService) isModelMatchesCasting(profile *models.ModelProfile, casting *models.Casting) bool {
	// Проверка пола
	if casting.Gender != "" && profile.Gender != "" && casting.Gender != profile.Gender {
		return false
	}

	// Проверка возраста
	if profile.Age > 0 {
		if casting.AgeMin != nil && profile.Age < *casting.AgeMin {
			return false
		}
		if casting.AgeMax != nil && profile.Age > *casting.AgeMax {
			return false
		}
	}

	// Проверка роста
	if profile.Height > 0 {
		if casting.HeightMin != nil && profile.Height < *casting.HeightMin {
			return false
		}
		if casting.HeightMax != nil && profile.Height > *casting.HeightMax {
			return false
		}
	}

	// Проверка веса
	if profile.Weight > 0 {
		if casting.WeightMin != nil && profile.Weight < *casting.WeightMin {
			return false
		}
		if casting.WeightMax != nil && profile.Weight > *casting.WeightMax {
			return false
		}
	}

	// Проверка размера одежды
	if casting.ClothingSize != nil && profile.ClothingSize != "" {
		if *casting.ClothingSize != profile.ClothingSize {
			return false
		}
	}

	// Проверка размера обуви
	if casting.ShoeSize != nil && profile.ShoeSize != "" {
		if *casting.ShoeSize != profile.ShoeSize {
			return false
		}
	}

	// Проверка категорий
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

// hasCommonElements - проверка наличия общих элементов в двух слайсах
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

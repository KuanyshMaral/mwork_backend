package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"

	"gorm.io/datatypes"
	"mwork_backend/internal/appErrors"
)

type CastingService struct {
	castingRepo      repositories.CastingRepository
	responseRepo     repositories.ResponseRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
	subscriptionRepo repositories.SubscriptionRepository
	notificationRepo repositories.NotificationRepository
	reviewRepo       repositories.ReviewRepository
}

func NewCastingService(
	castingRepo repositories.CastingRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	notificationRepo repositories.NotificationRepository,
	reviewRepo repositories.ReviewRepository,
) *CastingService {
	return &CastingService{
		castingRepo:      castingRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
		reviewRepo:       reviewRepo,
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

// Response Operations

func (s *CastingService) CreateResponse(req *dto.CreateResponseRequest) error {
	model, err := s.userRepo.FindByID(req.ModelID)
	if err != nil {
		return err
	}

	if model.Role != models.UserRoleModel {
		return appErrors.ErrInsufficientPermissions
	}

	casting, err := s.castingRepo.FindCastingByID(req.CastingID)
	if err != nil {
		return err
	}

	if casting.Status != models.CastingStatusActive {
		return appErrors.ErrCastingNotActive
	}

	if casting.CastingDate != nil && casting.CastingDate.Before(time.Now()) {
		return appErrors.ErrCastingExpired
	}

	if casting.EmployerID == req.ModelID {
		return appErrors.ErrCannotRespondToOwnCasting
	}

	canRespond, err := s.subscriptionRepo.CanUserRespond(req.ModelID)
	if err != nil {
		return err
	}

	if !canRespond {
		return appErrors.ErrSubscriptionLimit
	}

	response := &models.CastingResponse{
		CastingID: req.CastingID,
		ModelID:   req.ModelID,
		Message:   req.Message,
		Status:    models.ResponseStatusPending,
	}

	err = s.responseRepo.CreateResponse(response)
	if err != nil {
		if errors.Is(err, repositories.ErrResponseAlreadyExists) {
			return appErrors.ErrResponseAlreadyExists
		}
		return err
	}

	go s.subscriptionRepo.IncrementSubscriptionUsage(req.ModelID, "responses")
	go s.notificationRepo.CreateNewResponseNotification(
		casting.EmployerID,
		casting.ID,
		response.ID,
		model.Email,
	)

	return nil
}

func (s *CastingService) UpdateResponseStatus(responseID string, requesterID string, req *dto.UpdateResponseStatusRequest) error {
	response, err := s.responseRepo.FindResponseByID(responseID)
	if err != nil {
		return err
	}

	casting, err := s.castingRepo.FindCastingByID(response.CastingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != requesterID {
		return appErrors.ErrInsufficientPermissions
	}

	oldStatus := response.Status
	response.Status = req.Status

	err = s.responseRepo.UpdateResponseStatus(responseID, req.Status)
	if err != nil {
		return err
	}

	if oldStatus != req.Status {
		go s.notificationRepo.CreateResponseStatusNotification(
			response.ModelID,
			casting.Title,
			req.Status,
		)
	}

	if req.Status == models.ResponseStatusAccepted {
		go s.createReviewPlaceholder(casting, response)
	}

	return nil
}

func (s *CastingService) GetCastingResponses(castingID string, requesterID string) ([]dto.ResponseSummary, error) {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, err
	}

	if casting.EmployerID != requesterID {
		return nil, appErrors.ErrInsufficientPermissions
	}

	responses, err := s.responseRepo.FindResponsesByCasting(castingID)
	if err != nil {
		return nil, err
	}

	var summaries []dto.ResponseSummary
	for _, response := range responses {
		summary := dto.ResponseSummary{
			ID:        response.ID,
			ModelID:   response.ModelID,
			ModelName: response.Model.Name,
			Message:   response.Message,
			Status:    response.Status,
			CreatedAt: response.CreatedAt,
			Model:     &response.Model,
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *CastingService) GetModelResponses(modelID string, requesterID string) ([]models.CastingResponse, error) {
	if modelID != requesterID {
		return nil, appErrors.ErrInsufficientPermissions
	}

	return s.responseRepo.FindResponsesByModel(modelID)
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

func (s *CastingService) GetResponseStats(castingID string, requesterID string) (*repositories.ResponseStats, error) {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, err
	}

	if casting.EmployerID != requesterID {
		return nil, appErrors.ErrInsufficientPermissions
	}

	return s.responseRepo.GetResponseStats(castingID)
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

func (s *CastingService) createReviewPlaceholder(casting *models.Casting, response *models.CastingResponse) {
	review := &models.Review{
		ModelID:    response.ModelID,
		EmployerID: casting.EmployerID,
		CastingID:  &casting.ID,
		Rating:     0,
		ReviewText: "",
		Status:     models.ReviewStatusPending,
	}

	s.reviewRepo.CreateReview(review)
}

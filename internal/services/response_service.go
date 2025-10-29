package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

// ResponseService - интерфейс для операций, связанных с откликами (Responses).
type ResponseService interface {
	CreateResponse(modelID, castingID string, req *dto.CreateResponseRequest) (*models.CastingResponse, error)
	GetModelResponses(modelID string) ([]models.CastingResponse, error)
	DeleteResponse(modelID, responseID string) error
	GetCastingResponses(castingID, employerID string) ([]dto.ResponseSummary, error)
	UpdateResponseStatus(employerID, responseID string, status models.ResponseStatus) error
	MarkResponseAsViewed(employerID, responseID string) error
	GetResponseStats(castingID string) (*dto.CastingStatsResponse, error)
	GetResponse(responseID, userID string) (*models.CastingResponse, error)
}

// ResponseServiceImpl - конкретная реализация интерфейса ResponseService.
// ПЕРЕИМЕНОВАНО: Было ResponseService, стало ResponseServiceImpl.
type ResponseServiceImpl struct {
	responseRepo     repositories.ResponseRepository
	castingRepo      repositories.CastingRepository
	userRepo         repositories.UserRepository
	subscriptionRepo repositories.SubscriptionRepository
	notificationRepo repositories.NotificationRepository
	reviewRepo       repositories.ReviewRepository
}

// NewResponseService - конструктор, возвращающий тип ИНТЕРФЕЙСА.
func NewResponseService(
	responseRepo repositories.ResponseRepository,
	castingRepo repositories.CastingRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	notificationRepo repositories.NotificationRepository,
	reviewRepo repositories.ReviewRepository,
) ResponseService { // <--- ИЗМЕНЕНО: теперь возвращает интерфейс ResponseService
	return &ResponseServiceImpl{
		responseRepo:     responseRepo,
		castingRepo:      castingRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
		reviewRepo:       reviewRepo,
	}
}

// Response Operations

func (s *ResponseServiceImpl) CreateResponse(modelID, castingID string, req *dto.CreateResponseRequest) (*models.CastingResponse, error) {
	model, err := s.userRepo.FindByID(modelID)
	if err != nil {
		return nil, err
	}

	if model.Role != models.UserRoleModel {
		return nil, appErrors.ErrInsufficientPermissions
	}

	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, err
	}

	if casting.Status != models.CastingStatusActive {
		return nil, errors.New("casting is not active")
	}

	if casting.CastingDate != nil && casting.CastingDate.Before(time.Now()) {
		return nil, errors.New("casting has expired")
	}

	if casting.EmployerID == modelID {
		return nil, errors.New("cannot respond to your own casting")
	}

	// Check if already responded
	existingResponse, _ := s.responseRepo.FindResponseByCastingAndModel(castingID, modelID)
	if existingResponse != nil {
		return nil, errors.New("you have already responded to this casting")
	}

	canRespond, err := s.subscriptionRepo.CanUserRespond(modelID)
	if err != nil {
		return nil, err
	}

	if !canRespond {
		return nil, errors.New("subscription limit reached")
	}

	// Increment subscription usage BEFORE creating response to prevent race conditions
	err = s.subscriptionRepo.IncrementSubscriptionUsage(modelID, "responses")
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	response := &models.CastingResponse{
		CastingID: castingID,
		ModelID:   modelID,
		Message:   req.Message,
		Status:    models.ResponseStatusPending,
	}

	err = s.responseRepo.CreateResponse(response)
	if err != nil {
		// Rollback subscription usage if response creation fails
		rollbackErr := s.subscriptionRepo.DecrementSubscriptionUsage(modelID, "responses")
		if rollbackErr != nil {
			log.Printf("Failed to rollback subscription usage: %v", rollbackErr)
		}

		if errors.Is(err, repositories.ErrResponseAlreadyExists) {
			return nil, errors.New("response already exists")
		}
		return nil, err
	}

	// Create notification with error handling
	go func() {
		if err := s.notificationRepo.CreateNewResponseNotification(
			casting.EmployerID,
			casting.ID,
			response.ID,
			model.Email,
		); err != nil {
			log.Printf("Failed to create notification: %v", err)
		}
	}()

	return response, nil
}

func (s *ResponseServiceImpl) GetModelResponses(modelID string) ([]models.CastingResponse, error) {
	return s.responseRepo.FindResponsesByModel(modelID)
}

func (s *ResponseServiceImpl) DeleteResponse(modelID, responseID string) error {
	response, err := s.responseRepo.FindResponseByID(responseID)
	if err != nil {
		return err
	}

	if response.ModelID != modelID {
		return errors.New("access denied")
	}

	if response.Status != models.ResponseStatusPending {
		return errors.New("cannot delete response that has been reviewed")
	}

	// Decrement subscription usage when response is deleted
	err = s.responseRepo.DeleteResponse(responseID)
	if err != nil {
		return err
	}

	// Decrement subscription usage in background
	go func() {
		if err := s.subscriptionRepo.DecrementSubscriptionUsage(modelID, "responses"); err != nil {
			log.Printf("Failed to decrement subscription usage: %v", err)
		}
	}()

	return nil
}

func (s *ResponseServiceImpl) GetCastingResponses(castingID, employerID string) ([]dto.ResponseSummary, error) {
	casting, err := s.castingRepo.FindCastingByID(castingID)
	if err != nil {
		return nil, err
	}

	if casting.EmployerID != employerID {
		return nil, errors.New("access denied")
	}

	responses, err := s.responseRepo.FindResponsesByCasting(castingID)
	if err != nil {
		return nil, err
	}

	var summaries []dto.ResponseSummary
	for _, response := range responses {
		model, err := s.userRepo.FindByID(response.ModelID)
		modelName := ""
		if err == nil && model != nil {
			modelName = model.Name
		}

		summary := dto.ResponseSummary{
			ID:        response.ID,
			ModelID:   response.ModelID,
			ModelName: modelName,
			Message:   response.Message,
			Status:    response.Status,
			CreatedAt: response.CreatedAt,
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *ResponseServiceImpl) UpdateResponseStatus(employerID, responseID string, status models.ResponseStatus) error {
	response, err := s.responseRepo.FindResponseByID(responseID)
	if err != nil {
		return err
	}

	casting, err := s.castingRepo.FindCastingByID(response.CastingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != employerID {
		return errors.New("access denied")
	}

	oldStatus := response.Status
	err = s.responseRepo.UpdateResponseStatus(responseID, status)
	if err != nil {
		return err
	}

	if oldStatus != status {
		model, err := s.userRepo.FindByID(response.ModelID)
		if err == nil && model != nil {
			go func() {
				if err := s.notificationRepo.CreateResponseStatusNotification(
					response.ModelID,
					casting.Title,
					status,
				); err != nil {
					log.Printf("Failed to create status notification: %v", err)
				}
			}()
		}

		if status == models.ResponseStatusAccepted {
			go s.createReviewPlaceholder(casting, response)
		}
	}

	return nil
}

func (s *ResponseServiceImpl) MarkResponseAsViewed(employerID, responseID string) error {
	response, err := s.responseRepo.FindResponseByID(responseID)
	if err != nil {
		return err
	}

	casting, err := s.castingRepo.FindCastingByID(response.CastingID)
	if err != nil {
		return err
	}

	if casting.EmployerID != employerID {
		return errors.New("access denied")
	}

	return s.responseRepo.MarkResponseAsViewed(responseID)
}

func (s *ResponseServiceImpl) GetResponseStats(castingID string) (*dto.CastingStatsResponse, error) {
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

func (s *ResponseServiceImpl) GetResponse(responseID, userID string) (*models.CastingResponse, error) {
	response, err := s.responseRepo.FindResponseByID(responseID)
	if err != nil {
		return nil, err
	}

	// Check access rights
	casting, err := s.castingRepo.FindCastingByID(response.CastingID)
	if err != nil {
		return nil, err
	}

	// Allow access to model who created response or employer who owns casting
	if response.ModelID != userID && casting.EmployerID != userID {
		return nil, errors.New("access denied")
	}

	return response, nil
}

// Helper Methods

func (s *ResponseServiceImpl) createReviewPlaceholder(casting *models.Casting, response *models.CastingResponse) {
	review := &models.Review{
		ModelID:    response.ModelID,
		EmployerID: casting.EmployerID,
		CastingID:  &casting.ID,
		Rating:     0,
		ReviewText: "",
		Status:     models.ReviewStatusPending,
	}

	if err := s.reviewRepo.CreateReview(review); err != nil {
		log.Printf("Failed to create review placeholder: %v", err)
	}
}

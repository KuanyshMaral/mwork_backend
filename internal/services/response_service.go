package services

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type ResponseService interface {
	CreateResponse(db *gorm.DB, modelID, castingID string, req *dto.CreateResponseRequest) (*models.CastingResponse, error)
	GetModelResponses(db *gorm.DB, modelID string) ([]models.CastingResponse, error)
	DeleteResponse(db *gorm.DB, modelID, responseID string) error
	GetCastingResponses(db *gorm.DB, castingID, employerID string) ([]dto.ResponseSummary, error)
	UpdateResponseStatus(db *gorm.DB, employerID, responseID string, status models.ResponseStatus) error
	MarkResponseAsViewed(db *gorm.DB, employerID, responseID string) error
	GetResponseStats(db *gorm.DB, castingID string) (*dto.CastingStatsResponse, error)
	GetResponse(db *gorm.DB, responseID, userID string) (*models.CastingResponse, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type ResponseServiceImpl struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	responseRepo     repositories.ResponseRepository
	castingRepo      repositories.CastingRepository
	userRepo         repositories.UserRepository
	subscriptionRepo repositories.SubscriptionRepository
	notificationRepo repositories.NotificationRepository
	reviewRepo       repositories.ReviewRepository
}

// ✅ Конструктор обновлен (db убран)
func NewResponseService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	responseRepo repositories.ResponseRepository,
	castingRepo repositories.CastingRepository,
	userRepo repositories.UserRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	notificationRepo repositories.NotificationRepository,
	reviewRepo repositories.ReviewRepository,
) ResponseService {
	return &ResponseServiceImpl{
		// ❌ 'db: db,' УДАЛЕНО
		responseRepo:     responseRepo,
		castingRepo:      castingRepo,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		notificationRepo: notificationRepo,
		reviewRepo:       reviewRepo,
	}
}

// Response Operations

// CreateResponse - 'db' добавлен
func (s *ResponseServiceImpl) CreateResponse(db *gorm.DB, modelID, castingID string, req *dto.CreateResponseRequest) (*models.CastingResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	model, err := s.userRepo.FindByID(tx, modelID)
	if err != nil {
		return nil, handleResponseError(err)
	}
	if model.Role != models.UserRoleModel {
		return nil, apperrors.ErrInsufficientPermissions
	}

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, castingID)
	if err != nil {
		return nil, handleResponseError(err)
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

	// ✅ Передаем tx
	existingResponse, _ := s.responseRepo.FindResponseByCastingAndModel(tx, castingID, modelID)
	if existingResponse != nil {
		return nil, errors.New("you have already responded to this casting")
	}

	// ✅ Передаем tx
	canRespond, err := s.subscriptionRepo.CanUserRespond(tx, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if !canRespond {
		return nil, errors.New("subscription limit reached")
	}

	// ✅ Передаем tx
	err = s.subscriptionRepo.IncrementSubscriptionUsage(tx, modelID, "responses")
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	response := &models.CastingResponse{
		CastingID: castingID,
		ModelID:   modelID,
		Message:   req.Message,
		Status:    models.ResponseStatusPending,
	}

	// ✅ Передаем tx
	err = s.responseRepo.CreateResponse(tx, response)
	if err != nil {
		if errors.Is(err, repositories.ErrResponseAlreadyExists) {
			return nil, errors.New("response already exists")
		}
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Отправляем уведомление *после* коммита, передаем 'db' (пул)
	go func() {
		// ✅ Передаем 'db'
		if err := s.notificationRepo.CreateNewResponseNotification(
			db,
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

// GetModelResponses - 'db' добавлен
func (s *ResponseServiceImpl) GetModelResponses(db *gorm.DB, modelID string) ([]models.CastingResponse, error) {
	// ✅ Используем 'db' из параметра
	responses, err := s.responseRepo.FindResponsesByModel(db, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return responses, nil
}

// DeleteResponse - 'db' добавлен
func (s *ResponseServiceImpl) DeleteResponse(db *gorm.DB, modelID, responseID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	response, err := s.responseRepo.FindResponseByID(tx, responseID)
	if err != nil {
		return handleResponseError(err)
	}

	if response.ModelID != modelID {
		return errors.New("access denied")
	}
	if response.Status != models.ResponseStatusPending {
		return errors.New("cannot delete response that has been reviewed")
	}

	// ✅ Передаем tx
	err = s.responseRepo.DeleteResponse(tx, responseID)
	if err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.DecrementSubscriptionUsage(tx, modelID, "responses"); err != nil {
		log.Printf("Failed to decrement subscription usage: %v", err)
	}

	// ✅ Коммитим транзакцию
	return tx.Commit().Error
}

// GetCastingResponses - 'db' добавлен
func (s *ResponseServiceImpl) GetCastingResponses(db *gorm.DB, castingID, employerID string) ([]dto.ResponseSummary, error) {
	// ✅ Используем 'db' из параметра
	casting, err := s.castingRepo.FindCastingByID(db, castingID)
	if err != nil {
		return nil, handleResponseError(err)
	}
	if casting.EmployerID != employerID {
		return nil, errors.New("access denied")
	}

	// ✅ Используем 'db' из параметра
	responses, err := s.responseRepo.FindResponsesByCasting(db, castingID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var summaries []dto.ResponseSummary
	for _, response := range responses {
		// ✅ Используем 'db' из параметра
		model, err := s.userRepo.FindByID(db, response.ModelID)
		modelName := ""
		if err == nil && model != nil {
			modelName = model.Email
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

// UpdateResponseStatus - 'db' добавлен
func (s *ResponseServiceImpl) UpdateResponseStatus(db *gorm.DB, employerID, responseID string, status models.ResponseStatus) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	response, err := s.responseRepo.FindResponseByID(tx, responseID)
	if err != nil {
		return handleResponseError(err)
	}

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, response.CastingID)
	if err != nil {
		return handleResponseError(err)
	}

	if casting.EmployerID != employerID {
		return errors.New("access denied")
	}

	oldStatus := response.Status
	// ✅ Передаем tx
	err = s.responseRepo.UpdateResponseStatus(tx, responseID, status)
	if err != nil {
		return apperrors.InternalError(err)
	}

	if oldStatus != status && status == models.ResponseStatusAccepted {
		// ✅ Передаем tx
		if err := s.createReviewPlaceholder(tx, casting, response); err != nil {
			log.Printf("Failed to create review placeholder: %v", err)
		}
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Отправляем уведомление *после* коммита
	if oldStatus != status {
		go func() {
			// ✅ Передаем 'db' (пул)
			model, err := s.userRepo.FindByID(db, response.ModelID)
			if err == nil && model != nil {
				// ✅ Передаем 'db' (пул)
				if err := s.notificationRepo.CreateResponseStatusNotification(
					db,
					response.ModelID,
					casting.Title,
					status,
				); err != nil {
					log.Printf("Failed to create status notification: %v", err)
				}
			}
		}()
	}

	return nil
}

// MarkResponseAsViewed - 'db' добавлен
func (s *ResponseServiceImpl) MarkResponseAsViewed(db *gorm.DB, employerID, responseID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	response, err := s.responseRepo.FindResponseByID(tx, responseID)
	if err != nil {
		return handleResponseError(err)
	}

	// ✅ Передаем tx
	casting, err := s.castingRepo.FindCastingByID(tx, response.CastingID)
	if err != nil {
		return handleResponseError(err)
	}

	if casting.EmployerID != employerID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	if err := s.responseRepo.MarkResponseAsViewed(tx, responseID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetResponseStats - 'db' добавлен
func (s *ResponseServiceImpl) GetResponseStats(db *gorm.DB, castingID string) (*dto.CastingStatsResponse, error) {
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

// GetResponse - 'db' добавлен
func (s *ResponseServiceImpl) GetResponse(db *gorm.DB, responseID, userID string) (*models.CastingResponse, error) {
	// ✅ Используем 'db' из параметра
	response, err := s.responseRepo.FindResponseByID(db, responseID)
	if err != nil {
		return nil, handleResponseError(err)
	}

	// ✅ Используем 'db' из параметра
	casting, err := s.castingRepo.FindCastingByID(db, response.CastingID)
	if err != nil {
		return nil, handleResponseError(err)
	}

	if response.ModelID != userID && casting.EmployerID != userID {
		return nil, errors.New("access denied")
	}

	return response, nil
}

// Helper Methods

// createReviewPlaceholder - (уже был 'db')
func (s *ResponseServiceImpl) createReviewPlaceholder(db *gorm.DB, casting *models.Casting, response *models.CastingResponse) error {
	review := &models.Review{
		ModelID:    response.ModelID,
		EmployerID: casting.EmployerID,
		CastingID:  &casting.ID,
		Rating:     0,
		ReviewText: "",
		Status:     models.ReviewStatusPending,
	}

	// ✅ Передаем db
	if err := s.reviewRepo.CreateReview(db, review); err != nil {
		log.Printf("Failed to create review placeholder: %v", err)
		return err
	}
	return nil
}

// (Вспомогательный хелпер для ошибок - без изменений)
func handleResponseError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrResponseNotFound) ||
		errors.Is(err, repositories.ErrCastingNotFound) ||
		errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrResponseAlreadyExists) {
		return apperrors.ErrAlreadyExists(err)
	}
	return apperrors.InternalError(err)
}

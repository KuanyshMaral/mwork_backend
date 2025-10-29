package repositories

import (
	"errors"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

var (
	ErrResponseNotFound      = errors.New("response not found")
	ErrResponseAlreadyExists = errors.New("response already exists for this casting and model")
)

type ResponseRepository interface {
	CreateResponse(response *models.CastingResponse) error
	FindResponseByID(id string) (*models.CastingResponse, error)
	FindResponseByCastingAndModel(castingID, modelID string) (*models.CastingResponse, error)
	FindResponsesByCasting(castingID string) ([]models.CastingResponse, error)
	FindResponsesByModel(modelID string) ([]models.CastingResponse, error)
	UpdateResponseStatus(responseID string, status models.ResponseStatus) error
	MarkResponseAsViewed(responseID string) error
	DeleteResponse(responseID string) error
	GetResponseStats(castingID string) (*ResponseStats, error)
}

type ResponseRepositoryImpl struct {
	db *gorm.DB
}

// Statistics for responses
type ResponseStats struct {
	TotalResponses    int64 `json:"total_responses"`
	PendingResponses  int64 `json:"pending_responses"`
	AcceptedResponses int64 `json:"accepted_responses"`
	RejectedResponses int64 `json:"rejected_responses"`
}

func NewResponseRepository(db *gorm.DB) ResponseRepository {
	return &ResponseRepositoryImpl{db: db}
}

// CastingResponse operations

func (r *ResponseRepositoryImpl) CreateResponse(response *models.CastingResponse) error {
	// Check if response already exists
	var existing models.CastingResponse
	if err := r.db.Where("casting_id = ? AND model_id = ?",
		response.CastingID, response.ModelID).First(&existing).Error; err == nil {
		return ErrResponseAlreadyExists
	}

	return r.db.Create(response).Error
}

func (r *ResponseRepositoryImpl) FindResponseByID(id string) (*models.CastingResponse, error) {
	var response models.CastingResponse
	err := r.db.Preload("Casting").Preload("Casting.Employer").Preload("Model").
		First(&response, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrResponseNotFound
		}
		return nil, err
	}
	return &response, nil
}

func (r *ResponseRepositoryImpl) FindResponseByCastingAndModel(castingID, modelID string) (*models.CastingResponse, error) {
	var response models.CastingResponse
	err := r.db.Preload("Casting").Preload("Model").
		Where("casting_id = ? AND model_id = ?", castingID, modelID).
		First(&response).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrResponseNotFound
		}
		return nil, err
	}
	return &response, nil
}

func (r *ResponseRepositoryImpl) FindResponsesByCasting(castingID string) ([]models.CastingResponse, error) {
	var responses []models.CastingResponse
	err := r.db.Preload("Model").Preload("Model.PortfolioItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index ASC").Limit(2).Preload("Upload")
	}).Where("casting_id = ?", castingID).
		Order("created_at DESC").
		Find(&responses).Error
	return responses, err
}

func (r *ResponseRepositoryImpl) FindResponsesByModel(modelID string) ([]models.CastingResponse, error) {
	var responses []models.CastingResponse
	err := r.db.Preload("Casting").Preload("Casting.Employer").
		Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&responses).Error
	return responses, err
}

func (r *ResponseRepositoryImpl) UpdateResponseStatus(id string, status models.ResponseStatus) error {
	result := r.db.Model(&models.CastingResponse{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResponseNotFound
	}
	return nil
}

func (r *ResponseRepositoryImpl) MarkResponseAsViewed(responseID string) error {
	result := r.db.Model(&models.CastingResponse{}).Where("id = ?", responseID).Updates(map[string]interface{}{
		"employer_viewed": true,
		"updated_at":      time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResponseNotFound
	}
	return nil
}

func (r *ResponseRepositoryImpl) UpdateResponseViewedByEmployer(responseID string, viewed bool) error {
	result := r.db.Model(&models.CastingResponse{}).Where("id = ?", responseID).Update("employer_viewed", viewed)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResponseNotFound
	}
	return nil
}

func (r *ResponseRepositoryImpl) DeleteResponse(id string) error {
	result := r.db.Where("id = ?", id).Delete(&models.CastingResponse{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResponseNotFound
	}
	return nil
}

func (r *ResponseRepositoryImpl) GetResponseStats(castingID string) (*ResponseStats, error) {
	var stats ResponseStats

	// Total responses
	if err := r.db.Model(&models.CastingResponse{}).Where("casting_id = ?", castingID).
		Count(&stats.TotalResponses).Error; err != nil {
		return nil, err
	}

	// Pending responses
	if err := r.db.Model(&models.CastingResponse{}).Where("casting_id = ? AND status = ?",
		castingID, models.ResponseStatusPending).Count(&stats.PendingResponses).Error; err != nil {
		return nil, err
	}

	// Accepted responses
	if err := r.db.Model(&models.CastingResponse{}).Where("casting_id = ? AND status = ?",
		castingID, models.ResponseStatusAccepted).Count(&stats.AcceptedResponses).Error; err != nil {
		return nil, err
	}

	// Rejected responses
	if err := r.db.Model(&models.CastingResponse{}).Where("casting_id = ? AND status = ?",
		castingID, models.ResponseStatusRejected).Count(&stats.RejectedResponses).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

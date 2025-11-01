package repositories

import (
	"errors"
	"mwork_backend/internal/models"
	"time"

	"gorm.io/gorm"
)

var (
	ErrResponseNotFound      = errors.New("response not found")
	ErrResponseAlreadyExists = errors.New("response already exists for this casting and model")
)

type ResponseRepository interface {
	CreateResponse(db *gorm.DB, response *models.CastingResponse) error
	FindResponseByID(db *gorm.DB, id string) (*models.CastingResponse, error)
	FindResponseByCastingAndModel(db *gorm.DB, castingID, modelID string) (*models.CastingResponse, error)
	FindResponsesByCasting(db *gorm.DB, castingID string) ([]models.CastingResponse, error)
	FindResponsesByModel(db *gorm.DB, modelID string) ([]models.CastingResponse, error)
	UpdateResponseStatus(db *gorm.DB, responseID string, status models.ResponseStatus) error
	MarkResponseAsViewed(db *gorm.DB, responseID string) error
	DeleteResponse(db *gorm.DB, responseID string) error
	GetResponseStats(db *gorm.DB, castingID string) (*ResponseStats, error)
	UpdateResponseViewedByEmployer(db *gorm.DB, responseID string, viewed bool) error
}

type ResponseRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
}

// Statistics for responses
type ResponseStats struct {
	TotalResponses    int64 `json:"total_responses"`
	PendingResponses  int64 `json:"pending_responses"`
	AcceptedResponses int64 `json:"accepted_responses"`
	RejectedResponses int64 `json:"rejected_responses"`
}

// ✅ Конструктор не принимает db
func NewResponseRepository() ResponseRepository {
	return &ResponseRepositoryImpl{}
}

// CastingResponse operations

func (r *ResponseRepositoryImpl) CreateResponse(db *gorm.DB, response *models.CastingResponse) error {
	// Check if response already exists
	var existing models.CastingResponse
	// ✅ Используем 'db' из параметра
	if err := db.Where("casting_id = ? AND model_id = ?",
		response.CastingID, response.ModelID).First(&existing).Error; err == nil {
		return ErrResponseAlreadyExists
	}
	// ✅ Используем 'db' из параметра
	return db.Create(response).Error
}

func (r *ResponseRepositoryImpl) FindResponseByID(db *gorm.DB, id string) (*models.CastingResponse, error) {
	var response models.CastingResponse
	// ✅ Используем 'db' из параметра
	err := db.Preload("Casting").Preload("Casting.Employer").Preload("Model").
		First(&response, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrResponseNotFound
		}
		return nil, err
	}
	return &response, nil
}

func (r *ResponseRepositoryImpl) FindResponseByCastingAndModel(db *gorm.DB, castingID, modelID string) (*models.CastingResponse, error) {
	var response models.CastingResponse
	// ✅ Используем 'db' из параметра
	err := db.Preload("Casting").Preload("Model").
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

func (r *ResponseRepositoryImpl) FindResponsesByCasting(db *gorm.DB, castingID string) ([]models.CastingResponse, error) {
	var responses []models.CastingResponse
	// ✅ Используем 'db' из параметра
	err := db.Preload("Model").Preload("Model.PortfolioItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index ASC").Limit(2).Preload("Upload")
	}).Where("casting_id = ?", castingID).
		Order("created_at DESC").
		Find(&responses).Error
	return responses, err
}

func (r *ResponseRepositoryImpl) FindResponsesByModel(db *gorm.DB, modelID string) ([]models.CastingResponse, error) {
	var responses []models.CastingResponse
	// ✅ Используем 'db' из параметра
	err := db.Preload("Casting").Preload("Casting.Employer").
		Where("model_id = ?", modelID).
		Order("created_at DESC").
		Find(&responses).Error
	return responses, err
}

func (r *ResponseRepositoryImpl) UpdateResponseStatus(db *gorm.DB, id string, status models.ResponseStatus) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.CastingResponse{}).Where("id = ?", id).Updates(map[string]interface{}{
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

func (r *ResponseRepositoryImpl) MarkResponseAsViewed(db *gorm.DB, responseID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.CastingResponse{}).Where("id = ?", responseID).Updates(map[string]interface{}{
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

func (r *ResponseRepositoryImpl) UpdateResponseViewedByEmployer(db *gorm.DB, responseID string, viewed bool) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.CastingResponse{}).Where("id = ?", responseID).Update("employer_viewed", viewed)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResponseNotFound
	}
	return nil
}

func (r *ResponseRepositoryImpl) DeleteResponse(db *gorm.DB, id string) error {
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.CastingResponse{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResponseNotFound
	}
	return nil
}

func (r *ResponseRepositoryImpl) GetResponseStats(db *gorm.DB, castingID string) (*ResponseStats, error) {
	var stats ResponseStats

	// Total responses
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.CastingResponse{}).Where("casting_id = ?", castingID).
		Count(&stats.TotalResponses).Error; err != nil {
		return nil, err
	}

	// Pending responses
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.CastingResponse{}).Where("casting_id = ? AND status = ?",
		castingID, models.ResponseStatusPending).Count(&stats.PendingResponses).Error; err != nil {
		return nil, err
	}

	// Accepted responses
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.CastingResponse{}).Where("casting_id = ? AND status = ?",
		castingID, models.ResponseStatusAccepted).Count(&stats.AcceptedResponses).Error; err != nil {
		return nil, err
	}

	// Rejected responses
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.CastingResponse{}).Where("casting_id = ? AND status = ?",
		castingID, models.ResponseStatusRejected).Count(&stats.RejectedResponses).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

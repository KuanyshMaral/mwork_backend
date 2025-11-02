package repositories

import (
	"errors"
	"mwork_backend/internal/models"
	"time"

	"gorm.io/gorm"
)

var (
	ErrPortfolioItemNotFound = errors.New("portfolio item not found")
	ErrInvalidPortfolioOrder = errors.New("invalid portfolio order")
	// ▼▼▼ УДАЛЕНО ▼▼▼
	// ErrUploadNotFound        = errors.New("upload not found")
)

type PortfolioRepository interface {
	// PortfolioItem operations
	CreatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error
	FindPortfolioItemByID(db *gorm.DB, id string) (*models.PortfolioItem, error)
	FindPortfolioByModel(db *gorm.DB, modelID string) ([]models.PortfolioItem, error)
	FindFeaturedPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error)
	FindRecentPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error)
	UpdatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error
	UpdatePortfolioItemOrder(db *gorm.DB, item *models.PortfolioItem, newOrder int) error
	DeletePortfolioItem(db *gorm.DB, id string) error
	ReorderPortfolioItems(db *gorm.DB, modelID string, itemIDs []string) error
	GetPortfolioStats(db *gorm.DB, modelID string) (*PortfolioStats, error)

	// ▼▼▼ УДАЛЕНО: Все операции Upload теперь в UploadRepository ▼▼▼
	// CreateUpload(db *gorm.DB, upload *models.Upload) error
	// FindUploadByID(db *gorm.DB, id string) (*models.Upload, error)
	// FindUploadsByEntity(db *gorm.DB, entityType, entityID string) ([]models.Upload, error)
	// FindUploadsByUser(db *gorm.DB, userID string) ([]models.Upload, error)
	// FindUploadsByUsage(db *gorm.DB, userID, usage string) ([]models.Upload, error)
	// UpdateUpload(db *gorm.DB, upload *models.Upload) error
	// DeleteUpload(db *gorm.DB, id string) error
	// CleanOrphanedUploads(db *gorm.DB) error
	// GetUserStorageUsage(db *gorm.DB, userID string) (int64, error)
	// ▲▲▲ УДАЛЕНО ▲▲▲

	// ▼▼▼ УДАЛЕНО: Это логика сервисного уровня, а не репозитория ▼▼▼
	// CreatePortfolioWithUpload(db *gorm.DB, modelID string, item *models.PortfolioItem, upload *models.Upload) error
	// DeletePortfolioItemWithUpload(db *gorm.DB, itemID string) error
	// ▲▲▲ УДАЛЕНО ▲▲▲

	// GetModelPortfolioWithUploads (переименовано из-за удаления дубликата FindPortfolioByModel)
	GetModelPortfolioWithUploads(db *gorm.DB, modelID string) ([]models.PortfolioItem, error)

	// Additional methods
	FindPortfolioItemsByFileType(db *gorm.DB, modelID, fileType string) ([]models.PortfolioItem, error)
	UpdatePortfolioItemVisibility(db *gorm.DB, itemID string, isPublic bool) error
}

type PortfolioRepositoryImpl struct {
	// ✅ Пусто!
}

// Statistics for portfolio
type PortfolioStats struct {
	TotalItems     int64     `json:"total_items"`
	PhotosCount    int64     `json:"photos_count"`
	VideosCount    int64     `json:"videos_count"`
	DocumentsCount int64     `json:"documents_count"`
	TotalSize      int64     `json:"total_size"` // in bytes
	LastUpdated    time.Time `json:"last_updated"`
}

func NewPortfolioRepository() PortfolioRepository {
	return &PortfolioRepositoryImpl{}
}

// PortfolioItem operations

func (r *PortfolioRepositoryImpl) CreatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error {
	// (Логика CreatePortfolioItem без изменений)
	if item.OrderIndex == 0 {
		var maxOrder int
		db.Model(&models.PortfolioItem{}).Where("model_id = ?", item.ModelID).
			Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		item.OrderIndex = maxOrder + 1
	}
	return db.Create(item).Error
}

func (r *PortfolioRepositoryImpl) FindPortfolioItemByID(db *gorm.DB, id string) (*models.PortfolioItem, error) {
	// (Логика FindPortfolioItemByID без изменений)
	var item models.PortfolioItem
	err := db.Preload("Upload").First(&item, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPortfolioItemNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *PortfolioRepositoryImpl) FindPortfolioByModel(db *gorm.DB, modelID string) ([]models.PortfolioItem, error) {
	// (Логика FindPortfolioByModel без изменений)
	var items []models.PortfolioItem
	err := db.Preload("Upload").Where("model_id = ?", modelID).
		Order("order_index ASC").Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) FindFeaturedPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error) {
	// (Логика FindFeaturedPortfolioItems без изменений)
	var items []models.PortfolioItem
	err := db.Preload("Upload").Preload("Model").
		Joins("LEFT JOIN model_profiles mp ON portfolio_items.model_id = mp.id").
		Where("mp.rating >= ? AND mp.is_public = ?", 4.0, true).
		Order("mp.rating DESC, portfolio_items.order_index ASC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error {
	// (Логика UpdatePortfolioItem без изменений)
	result := db.Model(item).Updates(map[string]interface{}{
		"title":       item.Title,
		"description": item.Description,
		"order_index": item.OrderIndex,
		"updated_at":  time.Now(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPortfolioItemNotFound
	}
	return nil
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItemOrder(db *gorm.DB, item *models.PortfolioItem, newOrder int) error {
	// (Логика UpdatePortfolioItemOrder без изменений)
	var currentOrder int
	if err := db.Model(&models.PortfolioItem{}).Where("id = ?", item.ID).
		Select("order_index").Scan(&currentOrder).Error; err != nil {
		return err
	}
	if currentOrder == newOrder {
		return nil
	}
	if newOrder > currentOrder {
		if err := db.Model(&models.PortfolioItem{}).
			Where("model_id = ? AND order_index > ? AND order_index <= ?",
				item.ModelID, currentOrder, newOrder).
			Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
			return err
		}
	} else {
		if err := db.Model(&models.PortfolioItem{}).
			Where("model_id = ? AND order_index >= ? AND order_index < ?",
				item.ModelID, newOrder, currentOrder).
			Update("order_index", gorm.Expr("order_index + ?", 1)).Error; err != nil {
			return err
		}
	}
	if err := db.Model(item).Update("order_index", newOrder).Error; err != nil {
		return err
	}
	return nil
}

func (r *PortfolioRepositoryImpl) DeletePortfolioItem(db *gorm.DB, id string) error {
	// (Логика DeletePortfolioItem без изменений)
	// Эта функция ТЕПЕРЬ отвечает ТОЛЬКО за удаление PortfolioItem.
	// Сервисный слой должен будет отдельно вызвать UploadService.DeleteUpload(uploadID)
	var item models.PortfolioItem
	if err := db.First(&item, "id = ?", id).Error; err != nil {
		return ErrPortfolioItemNotFound
	}
	if err := db.Delete(&item).Error; err != nil {
		return err
	}
	if err := db.Model(&models.PortfolioItem{}).
		Where("model_id = ? AND order_index > ?", item.ModelID, item.OrderIndex).
		Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
		return err
	}
	return nil
}

func (r *PortfolioRepositoryImpl) ReorderPortfolioItems(db *gorm.DB, modelID string, itemIDs []string) error {
	// (Логика ReorderPortfolioItems без изменений)
	for order, itemID := range itemIDs {
		if err := db.Model(&models.PortfolioItem{}).
			Where("id = ? AND model_id = ?", itemID, modelID).
			Update("order_index", order+1).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *PortfolioRepositoryImpl) GetPortfolioStats(db *gorm.DB, modelID string) (*PortfolioStats, error) {
	// (Логика GetPortfolioStats без изменений)
	var stats PortfolioStats
	if err := db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Count(&stats.TotalItems).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "image").
		Count(&stats.PhotosCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "video").
		Count(&stats.VideosCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "document").
		Count(&stats.DocumentsCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Select("COALESCE(SUM(uploads.size), 0)").Scan(&stats.TotalSize).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Select("COALESCE(MAX(updated_at), MAX(created_at))").Scan(&stats.LastUpdated).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

// ▼▼▼ УДАЛЕНО: Все реализации Upload теперь в UploadRepository ▼▼▼
// func (r *PortfolioRepositoryImpl) CreateUpload...
// func (r *PortfolioRepositoryImpl) FindUploadByID...
// func (r *PortfolioRepositoryImpl) FindUploadsByEntity...
// func (r *PortfolioRepositoryImpl) FindUploadsByUser...
// func (r *PortfolioRepositoryImpl) FindUploadsByUsage...
// func (r *PortfolioRepositoryImpl) UpdateUpload...
// func (r *PortfolioRepositoryImpl) DeleteUpload...
// func (r *PortfolioRepositoryImpl) CleanOrphanedUploads...
// func (r *PortfolioRepositoryImpl) GetUserStorageUsage...
// ▲▲▲ УДАЛЕНО ▲▲▲

// ▼▼▼ УДАЛЕНО: Это логика сервисного уровня, а не репозитория ▼▼▼
// func (r *PortfolioRepositoryImpl) CreatePortfolioWithUpload...
// func (r *PortfolioRepositoryImpl) DeletePortfolioItemWithUpload...
// ▲▲▲ УДАЛЕНО ▲▲▲

func (r *PortfolioRepositoryImpl) GetModelPortfolioWithUploads(db *gorm.DB, modelID string) ([]models.PortfolioItem, error) {
	// (Логика GetModelPortfolioWithUploads без изменений)
	var items []models.PortfolioItem
	err := db.Preload("Upload").Where("model_id = ?", modelID).
		Order("order_index ASC").Find(&items).Error
	return items, err
}

// Additional methods

func (r *PortfolioRepositoryImpl) FindPortfolioItemsByFileType(db *gorm.DB, modelID, fileType string) ([]models.PortfolioItem, error) {
	// (Логика FindPortfolioItemsByFileType без изменений)
	var items []models.PortfolioItem
	err := db.Preload("Upload").Where("model_id = ?", modelID).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("uploads.file_type = ?", fileType).
		Order("portfolio_items.order_index ASC").
		Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItemVisibility(db *gorm.DB, itemID string, isPublic bool) error {
	// (Логика UpdatePortfolioItemVisibility без изменений)
	result := db.Model(&models.PortfolioItem{}).Where("id = ?", itemID).
		Update("is_public", isPublic)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPortfolioItemNotFound
	}
	return nil
}

func (r *PortfolioRepositoryImpl) FindRecentPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error) {
	// (Логика FindRecentPortfolioItems без изменений)
	var items []models.PortfolioItem
	err := db.Preload("Upload").Preload("Model").
		Order("portfolio_items.created_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

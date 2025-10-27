package repositories

import (
	"errors"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/gorm"
)

var (
	ErrPortfolioItemNotFound = errors.New("portfolio item not found")
	ErrUploadNotFound        = errors.New("upload not found")
	ErrInvalidPortfolioOrder = errors.New("invalid portfolio order")
)

type PortfolioRepository interface {
	// PortfolioItem operations
	CreatePortfolioItem(item *models.PortfolioItem) error
	FindPortfolioItemByID(id string) (*models.PortfolioItem, error)
	FindPortfolioByModel(modelID string) ([]models.PortfolioItem, error)
	FindFeaturedPortfolioItems(limit int) ([]models.PortfolioItem, error)
	FindRecentPortfolioItems(limit int) ([]models.PortfolioItem, error) // ДОБАВЛЕНО
	UpdatePortfolioItem(item *models.PortfolioItem) error
	UpdatePortfolioItemOrder(item *models.PortfolioItem, newOrder int) error
	DeletePortfolioItem(id string) error
	ReorderPortfolioItems(modelID string, itemIDs []string) error
	GetPortfolioStats(modelID string) (*PortfolioStats, error)

	// Upload operations
	CreateUpload(upload *models.Upload) error
	FindUploadByID(id string) (*models.Upload, error)
	FindUploadsByEntity(entityType, entityID string) ([]models.Upload, error)
	FindUploadsByUser(userID string) ([]models.Upload, error)
	FindUploadsByUsage(userID, usage string) ([]models.Upload, error)
	UpdateUpload(upload *models.Upload) error
	DeleteUpload(id string) error
	CleanOrphanedUploads() error
	GetUserStorageUsage(userID string) (int64, error) // ДОБАВЛЕНО

	// Combined operations
	CreatePortfolioWithUpload(modelID string, item *models.PortfolioItem, upload *models.Upload) error
	DeletePortfolioItemWithUpload(itemID string) error
	GetModelPortfolioWithUploads(modelID string) ([]models.PortfolioItem, error)

	// Additional methods
	FindPortfolioItemsByFileType(modelID, fileType string) ([]models.PortfolioItem, error)
	UpdatePortfolioItemVisibility(itemID string, isPublic bool) error
}

type PortfolioRepositoryImpl struct {
	db *gorm.DB
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

func NewPortfolioRepository(db *gorm.DB) PortfolioRepository {
	return &PortfolioRepositoryImpl{db: db}
}

// PortfolioItem operations

func (r *PortfolioRepositoryImpl) CreatePortfolioItem(item *models.PortfolioItem) error {
	// Set order index to last position if not provided
	if item.OrderIndex == 0 {
		var maxOrder int
		r.db.Model(&models.PortfolioItem{}).Where("model_id = ?", item.ModelID).
			Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		item.OrderIndex = maxOrder + 1
	}

	return r.db.Create(item).Error
}

func (r *PortfolioRepositoryImpl) FindPortfolioItemByID(id string) (*models.PortfolioItem, error) {
	var item models.PortfolioItem
	err := r.db.Preload("Upload").First(&item, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPortfolioItemNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *PortfolioRepositoryImpl) FindPortfolioByModel(modelID string) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	err := r.db.Preload("Upload").Where("model_id = ?", modelID).
		Order("order_index ASC").Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) FindFeaturedPortfolioItems(limit int) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem

	// Находим портфолио моделей с высоким рейтингом
	err := r.db.Preload("Upload").Preload("Model").
		Joins("LEFT JOIN model_profiles mp ON portfolio_items.model_id = mp.id").
		Where("mp.rating >= ? AND mp.is_public = ?", 4.0, true).
		Order("mp.rating DESC, portfolio_items.order_index ASC").
		Limit(limit).
		Find(&items).Error

	return items, err
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItem(item *models.PortfolioItem) error {
	result := r.db.Model(item).Updates(map[string]interface{}{
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

func (r *PortfolioRepositoryImpl) UpdatePortfolioItemOrder(item *models.PortfolioItem, newOrder int) error {
	// Используем транзакцию для атомарного обновления порядка
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Получаем текущий порядок
		var currentOrder int
		if err := tx.Model(&models.PortfolioItem{}).Where("id = ?", item.ID).
			Select("order_index").Scan(&currentOrder).Error; err != nil {
			return err
		}

		if currentOrder == newOrder {
			return nil // порядок не изменился
		}

		// Обновляем порядок других элементов
		if newOrder > currentOrder {
			// Сдвигаем элементы вниз
			if err := tx.Model(&models.PortfolioItem{}).
				Where("model_id = ? AND order_index > ? AND order_index <= ?",
					item.ModelID, currentOrder, newOrder).
				Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
				return err
			}
		} else {
			// Сдвигаем элементы вверх
			if err := tx.Model(&models.PortfolioItem{}).
				Where("model_id = ? AND order_index >= ? AND order_index < ?",
					item.ModelID, newOrder, currentOrder).
				Update("order_index", gorm.Expr("order_index + ?", 1)).Error; err != nil {
				return err
			}
		}

		// Обновляем порядок текущего элемента
		if err := tx.Model(item).Update("order_index", newOrder).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *PortfolioRepositoryImpl) DeletePortfolioItem(id string) error {
	// Используем транзакцию для удаления элемента и обновления порядка
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Получаем информацию об элементе
		var item models.PortfolioItem
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return ErrPortfolioItemNotFound
		}

		// Удаляем элемент
		if err := tx.Delete(&item).Error; err != nil {
			return err
		}

		// Обновляем порядок оставшихся элементов
		if err := tx.Model(&models.PortfolioItem{}).
			Where("model_id = ? AND order_index > ?", item.ModelID, item.OrderIndex).
			Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *PortfolioRepositoryImpl) ReorderPortfolioItems(modelID string, itemIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for order, itemID := range itemIDs {
			if err := tx.Model(&models.PortfolioItem{}).
				Where("id = ? AND model_id = ?", itemID, modelID).
				Update("order_index", order+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PortfolioRepositoryImpl) GetPortfolioStats(modelID string) (*PortfolioStats, error) {
	var stats PortfolioStats

	// Total items
	if err := r.db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Count(&stats.TotalItems).Error; err != nil {
		return nil, err
	}

	// Count by file type through uploads
	if err := r.db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "image").
		Count(&stats.PhotosCount).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "video").
		Count(&stats.VideosCount).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "document").
		Count(&stats.DocumentsCount).Error; err != nil {
		return nil, err
	}

	// Total size
	if err := r.db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Select("COALESCE(SUM(uploads.size), 0)").Scan(&stats.TotalSize).Error; err != nil {
		return nil, err
	}

	// Last updated
	if err := r.db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Select("COALESCE(MAX(updated_at), MAX(created_at))").Scan(&stats.LastUpdated).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Upload operations

func (r *PortfolioRepositoryImpl) CreateUpload(upload *models.Upload) error {
	return r.db.Create(upload).Error
}

func (r *PortfolioRepositoryImpl) FindUploadByID(id string) (*models.Upload, error) {
	var upload models.Upload
	err := r.db.First(&upload, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUploadNotFound
		}
		return nil, err
	}
	return &upload, nil
}

func (r *PortfolioRepositoryImpl) FindUploadsByEntity(entityType, entityID string) ([]models.Upload, error) {
	var uploads []models.Upload
	err := r.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

func (r *PortfolioRepositoryImpl) FindUploadsByUser(userID string) ([]models.Upload, error) {
	var uploads []models.Upload
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

func (r *PortfolioRepositoryImpl) FindUploadsByUsage(userID, usage string) ([]models.Upload, error) {
	var uploads []models.Upload
	query := r.db.Where("user_id = ?", userID)

	if usage != "" {
		query = query.Where("usage = ?", usage)
	}

	err := query.Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

func (r *PortfolioRepositoryImpl) UpdateUpload(upload *models.Upload) error {
	result := r.db.Model(upload).Updates(map[string]interface{}{
		"entity_type": upload.EntityType,
		"entity_id":   upload.EntityID,
		"file_type":   upload.FileType,
		"usage":       upload.Usage,
		"path":        upload.Path,
		"mime_type":   upload.MimeType,
		"size":        upload.Size,
		"is_public":   upload.IsPublic,
		"updated_at":  time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUploadNotFound
	}
	return nil
}

func (r *PortfolioRepositoryImpl) DeleteUpload(id string) error {
	// Проверяем, не используется ли upload в портфолио
	var portfolioCount int64
	if err := r.db.Model(&models.PortfolioItem{}).Where("upload_id = ?", id).
		Count(&portfolioCount).Error; err != nil {
		return err
	}

	if portfolioCount > 0 {
		return errors.New("cannot delete upload that is used in portfolio")
	}

	result := r.db.Where("id = ?", id).Delete(&models.Upload{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUploadNotFound
	}
	return nil
}

func (r *PortfolioRepositoryImpl) CleanOrphanedUploads() error {
	// Находим uploads, которые не связаны с портфолио и созданы больше суток назад
	dayAgo := time.Now().AddDate(0, 0, -1)

	return r.db.Where("id NOT IN (SELECT DISTINCT upload_id FROM portfolio_items WHERE upload_id IS NOT NULL) AND created_at < ?", dayAgo).
		Delete(&models.Upload{}).Error
}

// Combined operations

func (r *PortfolioRepositoryImpl) CreatePortfolioWithUpload(modelID string, item *models.PortfolioItem, upload *models.Upload) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Создаем upload
		if err := tx.Create(upload).Error; err != nil {
			return err
		}

		// Создаем portfolio item с ссылкой на upload
		item.UploadID = upload.ID
		item.ModelID = modelID

		// Устанавливаем порядок
		var maxOrder int
		tx.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
			Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		item.OrderIndex = maxOrder + 1

		if err := tx.Create(item).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *PortfolioRepositoryImpl) DeletePortfolioItemWithUpload(itemID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Находим portfolio item
		var item models.PortfolioItem
		if err := tx.Preload("Upload").First(&item, "id = ?", itemID).Error; err != nil {
			return ErrPortfolioItemNotFound
		}

		// Удаляем portfolio item
		if err := tx.Delete(&item).Error; err != nil {
			return err
		}

		// Обновляем порядок оставшихся элементов
		if err := tx.Model(&models.PortfolioItem{}).
			Where("model_id = ? AND order_index > ?", item.ModelID, item.OrderIndex).
			Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
			return err
		}

		// Удаляем связанный upload
		if item.Upload != nil {
			if err := tx.Delete(item.Upload).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *PortfolioRepositoryImpl) GetModelPortfolioWithUploads(modelID string) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	err := r.db.Preload("Upload").Where("model_id = ?", modelID).
		Order("order_index ASC").Find(&items).Error
	return items, err
}

// Additional methods for specific use cases

func (r *PortfolioRepositoryImpl) FindPortfolioItemsByFileType(modelID, fileType string) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	err := r.db.Preload("Upload").Where("model_id = ?", modelID).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("uploads.file_type = ?", fileType).
		Order("portfolio_items.order_index ASC").
		Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItemVisibility(itemID string, isPublic bool) error {
	result := r.db.Model(&models.PortfolioItem{}).Where("id = ?", itemID).
		Update("is_public", isPublic)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPortfolioItemNotFound
	}
	return nil
}

func (r *PortfolioRepositoryImpl) GetUserStorageUsage(userID string) (int64, error) {
	var totalSize int64
	err := r.db.Model(&models.Upload{}).Where("user_id = ?", userID).
		Select("COALESCE(SUM(size), 0)").Scan(&totalSize).Error
	return totalSize, err
}

func (r *PortfolioRepositoryImpl) FindRecentPortfolioItems(limit int) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	err := r.db.Preload("Upload").Preload("Model").
		Order("portfolio_items.created_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

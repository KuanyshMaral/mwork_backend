package repositories

import (
	"errors"
	"mwork_backend/internal/models"
	"time"

	"gorm.io/gorm"
)

var (
	ErrPortfolioItemNotFound = errors.New("portfolio item not found")
	ErrUploadNotFound        = errors.New("upload not found")
	ErrInvalidPortfolioOrder = errors.New("invalid portfolio order")
)

type PortfolioRepository interface {
	// PortfolioItem operations
	CreatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error
	FindPortfolioItemByID(db *gorm.DB, id string) (*models.PortfolioItem, error)
	FindPortfolioByModel(db *gorm.DB, modelID string) ([]models.PortfolioItem, error)
	FindFeaturedPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error)
	FindRecentPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error) // ДОБАВЛЕНО
	UpdatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error
	UpdatePortfolioItemOrder(db *gorm.DB, item *models.PortfolioItem, newOrder int) error
	DeletePortfolioItem(db *gorm.DB, id string) error
	ReorderPortfolioItems(db *gorm.DB, modelID string, itemIDs []string) error
	GetPortfolioStats(db *gorm.DB, modelID string) (*PortfolioStats, error)

	// Upload operations
	CreateUpload(db *gorm.DB, upload *models.Upload) error
	FindUploadByID(db *gorm.DB, id string) (*models.Upload, error)
	FindUploadsByEntity(db *gorm.DB, entityType, entityID string) ([]models.Upload, error)
	FindUploadsByUser(db *gorm.DB, userID string) ([]models.Upload, error)
	FindUploadsByUsage(db *gorm.DB, userID, usage string) ([]models.Upload, error)
	UpdateUpload(db *gorm.DB, upload *models.Upload) error
	DeleteUpload(db *gorm.DB, id string) error
	CleanOrphanedUploads(db *gorm.DB) error
	GetUserStorageUsage(db *gorm.DB, userID string) (int64, error) // ДОБАВЛЕНО

	// Combined operations
	CreatePortfolioWithUpload(db *gorm.DB, modelID string, item *models.PortfolioItem, upload *models.Upload) error
	DeletePortfolioItemWithUpload(db *gorm.DB, itemID string) error
	GetModelPortfolioWithUploads(db *gorm.DB, modelID string) ([]models.PortfolioItem, error)

	// Additional methods
	FindPortfolioItemsByFileType(db *gorm.DB, modelID, fileType string) ([]models.PortfolioItem, error)
	UpdatePortfolioItemVisibility(db *gorm.DB, itemID string, isPublic bool) error
}

type PortfolioRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
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

// ✅ Конструктор не принимает db
func NewPortfolioRepository() PortfolioRepository {
	return &PortfolioRepositoryImpl{}
}

// PortfolioItem operations

func (r *PortfolioRepositoryImpl) CreatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error {
	// Set order index to last position if not provided
	if item.OrderIndex == 0 {
		var maxOrder int
		// ✅ Используем 'db' из параметра
		db.Model(&models.PortfolioItem{}).Where("model_id = ?", item.ModelID).
			Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		item.OrderIndex = maxOrder + 1
	}
	// ✅ Используем 'db' из параметра
	return db.Create(item).Error
}

func (r *PortfolioRepositoryImpl) FindPortfolioItemByID(db *gorm.DB, id string) (*models.PortfolioItem, error) {
	var item models.PortfolioItem
	// ✅ Используем 'db' из параметра
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
	var items []models.PortfolioItem
	// ✅ Используем 'db' из параметра
	err := db.Preload("Upload").Where("model_id = ?", modelID).
		Order("order_index ASC").Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) FindFeaturedPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem

	// Находим портфолио моделей с высоким рейтингом
	// ✅ Используем 'db' из параметра
	err := db.Preload("Upload").Preload("Model").
		Joins("LEFT JOIN model_profiles mp ON portfolio_items.model_id = mp.id").
		Where("mp.rating >= ? AND mp.is_public = ?", 4.0, true).
		Order("mp.rating DESC, portfolio_items.order_index ASC").
		Limit(limit).
		Find(&items).Error

	return items, err
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItem(db *gorm.DB, item *models.PortfolioItem) error {
	// ✅ Используем 'db' из параметра
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
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	// Получаем текущий порядок
	var currentOrder int
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).Where("id = ?", item.ID).
		Select("order_index").Scan(&currentOrder).Error; err != nil {
		return err
	}

	if currentOrder == newOrder {
		return nil // порядок не изменился
	}

	// Обновляем порядок других элементов
	if newOrder > currentOrder {
		// Сдвигаем элементы вниз
		// ✅ Используем 'db' из параметра
		if err := db.Model(&models.PortfolioItem{}).
			Where("model_id = ? AND order_index > ? AND order_index <= ?",
				item.ModelID, currentOrder, newOrder).
			Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
			return err
		}
	} else {
		// Сдвигаем элементы вверх
		// ✅ Используем 'db' из параметра
		if err := db.Model(&models.PortfolioItem{}).
			Where("model_id = ? AND order_index >= ? AND order_index < ?",
				item.ModelID, newOrder, currentOrder).
			Update("order_index", gorm.Expr("order_index + ?", 1)).Error; err != nil {
			return err
		}
	}

	// Обновляем порядок текущего элемента
	// ✅ Используем 'db' из параметра
	if err := db.Model(item).Update("order_index", newOrder).Error; err != nil {
		return err
	}

	return nil
}

func (r *PortfolioRepositoryImpl) DeletePortfolioItem(db *gorm.DB, id string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	// Получаем информацию об элементе
	var item models.PortfolioItem
	// ✅ Используем 'db' из параметра
	if err := db.First(&item, "id = ?", id).Error; err != nil {
		return ErrPortfolioItemNotFound
	}

	// Удаляем элемент
	// ✅ Используем 'db' из параметра
	if err := db.Delete(&item).Error; err != nil {
		return err
	}

	// Обновляем порядок оставшихся элементов
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).
		Where("model_id = ? AND order_index > ?", item.ModelID, item.OrderIndex).
		Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

func (r *PortfolioRepositoryImpl) ReorderPortfolioItems(db *gorm.DB, modelID string, itemIDs []string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	for order, itemID := range itemIDs {
		// ✅ Используем 'db' из параметра
		if err := db.Model(&models.PortfolioItem{}).
			Where("id = ? AND model_id = ?", itemID, modelID).
			Update("order_index", order+1).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *PortfolioRepositoryImpl) GetPortfolioStats(db *gorm.DB, modelID string) (*PortfolioStats, error) {
	var stats PortfolioStats

	// Total items
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Count(&stats.TotalItems).Error; err != nil {
		return nil, err
	}

	// Count by file type through uploads
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "image").
		Count(&stats.PhotosCount).Error; err != nil {
		return nil, err
	}
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "video").
		Count(&stats.VideosCount).Error; err != nil {
		return nil, err
	}
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Where("uploads.file_type = ?", "document").
		Count(&stats.DocumentsCount).Error; err != nil {
		return nil, err
	}

	// Total size
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("portfolio_items.model_id = ?", modelID).
		Select("COALESCE(SUM(uploads.size), 0)").Scan(&stats.TotalSize).Error; err != nil {
		return nil, err
	}

	// Last updated
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Select("COALESCE(MAX(updated_at), MAX(created_at))").Scan(&stats.LastUpdated).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// Upload operations

func (r *PortfolioRepositoryImpl) CreateUpload(db *gorm.DB, upload *models.Upload) error {
	// ✅ Используем 'db' из параметра
	return db.Create(upload).Error
}

func (r *PortfolioRepositoryImpl) FindUploadByID(db *gorm.DB, id string) (*models.Upload, error) {
	var upload models.Upload
	// ✅ Используем 'db' из параметра
	err := db.First(&upload, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUploadNotFound
		}
		return nil, err
	}
	return &upload, nil
}

func (r *PortfolioRepositoryImpl) FindUploadsByEntity(db *gorm.DB, entityType, entityID string) ([]models.Upload, error) {
	var uploads []models.Upload
	// ✅ Используем 'db' из параметра
	err := db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

func (r *PortfolioRepositoryImpl) FindUploadsByUser(db *gorm.DB, userID string) ([]models.Upload, error) {
	var uploads []models.Upload
	// ✅ Используем 'db' из параметра
	err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

func (r *PortfolioRepositoryImpl) FindUploadsByUsage(db *gorm.DB, userID, usage string) ([]models.Upload, error) {
	var uploads []models.Upload
	// ✅ Используем 'db' из параметра
	query := db.Where("user_id = ?", userID)

	if usage != "" {
		query = query.Where("usage = ?", usage)
	}

	err := query.Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

func (r *PortfolioRepositoryImpl) UpdateUpload(db *gorm.DB, upload *models.Upload) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(upload).Updates(map[string]interface{}{
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

func (r *PortfolioRepositoryImpl) DeleteUpload(db *gorm.DB, id string) error {
	// Проверяем, не используется ли upload в портфолио
	var portfolioCount int64
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).Where("upload_id = ?", id).
		Count(&portfolioCount).Error; err != nil {
		return err
	}

	if portfolioCount > 0 {
		return errors.New("cannot delete upload that is used in portfolio")
	}
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.Upload{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUploadNotFound
	}
	return nil
}

func (r *PortfolioRepositoryImpl) CleanOrphanedUploads(db *gorm.DB) error {
	// Находим uploads, которые не связаны с портфолио и созданы больше суток назад
	dayAgo := time.Now().AddDate(0, 0, -1)
	// ✅ Используем 'db' из параметра
	return db.Where("id NOT IN (SELECT DISTINCT upload_id FROM portfolio_items WHERE upload_id IS NOT NULL) AND created_at < ?", dayAgo).
		Delete(&models.Upload{}).Error
}

// Combined operations

func (r *PortfolioRepositoryImpl) CreatePortfolioWithUpload(db *gorm.DB, modelID string, item *models.PortfolioItem, upload *models.Upload) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	// Создаем upload
	// ✅ Используем 'db' из параметра
	if err := db.Create(upload).Error; err != nil {
		return err
	}

	// Создаем portfolio item с ссылкой на upload
	item.UploadID = upload.ID
	item.ModelID = modelID

	// Устанавливаем порядок
	var maxOrder int
	// ✅ Используем 'db' из параметра
	db.Model(&models.PortfolioItem{}).Where("model_id = ?", modelID).
		Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
	item.OrderIndex = maxOrder + 1
	// ✅ Используем 'db' из параметра
	if err := db.Create(item).Error; err != nil {
		return err
	}

	return nil
}

func (r *PortfolioRepositoryImpl) DeletePortfolioItemWithUpload(db *gorm.DB, itemID string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	// Находим portfolio item
	var item models.PortfolioItem
	// ✅ Используем 'db' из параметра
	if err := db.Preload("Upload").First(&item, "id = ?", itemID).Error; err != nil {
		return ErrPortfolioItemNotFound
	}

	// Удаляем portfolio item
	// ✅ Используем 'db' из параметра
	if err := db.Delete(&item).Error; err != nil {
		return err
	}

	// Обновляем порядок оставшихся элементов
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PortfolioItem{}).
		Where("model_id = ? AND order_index > ?", item.ModelID, item.OrderIndex).
		Update("order_index", gorm.Expr("order_index - ?", 1)).Error; err != nil {
		return err
	}

	// Удаляем связанный upload
	if item.Upload != nil {
		// ✅ Используем 'db' из параметра
		if err := db.Delete(item.Upload).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *PortfolioRepositoryImpl) GetModelPortfolioWithUploads(db *gorm.DB, modelID string) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	// ✅ Используем 'db' из параметра
	err := db.Preload("Upload").Where("model_id = ?", modelID).
		Order("order_index ASC").Find(&items).Error
	return items, err
}

// Additional methods for specific use cases

func (r *PortfolioRepositoryImpl) FindPortfolioItemsByFileType(db *gorm.DB, modelID, fileType string) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	// ✅ Используем 'db' из параметра
	err := db.Preload("Upload").Where("model_id = ?", modelID).
		Joins("LEFT JOIN uploads ON portfolio_items.upload_id = uploads.id").
		Where("uploads.file_type = ?", fileType).
		Order("portfolio_items.order_index ASC").
		Find(&items).Error
	return items, err
}

func (r *PortfolioRepositoryImpl) UpdatePortfolioItemVisibility(db *gorm.DB, itemID string, isPublic bool) error {
	// ✅ Используем 'db' из параметра
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

func (r *PortfolioRepositoryImpl) GetUserStorageUsage(db *gorm.DB, userID string) (int64, error) {
	var totalSize int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Upload{}).Where("user_id = ?", userID).
		Select("COALESCE(SUM(size), 0)").Scan(&totalSize).Error
	return totalSize, err
}

func (r *PortfolioRepositoryImpl) FindRecentPortfolioItems(db *gorm.DB, limit int) ([]models.PortfolioItem, error) {
	var items []models.PortfolioItem
	// ✅ Используем 'db' из параметра
	err := db.Preload("Upload").Preload("Model").
		Order("portfolio_items.created_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

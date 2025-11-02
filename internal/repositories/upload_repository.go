package repositories

import (
	"mwork_backend/internal/models"
	"mwork_backend/internal/types"

	"gorm.io/gorm"
)

// ============================================
// UPLOAD REPOSITORY INTERFACE
// ============================================

type UploadRepository interface {
	// CRUD операции
	Create(db *gorm.DB, upload *models.Upload) error
	FindByID(db *gorm.DB, uploadID string) (*models.Upload, error)
	Update(db *gorm.DB, upload *models.Upload) error
	Delete(db *gorm.DB, uploadID string) error

	// Поиск
	FindByUser(db *gorm.DB, userID string, filters *types.UploadFilters) ([]*models.Upload, error)
	FindByEntity(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error)
	FindByModule(db *gorm.DB, module string, filters *types.UploadFilters) ([]*models.Upload, error)

	// Статистика
	GetUserStorageUsage(db *gorm.DB, userID string) (int64, error)
	GetStats(db *gorm.DB) (map[string]interface{}, error) // Оставили GetStats

	// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
	// Очистка
	// CleanOrphaned(db *gorm.DB) error // Удалено
	DeleteOlderThan(db *gorm.DB, days int) error
	// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲
}

// ============================================
// IMPLEMENTATION
// ============================================

type uploadRepository struct{}

func NewUploadRepository() UploadRepository {
	return &uploadRepository{}
}

// Create - создание записи о файле
func (r *uploadRepository) Create(db *gorm.DB, upload *models.Upload) error {
	return db.Create(upload).Error
}

// FindByID - поиск по ID
func (r *uploadRepository) FindByID(db *gorm.DB, uploadID string) (*models.Upload, error) {
	var upload models.Upload
	err := db.Where("id = ?", uploadID).First(&upload).Error
	return &upload, err
}

// Update - обновление записи
func (r *uploadRepository) Update(db *gorm.DB, upload *models.Upload) error {
	return db.Save(upload).Error
}

// Delete - удаление записи
func (r *uploadRepository) Delete(db *gorm.DB, uploadID string) error {
	return db.Delete(&models.Upload{}, "id = ?", uploadID).Error
}

// FindByUser - все файлы пользователя с фильтрами
func (r *uploadRepository) FindByUser(db *gorm.DB, userID string, filters *types.UploadFilters) ([]*models.Upload, error) {
	query := db.Where("user_id = ?", userID)
	query = r.applyFilters(query, filters)

	var uploads []*models.Upload
	err := query.Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

// FindByEntity - все файлы сущности
func (r *uploadRepository) FindByEntity(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error) {
	var uploads []*models.Upload
	err := db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Find(&uploads).Error
	return uploads, err
}

// FindByModule - все файлы модуля с фильтрами
func (r *uploadRepository) FindByModule(db *gorm.DB, module string, filters *types.UploadFilters) ([]*models.Upload, error) {
	query := db.Where("module = ?", module)
	query = r.applyFilters(query, filters)

	var uploads []*models.Upload
	err := query.Order("created_at DESC").Find(&uploads).Error
	return uploads, err
}

// GetUserStorageUsage - общий размер файлов пользователя
func (r *uploadRepository) GetUserStorageUsage(db *gorm.DB, userID string) (int64, error) {
	var totalSize int64
	err := db.Model(&models.Upload{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(size), 0)").
		Scan(&totalSize).Error
	return totalSize, err
}

// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
// GetStats - статистика платформы (оставлен)
func (r *uploadRepository) GetStats(db *gorm.DB) (map[string]interface{}, error) {
	// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲
	stats := make(map[string]interface{})

	var totalUploads int64
	var totalSize int64

	// Общее количество и размер
	db.Model(&models.Upload{}).Count(&totalUploads)
	db.Model(&models.Upload{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)

	stats["total_uploads"] = totalUploads
	stats["total_size"] = totalSize

	// По модулям
	type ModuleCount struct {
		Module string
		Count  int64
	}
	var moduleCounts []ModuleCount
	db.Model(&models.Upload{}).
		Select("module, COUNT(*) as count").
		Group("module").
		Scan(&moduleCounts)

	byModule := make(map[string]int64)
	for _, mc := range moduleCounts {
		byModule[mc.Module] = mc.Count
	}
	stats["by_module"] = byModule

	// По типам файлов
	type FileTypeCount struct {
		FileType string
		Count    int64
	}
	var fileTypeCounts []FileTypeCount
	db.Model(&models.Upload{}).
		Select("file_type, COUNT(*) as count").
		Group("file_type").
		Scan(&fileTypeCounts)

	byFileType := make(map[string]int64)
	for _, ftc := range fileTypeCounts {
		byFileType[ftc.FileType] = ftc.Count
	}
	stats["by_file_type"] = byFileType

	// Активные пользователи (пользователи с файлами)
	var activeUsers int64
	db.Model(&models.Upload{}).
		Distinct("user_id").
		Count(&activeUsers)
	stats["active_users"] = activeUsers

	stats["storage_used"] = totalSize

	return stats, nil
}

// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
/*
// CleanOrphaned - удаление файлов без связанных сущностей
func (r *uploadRepository) CleanOrphaned(db *gorm.DB) error {
	// Пример: удаляем файлы, у которых entity_id не существует
	// Это нужно адаптировать под вашу логику
	return db.Where("entity_id IS NOT NULL").
		Where("entity_id NOT IN (SELECT id FROM portfolio_items)").
		Delete(&models.Upload{}).Error
}
*/
// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲

// DeleteOlderThan - удаление старых файлов
func (r *uploadRepository) DeleteOlderThan(db *gorm.DB, days int) error {
	return db.Where("created_at < NOW() - INTERVAL ? DAY", days).
		Delete(&models.Upload{}).Error
}

// ============================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ============================================

func (r *uploadRepository) applyFilters(query *gorm.DB, filters *types.UploadFilters) *gorm.DB {
	if filters == nil {
		return query
	}

	if filters.Module != "" {
		query = query.Where("module = ?", filters.Module)
	}

	if filters.EntityType != "" {
		query = query.Where("entity_type = ?", filters.EntityType)
	}

	if filters.EntityID != "" {
		query = query.Where("entity_id = ?", filters.EntityID)
	}

	if len(filters.FileTypes) > 0 {
		query = query.Where("file_type IN ?", filters.FileTypes)
	}

	if filters.Usage != "" {
		query = query.Where("usage = ?", filters.Usage)
	}

	if filters.IsPublic != nil {
		query = query.Where("is_public = ?", *filters.IsPublic)
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}

package services

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mime/multipart"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto" // <-- Используем DTO универсального сервиса
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ИСПРАВЛЕН
// =======================
// - Добавлен 'ctx context.Context'
// - Удалены все дублирующие методы UploadService
type PortfolioService interface {
	// Portfolio operations
	CreatePortfolioItem(ctx context.Context, db *gorm.DB, userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error)
	GetPortfolioItem(ctx context.Context, db *gorm.DB, itemID string) (*dto.PortfolioResponse, error)
	GetModelPortfolio(ctx context.Context, db *gorm.DB, modelID string) ([]*dto.PortfolioResponse, error)
	UpdatePortfolioItem(ctx context.Context, db *gorm.DB, userID, itemID string, req *dto.UpdatePortfolioRequest) error
	UpdatePortfolioOrder(ctx context.Context, db *gorm.DB, userID string, req *dto.ReorderPortfolioRequest) error
	DeletePortfolioItem(ctx context.Context, db *gorm.DB, userID, itemID string) error
	GetPortfolioStats(ctx context.Context, db *gorm.DB, modelID string) (*repositories.PortfolioStats, error)
	TogglePortfolioVisibility(ctx context.Context, db *gorm.DB, userID, itemID string, req *dto.PortfolioVisibilityRequest) error

	// ▼▼▼ УДАЛЕНО: Эти методы теперь в UploadService ▼▼▼
	// UploadFile(...)
	// GetUpload(...)
	// GetUserUploads(...)
	// GetEntityUploads(...)
	// DeleteUpload(...)
	// GetUserStorageUsage(...)
	// CleanOrphanedUploads(...)
	// GetPlatformUploadStats(...)
	// ▲▲▲ УДАЛЕНО ▲▲▲

	// Combined operations
	// (CreatePortfolioWithUpload и DeletePortfolioWithUpload удалены, т.к. стали дубликатами)
	GetFeaturedPortfolio(ctx context.Context, db *gorm.DB, limit int) (*dto.PortfolioListResponse, error)
	GetRecentPortfolio(ctx context.Context, db *gorm.DB, limit int) (*dto.PortfolioListResponse, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ИСПРАВЛЕНА
// =======================
type portfolioService struct {
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
	profileRepo   repositories.ProfileRepository
	uploadService UploadService // <-- ВНЕДРЕН УНИВЕРСАЛЬНЫЙ СЕРВИС

	// ▼▼▼ УДАЛЕНО ▼▼▼
	// fileConfig    dto.FileConfigPortfolio
	// storage       storage.Storage
	// imageProc     *imageprocessor.Processor
	// ▲▲▲ УДАЛЕНО ▲▲▲
}

// ✅ Конструктор обновлен
func NewPortfolioService(
	portfolioRepo repositories.PortfolioRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	uploadService UploadService, // <-- ПРИНИМАЕМ УНИВЕРСАЛЬНЫЙ СЕРВИС
) PortfolioService {
	return &portfolioService{
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		uploadService: uploadService, // <-- СОХРАНЯЕМ УНИВЕРСАЛЬНЫЙ СЕРВИС
		// ▼▼▼ УДАЛЕНО ▼▼▼
		// fileConfig:    config.PortfolioFileConfig,
		// storage:       storage,
		// imageProc:     imageprocessor.NewProcessor(config.AppConfig.Upload.ImageQuality),
		// ▲▲▲ УДАЛЕНО ▲▲▲
	}
}

// Portfolio operations

func (s *portfolioService) CreatePortfolioItem(ctx context.Context, db *gorm.DB, userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil {
		return nil, errors.New("model profile not found or access denied")
	}

	if modelProfile.ID != req.ModelID {
		return nil, errors.New("invalid model ID")
	}

	// 1. Создаем PortfolioItem *сначала* (без UploadID), чтобы получить его ID
	portfolioItem := &models.PortfolioItem{
		ModelID:     req.ModelID,
		Title:       req.Title,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
		// UploadID пока пуст
	}

	if err := s.portfolioRepo.CreatePortfolioItem(tx, portfolioItem); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 2. Теперь загружаем файл, используя ID созданного PortfolioItem
	uploadReq := &dto.UniversalUploadRequest{
		UserID:     userID,
		Module:     "portfolio",
		EntityType: "portfolio_item",
		EntityID:   portfolioItem.ID, // <-- Привязываем к созданному Item
		Usage:      "portfolio_photo",
		IsPublic:   true,
		File:       file,
	}

	// Используем s.uploadService для всей логики загрузки
	uploadRes, err := s.uploadService.UploadFile(ctx, tx, uploadReq)
	if err != nil {
		return nil, err // uploadService уже вернул apperror
	}

	// 3. Обновляем PortfolioItem, добавляя ID загрузки
	portfolioItem.UploadID = uploadRes.ID
	if err := s.portfolioRepo.UpdatePortfolioItem(tx, portfolioItem); err != nil {
		// (Если обновление не удалось, uploadService отменит загрузку при откате tx)
		return nil, apperrors.InternalError(err)
	}

	// 4. Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// 5. Загружаем полные данные для ответа (включая Preload("Upload"))
	newItem, err := s.portfolioRepo.FindPortfolioItemByID(db, portfolioItem.ID)
	if err != nil {
		return nil, handlePortfolioError(err)
	}

	return s.buildPortfolioResponse(db, newItem), nil
}

func (s *portfolioService) GetPortfolioItem(ctx context.Context, db *gorm.DB, itemID string) (*dto.PortfolioResponse, error) {
	item, err := s.portfolioRepo.FindPortfolioItemByID(db, itemID)
	if err != nil {
		return nil, handlePortfolioError(err)
	}
	return s.buildPortfolioResponse(db, item), nil
}

func (s *portfolioService) GetModelPortfolio(ctx context.Context, db *gorm.DB, modelID string) ([]*dto.PortfolioResponse, error) {
	items, err := s.portfolioRepo.FindPortfolioByModel(db, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		// item уже содержит item.Upload благодаря Preload в репозитории
		responses = append(responses, s.buildPortfolioResponse(db, &item))
	}

	return responses, nil
}

func (s *portfolioService) UpdatePortfolioItem(ctx context.Context, db *gorm.DB, userID, itemID string, req *dto.UpdatePortfolioRequest) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Description != nil {
		item.Description = *req.Description
	}

	if req.OrderIndex != nil {
		if err := s.portfolioRepo.UpdatePortfolioItemOrder(tx, item, *req.OrderIndex); err != nil {
			return apperrors.InternalError(err)
		}
	} else {
		if err := s.portfolioRepo.UpdatePortfolioItem(tx, item); err != nil {
			return apperrors.InternalError(err)
		}
	}

	return tx.Commit().Error
}

func (s *portfolioService) UpdatePortfolioOrder(ctx context.Context, db *gorm.DB, userID string, req *dto.ReorderPortfolioRequest) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil {
		return errors.New("model profile not found")
	}

	for _, itemID := range req.ItemIDs {
		item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
		if err != nil {
			return handlePortfolioError(err)
		}
		if item.ModelID != modelProfile.ID {
			return errors.New("access denied for some items")
		}
	}

	if err := s.portfolioRepo.ReorderPortfolioItems(tx, modelProfile.ID, req.ItemIDs); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *portfolioService) DeletePortfolioItem(ctx context.Context, db *gorm.DB, userID, itemID string) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	uploadID := item.UploadID

	// 1. Удаляем PortfolioItem
	if err := s.portfolioRepo.DeletePortfolioItem(tx, itemID); err != nil {
		return apperrors.InternalError(err)
	}

	// 2. Делегируем удаление файла UploadService
	if uploadID != "" {
		if err := s.uploadService.DeleteUpload(ctx, tx, userID, uploadID); err != nil {
			// Логируем, но не отменяем транзакцию,
			// т.к. основная запись (PortfolioItem) уже удалена.
			fmt.Printf("Failed to delete upload (%s) during portfolio item delete: %v\n", uploadID, err)
		}
	}

	return tx.Commit().Error
}

func (s *portfolioService) GetPortfolioStats(ctx context.Context, db *gorm.DB, modelID string) (*repositories.PortfolioStats, error) {
	return s.portfolioRepo.GetPortfolioStats(db, modelID)
}

func (s *portfolioService) TogglePortfolioVisibility(ctx context.Context, db *gorm.DB, userID, itemID string, req *dto.PortfolioVisibilityRequest) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	// ▼▼▼ ИСПРАВЛЕНО: Обновляем 'portfolio_items', а не 'uploads' ▼▼▼
	if err := s.portfolioRepo.UpdatePortfolioItemVisibility(tx, itemID, req.IsPublic); err != nil {
		return apperrors.InternalError(err)
	}
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

	// (Логика обновления upload.IsPublic удалена, т.к. это неверно)

	return tx.Commit().Error
}

// ▼▼▼ УДАЛЕНО: Все операции Upload теперь в UploadService ▼▼▼
// func (s *portfolioService) UploadFile(...)
// func (s *portfolioService) GetUpload(...)
// func (s *portfolioService) GetUserUploads(...)
// ... и т.д.
// ▲▲▲ УДАЛЕНО ▲▲▲

// Combined operations

func (s *portfolioService) GetFeaturedPortfolio(ctx context.Context, db *gorm.DB, limit int) (*dto.PortfolioListResponse, error) {
	items, err := s.portfolioRepo.FindFeaturedPortfolioItems(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		responses = append(responses, s.buildPortfolioResponse(db, &item))
	}

	return &dto.PortfolioListResponse{
		Items: responses,
		Total: len(responses),
	}, nil
}

func (s *portfolioService) GetRecentPortfolio(ctx context.Context, db *gorm.DB, limit int) (*dto.PortfolioListResponse, error) {
	items, err := s.portfolioRepo.FindRecentPortfolioItems(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		responses = append(responses, s.buildPortfolioResponse(db, &item))
	}

	return &dto.PortfolioListResponse{
		Items: responses,
		Total: len(responses),
	}, nil
}

// ▼▼▼ УДАЛЕНО: Admin operations теперь в UploadService ▼▼▼
// func (s *portfolioService) CleanOrphanedUploads(...)
// func (s *portfolioService) GetPlatformUploadStats(...)
// ▲▲▲ УДАЛЕНО ▲▲▲

// Helper methods

// ▼▼▼ УДАЛЕНО: Все хелперы загрузки ▼▼▼
// func (s *portfolioService) createUploadRecord(...)
// func (s *portfolioService) isValidFileType(...)
// func (s *portfolioService) isValidUsage(...)
// func (s *portfolioService) validateEntityAccess(...)
// func (s *portfolioService) generateRandomString(...)
// func (s *portfolioService) generateFileURL(...)
// func (s *portfolioService) generateResizedVersions(...)
// func (s *portfolioService) deleteResizedVersions(...)
// ▲▲▲ УДАЛЕНО ▲▲▲

// buildPortfolioResponse - Упрощено
func (s *portfolioService) buildPortfolioResponse(db *gorm.DB, item *models.PortfolioItem) *dto.PortfolioResponse {
	response := &dto.PortfolioResponse{
		ID:          item.ID,
		ModelID:     item.ModelID,
		Title:       item.Title,
		Description: item.Description,
		OrderIndex:  item.OrderIndex,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}

	// Используем 'item.Upload', который был получен через Preload
	if item.Upload != nil {
		// (buildUploadResponse теперь ожидает models.Upload, а не uploadService.buildUploadResponse)
		response.Upload = &dto.UploadResponse{
			ID:         item.Upload.ID,
			UserID:     item.Upload.UserID,
			EntityType: item.Upload.EntityType,
			EntityID:   item.Upload.EntityID,
			FileType:   item.Upload.FileType,
			Usage:      item.Upload.Usage,
			URL:        item.Upload.Path, // (Примечание: URL теперь должен генерироваться UploadService.GetURL)
			MimeType:   item.Upload.MimeType,
			Size:       item.Upload.Size,
			IsPublic:   item.Upload.IsPublic,
			CreatedAt:  item.Upload.CreatedAt,
		}
	}

	return response
}

// buildUploadResponse - Удален (теперь в 'buildPortfolioResponse')

// (Вспомогательный хелпер для ошибок - без изменений)
func handlePortfolioError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrPortfolioItemNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrProfileNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

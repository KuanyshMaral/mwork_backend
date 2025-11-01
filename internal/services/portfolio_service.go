package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"mwork_backend/internal/config"
	"mwork_backend/internal/imageprocessor"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/storage"
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type PortfolioService interface {
	// Portfolio operations
	CreatePortfolioItem(db *gorm.DB, userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error)
	GetPortfolioItem(db *gorm.DB, itemID string) (*dto.PortfolioResponse, error)
	GetModelPortfolio(db *gorm.DB, modelID string) ([]*dto.PortfolioResponse, error)
	UpdatePortfolioItem(db *gorm.DB, userID, itemID string, req *dto.UpdatePortfolioRequest) error
	UpdatePortfolioOrder(db *gorm.DB, userID string, req *dto.ReorderPortfolioRequest) error
	DeletePortfolioItem(db *gorm.DB, userID, itemID string) error
	GetPortfolioStats(db *gorm.DB, modelID string) (*repositories.PortfolioStats, error)
	TogglePortfolioVisibility(db *gorm.DB, userID, itemID string, req *dto.PortfolioVisibilityRequest) error

	// Upload operations
	UploadFile(db *gorm.DB, userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error)
	GetUpload(db *gorm.DB, uploadID string) (*models.Upload, error)
	GetUserUploads(db *gorm.DB, userID string) ([]*models.Upload, error)
	GetEntityUploads(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error)
	DeleteUpload(db *gorm.DB, userID, uploadID string) error
	GetUserStorageUsage(db *gorm.DB, userID string) (*dto.StorageUsageResponse, error)

	// Combined operations
	CreatePortfolioWithUpload(db *gorm.DB, userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error)
	DeletePortfolioWithUpload(db *gorm.DB, userID, itemID string) error
	GetFeaturedPortfolio(db *gorm.DB, limit int) (*dto.PortfolioListResponse, error)
	GetRecentPortfolio(db *gorm.DB, limit int) (*dto.PortfolioListResponse, error)

	// Admin operations
	CleanOrphanedUploads(db *gorm.DB) error
	GetPlatformUploadStats(db *gorm.DB) (*dto.UploadStats, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type portfolioService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
	profileRepo   repositories.ProfileRepository
	fileConfig    dto.FileConfigPortfolio
	storage       storage.Storage
	imageProc     *imageprocessor.Processor
}

// ✅ Конструктор обновлен (db убран)
func NewPortfolioService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	portfolioRepo repositories.PortfolioRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	storage storage.Storage,
) PortfolioService {
	return &portfolioService{
		// ❌ 'db: db,' УДАЛЕНО
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		fileConfig:    config.PortfolioFileConfig,
		storage:       storage,
		imageProc:     imageprocessor.NewProcessor(config.AppConfig.Upload.ImageQuality),
	}
}

// Portfolio operations

// CreatePortfolioItem - 'db' добавлен
func (s *portfolioService) CreatePortfolioItem(db *gorm.DB, userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil {
		return nil, errors.New("model profile not found or access denied")
	}

	if modelProfile.ID != req.ModelID {
		return nil, errors.New("invalid model ID")
	}

	// ✅ Передаем tx
	upload, err := s.createUploadRecord(tx, userID, file, &dto.UploadRequest{
		EntityType: "portfolio",
		EntityID:   "", // Будет установлен ниже
		FileType:   "image",
		Usage:      "portfolio_photo",
		IsPublic:   true,
	})
	if err != nil {
		return nil, err
	}

	portfolioItem := &models.PortfolioItem{
		ModelID:     req.ModelID,
		UploadID:    upload.ID,
		Title:       req.Title,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
	}

	// ✅ Передаем tx
	if err := s.portfolioRepo.CreatePortfolioItem(tx, portfolioItem); err != nil {
		return nil, apperrors.InternalError(err)
	}

	upload.EntityID = portfolioItem.ID
	// ✅ Передаем tx
	if err := s.portfolioRepo.UpdateUpload(tx, upload); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// --- Внешнее I/O ---
	ctx := context.TODO()
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	if err := s.storage.Save(ctx, upload.Path, src, upload.MimeType); err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}
	// --- Конец I/O ---

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		s.storage.Delete(ctx, upload.Path)
		return nil, apperrors.InternalError(err)
	}

	go s.generateResizedVersions(upload.Path, file)

	// ✅ Передаем 'db' (пул) в хелпер
	return s.buildPortfolioResponse(db, portfolioItem, upload), nil
}

// GetPortfolioItem - 'db' добавлен
func (s *portfolioService) GetPortfolioItem(db *gorm.DB, itemID string) (*dto.PortfolioResponse, error) {
	// ✅ Используем 'db' из параметра
	item, err := s.portfolioRepo.FindPortfolioItemByID(db, itemID)
	if err != nil {
		return nil, handlePortfolioError(err)
	}

	var upload *models.Upload
	if item.Upload != nil {
		upload = item.Upload
	} else {
		// ✅ Используем 'db' из параметра
		upload, _ = s.portfolioRepo.FindUploadByID(db, item.UploadID)
	}

	// ✅ Используем 'db' из параметра
	return s.buildPortfolioResponse(db, item, upload), nil
}

// GetModelPortfolio - 'db' добавлен
func (s *portfolioService) GetModelPortfolio(db *gorm.DB, modelID string) ([]*dto.PortfolioResponse, error) {
	// ✅ Используем 'db' из параметра
	items, err := s.portfolioRepo.FindPortfolioByModel(db, modelID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		var upload *models.Upload
		if item.Upload != nil {
			upload = item.Upload
		} else {
			// ✅ Используем 'db' из параметра
			upload, _ = s.portfolioRepo.FindUploadByID(db, item.UploadID)
		}
		// ✅ Используем 'db' из параметра
		responses = append(responses, s.buildPortfolioResponse(db, &item, upload))
	}

	return responses, nil
}

// UpdatePortfolioItem - 'db' добавлен
func (s *portfolioService) UpdatePortfolioItem(db *gorm.DB, userID, itemID string, req *dto.UpdatePortfolioRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	// ✅ Передаем tx
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
		// ✅ Передаем tx
		if err := s.portfolioRepo.UpdatePortfolioItemOrder(tx, item, *req.OrderIndex); err != nil {
			return apperrors.InternalError(err)
		}
	} else {
		// ✅ Передаем tx
		if err := s.portfolioRepo.UpdatePortfolioItem(tx, item); err != nil {
			return apperrors.InternalError(err)
		}
	}

	return tx.Commit().Error
}

// UpdatePortfolioOrder - 'db' добавлен
func (s *portfolioService) UpdatePortfolioOrder(db *gorm.DB, userID string, req *dto.ReorderPortfolioRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil {
		return errors.New("model profile not found")
	}

	for _, itemID := range req.ItemIDs {
		// ✅ Передаем tx
		item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
		if err != nil {
			return handlePortfolioError(err)
		}
		if item.ModelID != modelProfile.ID {
			return errors.New("access denied for some items")
		}
	}

	// ✅ Передаем tx
	if err := s.portfolioRepo.ReorderPortfolioItems(tx, modelProfile.ID, req.ItemIDs); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeletePortfolioItem - 'db' добавлен
func (s *portfolioService) DeletePortfolioItem(db *gorm.DB, userID, itemID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	// ✅ Передаем tx
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	if err := s.portfolioRepo.DeletePortfolioItem(tx, itemID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetPortfolioStats - 'db' добавлен
func (s *portfolioService) GetPortfolioStats(db *gorm.DB, modelID string) (*repositories.PortfolioStats, error) {
	// ✅ Используем 'db' из параметра
	return s.portfolioRepo.GetPortfolioStats(db, modelID)
}

// TogglePortfolioVisibility - 'db' добавлен
func (s *portfolioService) TogglePortfolioVisibility(db *gorm.DB, userID, itemID string, req *dto.PortfolioVisibilityRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	// ✅ Передаем tx
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	upload, err := s.portfolioRepo.FindUploadByID(tx, item.UploadID)
	if err != nil {
		return handlePortfolioError(err)
	}

	upload.IsPublic = req.IsPublic
	// ✅ Передаем tx
	if err := s.portfolioRepo.UpdateUpload(tx, upload); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Upload operations

// UploadFile - 'db' добавлен
func (s *portfolioService) UploadFile(db *gorm.DB, userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.validateEntityAccess(tx, userID, req.EntityType, req.EntityID); err != nil {
		return nil, err
	}

	// ✅ Передаем tx
	upload, err := s.createUploadRecord(tx, userID, file, req)
	if err != nil {
		return nil, err
	}

	// --- Внешнее I/O ---
	ctx := context.TODO()
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	if err := s.storage.Save(ctx, upload.Path, src, upload.MimeType); err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}
	// --- Конец I/O ---

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		s.storage.Delete(ctx, upload.Path)
		return nil, apperrors.InternalError(err)
	}

	if strings.HasPrefix(upload.MimeType, "image/") {
		go s.generateResizedVersions(upload.Path, file)
	}

	return s.buildUploadResponse(upload), nil
}

// GetUpload - 'db' добавлен
func (s *portfolioService) GetUpload(db *gorm.DB, uploadID string) (*models.Upload, error) {
	// ✅ Используем 'db' из параметра
	upload, err := s.portfolioRepo.FindUploadByID(db, uploadID)
	if err != nil {
		return nil, handlePortfolioError(err)
	}
	return upload, nil
}

// GetUserUploads - 'db' добавлен
func (s *portfolioService) GetUserUploads(db *gorm.DB, userID string) ([]*models.Upload, error) {
	// ✅ Используем 'db' из параметра
	uploads, err := s.portfolioRepo.FindUploadsByUser(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var result []*models.Upload
	for i := range uploads {
		result = append(result, &uploads[i])
	}
	return result, nil
}

// GetEntityUploads - 'db' добавлен
func (s *portfolioService) GetEntityUploads(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error) {
	// ✅ Используем 'db' из параметра
	uploads, err := s.portfolioRepo.FindUploadsByEntity(db, entityType, entityID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var result []*models.Upload
	for i := range uploads {
		result = append(result, &uploads[i])
	}
	return result, nil
}

// DeleteUpload - 'db' добавлен
func (s *portfolioService) DeleteUpload(db *gorm.DB, userID, uploadID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	upload, err := s.portfolioRepo.FindUploadByID(tx, uploadID)
	if err != nil {
		return handlePortfolioError(err)
	}

	if upload.UserID != userID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	if err := s.portfolioRepo.DeleteUpload(tx, uploadID); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}

	// --- Внешнее I/O (ПОСЛЕ коммита) ---
	ctx := context.TODO()
	if err := s.storage.Delete(ctx, upload.Path); err != nil {
		fmt.Printf("Failed to delete file from storage: %v\n", err)
	}
	if strings.HasPrefix(upload.MimeType, "image/") {
		s.deleteResizedVersions(upload.Path)
	}
	// --- Конец I/O ---

	return nil
}

// GetUserStorageUsage - 'db' добавлен
func (s *portfolioService) GetUserStorageUsage(db *gorm.DB, userID string) (*dto.StorageUsageResponse, error) {
	// ✅ Используем 'db' из параметра
	used, err := s.portfolioRepo.GetUserStorageUsage(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &dto.StorageUsageResponse{
		Used:  used,
		Limit: s.fileConfig.MaxUserStorage,
	}, nil
}

// Combined operations

// CreatePortfolioWithUpload - 'db' добавлен
func (s *portfolioService) CreatePortfolioWithUpload(db *gorm.DB, userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error) {
	// ✅ Передаем 'db'
	return s.CreatePortfolioItem(db, userID, req, file)
}

// DeletePortfolioWithUpload - 'db' добавлен
func (s *portfolioService) DeletePortfolioWithUpload(db *gorm.DB, userID, itemID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	item, err := s.portfolioRepo.FindPortfolioItemByID(tx, itemID)
	if err != nil {
		return handlePortfolioError(err)
	}

	// ✅ Передаем tx
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(tx, userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	upload, err := s.portfolioRepo.FindUploadByID(tx, item.UploadID)
	if err != nil {
		return handlePortfolioError(err)
	}

	// ✅ Передаем tx
	if err := s.portfolioRepo.DeletePortfolioItemWithUpload(tx, itemID); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}

	// --- Внешнее I/O (ПОСЛЕ коммита) ---
	ctx := context.TODO()
	if err := s.storage.Delete(ctx, upload.Path); err != nil {
		fmt.Printf("Failed to delete file from storage: %v\n", err)
	}
	if strings.HasPrefix(upload.MimeType, "image/") {
		s.deleteResizedVersions(upload.Path)
	}
	// --- Конец I/O ---

	return nil
}

// GetFeaturedPortfolio - 'db' добавлен
func (s *portfolioService) GetFeaturedPortfolio(db *gorm.DB, limit int) (*dto.PortfolioListResponse, error) {
	// ✅ Используем 'db' из параметра
	items, err := s.portfolioRepo.FindFeaturedPortfolioItems(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		var upload *models.Upload
		if item.Upload != nil {
			upload = item.Upload
		} else {
			// ✅ Используем 'db' из параметра
			upload, _ = s.portfolioRepo.FindUploadByID(db, item.UploadID)
		}
		// ✅ Используем 'db' из параметра
		responses = append(responses, s.buildPortfolioResponse(db, &item, upload))
	}

	return &dto.PortfolioListResponse{
		Items: responses,
		Total: len(responses),
	}, nil
}

// GetRecentPortfolio - 'db' добавлен
func (s *portfolioService) GetRecentPortfolio(db *gorm.DB, limit int) (*dto.PortfolioListResponse, error) {
	// ✅ Используем 'db' из параметра
	items, err := s.portfolioRepo.FindRecentPortfolioItems(db, limit)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		var upload *models.Upload
		if item.Upload != nil {
			upload = item.Upload
		} else {
			// ✅ Используем 'db' из параметра
			upload, _ = s.portfolioRepo.FindUploadByID(db, item.UploadID)
		}
		// ✅ Используем 'db' из параметра
		responses = append(responses, s.buildPortfolioResponse(db, &item, upload))
	}

	return &dto.PortfolioListResponse{
		Items: responses,
		Total: len(responses),
	}, nil
}

// Admin operations

// CleanOrphanedUploads - 'db' добавлен
func (s *portfolioService) CleanOrphanedUploads(db *gorm.DB) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.portfolioRepo.CleanOrphanedUploads(tx); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetPlatformUploadStats - 'db' добавлен
func (s *portfolioService) GetPlatformUploadStats(db *gorm.DB) (*dto.UploadStats, error) {
	// ✅ Используем 'db' из параметра
	// TODO: Реализовать s.portfolioRepo.GetPlatformUploadStats(db)
	return &dto.UploadStats{
		TotalUploads: 0,
		TotalSize:    0,
		ByFileType:   make(map[string]int64),
		ByUsage:      make(map[string]int64),
		ActiveUsers:  0,
		StorageUsed:  0,
		StorageLimit: 0,
	}, nil
}

// Helper methods

// (createUploadRecord - 'db' уже был)
func (s *portfolioService) createUploadRecord(db *gorm.DB, userID string, file *multipart.FileHeader, req *dto.UploadRequest) (*models.Upload, error) {
	if file.Size > s.fileConfig.MaxSize {
		return nil, apperrors.ErrFileTooLarge
	}
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = getMimeTypeFromFilename(file.Filename)
	}
	if !s.isValidFileType(mimeType) {
		return nil, apperrors.ErrInvalidFileType
	}
	if !s.isValidUsage(req.EntityType, req.Usage) {
		return nil, apperrors.ErrInvalidUploadUsage
	}

	// ✅ Передаем db
	currentUsage, err := s.portfolioRepo.GetUserStorageUsage(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	if currentUsage+file.Size > s.fileConfig.MaxUserStorage {
		return nil, apperrors.ErrStorageLimitExceeded
	}

	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), s.generateRandomString(8), fileExt)
	filePath := filepath.Join(req.EntityType, fileName)

	upload := &models.Upload{
		UserID:     userID,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		FileType:   s.getFileTypeFromMIME(mimeType),
		Usage:      req.Usage,
		Path:       filePath,
		MimeType:   mimeType,
		Size:       file.Size,
		IsPublic:   req.IsPublic,
	}

	// ✅ Передаем db
	if err := s.portfolioRepo.CreateUpload(db, upload); err != nil {
		return nil, apperrors.InternalError(err)
	}
	return upload, nil
}

// (isValidFileType - чистая функция, без изменений)
func (s *portfolioService) isValidFileType(mimeType string) bool {
	for _, allowedType := range s.fileConfig.AllowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

// (isValidUsage - чистая функция, без изменений)
func (s *portfolioService) isValidUsage(entityType, usage string) bool {
	allowedUsages, ok := s.fileConfig.AllowedUsages[entityType]
	if !ok {
		return false
	}
	for _, allowedUsage := range allowedUsages {
		if usage == allowedUsage {
			return true
		}
	}
	return false
}

// (getFileTypeFromMIME - чистая функция, без изменений)
func (s *portfolioService) getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "file"
}

// (validateEntityAccess - 'db' уже был)
func (s *portfolioService) validateEntityAccess(db *gorm.DB, userID, entityType, entityID string) error {
	switch entityType {
	case "portfolio":
		if entityID != "" {
			// ✅ Передаем db
			item, err := s.portfolioRepo.FindPortfolioItemByID(db, entityID)
			if err != nil {
				return errors.New("portfolio item not found")
			}
			// ✅ Передаем db
			modelProfile, err := s.profileRepo.FindModelProfileByUserID(db, userID)
			if err != nil || modelProfile.ID != item.ModelID {
				return errors.New("access denied")
			}
		}
	case "model_profile":
		if entityID != "" {
			// ✅ Передаем db
			profile, err := s.profileRepo.FindModelProfileByID(db, entityID)
			if err != nil {
				return errors.New("profile not found")
			}
			if profile.UserID != userID {
				return errors.New("access denied")
			}
		}
	}
	return nil
}

// (buildPortfolioResponse - 'db' уже был)
func (s *portfolioService) buildPortfolioResponse(db *gorm.DB, item *models.PortfolioItem, upload *models.Upload) *dto.PortfolioResponse {
	response := &dto.PortfolioResponse{
		ID:          item.ID,
		ModelID:     item.ModelID,
		Title:       item.Title,
		Description: item.Description,
		OrderIndex:  item.OrderIndex,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}

	if upload == nil && item.UploadID != "" {
		// ✅ Передаем db
		upload, _ = s.portfolioRepo.FindUploadByID(db, item.UploadID)
	}

	if upload != nil {
		response.Upload = s.buildUploadResponse(upload)
	}

	return response
}

// (buildUploadResponse - чистая функция, без изменений)
func (s *portfolioService) buildUploadResponse(upload *models.Upload) *dto.UploadResponse {
	return &dto.UploadResponse{
		ID:         upload.ID,
		UserID:     upload.UserID,
		EntityType: upload.EntityType,
		EntityID:   upload.EntityID,
		FileType:   upload.FileType,
		Usage:      upload.Usage,
		URL:        s.generateFileURL(upload),
		MimeType:   upload.MimeType,
		Size:       upload.Size,
		IsPublic:   upload.IsPublic,
		CreatedAt:  upload.CreatedAt,
	}
}

// (generateRandomString - чистая функция, без изменений)
func (s *portfolioService) generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

// (generateFileURL - чистая функция, без изменений)
func (s *portfolioService) generateFileURL(upload *models.Upload) string {
	ctx := context.TODO()
	url, err := s.storage.GetURL(ctx, upload.Path)
	if err != nil {
		return fmt.Sprintf("/api/v1/files/%s", upload.ID)
	}
	return url
}

// (generateResizedVersions - чистая функция, без изменений)
func (s *portfolioService) generateResizedVersions(originalPath string, file *multipart.FileHeader) {
	ctx := context.TODO()
	sizes := []imageprocessor.ImageSize{
		imageprocessor.SizeThumbnail,
		imageprocessor.SizeSmall,
		imageprocessor.SizeMedium,
	}
	for _, size := range sizes {
		src, err := file.Open()
		if err != nil {
			continue
		}
		format := strings.TrimPrefix(filepath.Ext(originalPath), ".")
		resized, err := s.imageProc.ProcessImage(src, size, format)
		src.Close()
		if err != nil {
			continue
		}
		resizedPath := getResizedPath(originalPath, size.Name)
		mimeType := "image/" + format
		s.storage.Save(ctx, resizedPath, resized, mimeType)
	}
}

// (deleteResizedVersions - чистая функция, без изменений)
func (s *portfolioService) deleteResizedVersions(originalPath string) {
	ctx := context.TODO()
	sizes := []string{"thumbnail", "small", "medium"}
	for _, size := range sizes {
		resizedPath := getResizedPath(originalPath, size)
		s.storage.Delete(ctx, resizedPath)
	}
}

// (Вспомогательный хелпер для ошибок - без изменений)
func handlePortfolioError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrPortfolioItemNotFound) ||
		errors.Is(err, repositories.ErrUploadNotFound) {
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

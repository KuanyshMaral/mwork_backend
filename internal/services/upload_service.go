package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"gorm.io/gorm" // 👈 gorm импорт уже был
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
type UploadService interface {
	UploadFile(db *gorm.DB, userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error)
	GetUpload(db *gorm.DB, uploadID string) (*models.Upload, error)
	GetUserUploads(db *gorm.DB, userID string) ([]*models.Upload, error)
	GetEntityUploads(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error)
	DeleteUpload(db *gorm.DB, userID, uploadID string) error
	GetUserStorageUsage(db *gorm.DB, userID string) (*dto.StorageUsageResponse, error)
	CleanOrphanedUploads(db *gorm.DB) error
	GetPlatformUploadStats(db *gorm.DB) (*dto.UploadStats, error)
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type uploadService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
	profileRepo   repositories.ProfileRepository
	storage       storage.Storage
	imageProc     *imageprocessor.Processor
	fileConfig    dto.FileConfigPortfolio
}

// ✅ Конструктор обновлен (db убран)
func NewUploadService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	portfolioRepo repositories.PortfolioRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	storage storage.Storage,
) UploadService {
	return &uploadService{
		// ❌ 'db: db,' УДАЛЕНО
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		storage:       storage,
		imageProc:     imageprocessor.NewProcessor(config.AppConfig.Upload.ImageQuality),
		fileConfig:    config.PortfolioFileConfig,
	}
}

// UploadFile - 'db' добавлен
func (s *uploadService) UploadFile(db *gorm.DB, userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error) {
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
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	ctx := context.TODO()
	// Сохраняем в S3/Storage.
	if err := s.storage.Save(ctx, upload.Path, src, upload.MimeType); err != nil {
		// Ошибка I/O, транзакция автоматически откатится (defer tx.Rollback())
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}
	// --- Конец Внешнего I/O ---

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		// Ошибка коммита: файл уже в S3, но запись в БД не удалась.
		// Удаляем "осиротевший" файл.
		if delErr := s.storage.Delete(ctx, upload.Path); delErr != nil {
			return nil, fmt.Errorf("db commit failed (%v) and storage cleanup failed (%v)", err, delErr)
		}
		return nil, apperrors.InternalError(err)
	}

	// Запускаем асинхронную нарезку (эта go-рутина не использует БД)
	if strings.HasPrefix(upload.MimeType, "image/") {
		go s.generateResizedVersions(upload.Path, file)
	}

	return s.buildUploadResponse(upload), nil
}

// GetUpload - 'db' добавлен
func (s *uploadService) GetUpload(db *gorm.DB, uploadID string) (*models.Upload, error) {
	// ✅ Используем 'db' из параметра
	upload, err := s.portfolioRepo.FindUploadByID(db, uploadID)
	if err != nil {
		return nil, handleUploadError(err)
	}
	return upload, nil
}

// GetUserUploads - 'db' добавлен
func (s *uploadService) GetUserUploads(db *gorm.DB, userID string) ([]*models.Upload, error) {
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
func (s *uploadService) GetEntityUploads(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error) {
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
func (s *uploadService) DeleteUpload(db *gorm.DB, userID, uploadID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	upload, err := s.portfolioRepo.FindUploadByID(tx, uploadID)
	if err != nil {
		return handleUploadError(err)
	}

	if upload.UserID != userID {
		// TODO: Использовать apperrors.ErrForbidden
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
	// --- Конец Внешнего I/O ---

	return nil
}

// GetUserStorageUsage - 'db' добавлен
func (s *uploadService) GetUserStorageUsage(db *gorm.DB, userID string) (*dto.StorageUsageResponse, error) {
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

// CleanOrphanedUploads - 'db' добавлен
func (s *uploadService) CleanOrphanedUploads(db *gorm.DB) error {
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
func (s *uploadService) GetPlatformUploadStats(db *gorm.DB) (*dto.UploadStats, error) {
	// ✅ Используем 'db' из параметра
	// (Реализация-заглушка, замени на вызовы репозитория)
	var totalSize int64
	var totalUploads int64
	db.Model(&models.Upload{}).Count(&totalUploads)
	db.Model(&models.Upload{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)

	return &dto.UploadStats{
		TotalUploads: totalUploads,
		TotalSize:    totalSize,
		ByFileType:   make(map[string]int64),
		ByUsage:      make(map[string]int64),
		ActiveUsers:  0, // (Нужен userRepo)
		StorageUsed:  totalSize,
		StorageLimit: 0, // (Общий лимит?)
	}, nil
}

// =======================
// 3. ХЕЛПЕРЫ
// =======================

// ✅ createUploadRecord - 'db' добавлен
func (s *uploadService) createUploadRecord(db *gorm.DB, userID string, file *multipart.FileHeader, req *dto.UploadRequest) (*models.Upload, error) {
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
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), generateSecureRandomString(8), fileExt)
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

// ✅ validateEntityAccess - 'db' добавлен
func (s *uploadService) validateEntityAccess(db *gorm.DB, userID, entityType, entityID string) error {
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

// (generateResizedVersions не использует БД, без изменений)
func (s *uploadService) generateResizedVersions(originalPath string, file *multipart.FileHeader) {
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

// (deleteResizedVersions не использует БД, без изменений)
func (s *uploadService) deleteResizedVersions(originalPath string) {
	ctx := context.TODO()
	sizes := []string{"thumbnail", "small", "medium"}
	for _, size := range sizes {
		resizedPath := getResizedPath(originalPath, size)
		s.storage.Delete(ctx, resizedPath)
	}
}

// (isValidFileType - чистая функция, без изменений)
func (s *uploadService) isValidFileType(mimeType string) bool {
	for _, allowedType := range s.fileConfig.AllowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

// (isValidUsage - чистая функция, без изменений)
func (s *uploadService) isValidUsage(entityType, usage string) bool {
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
func (s *uploadService) getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "file"
}

// (buildUploadResponse не использует БД, без изменений)
func (s *uploadService) buildUploadResponse(upload *models.Upload) *dto.UploadResponse {
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

// (generateFileURL не использует БД, без изменений)
func (s *uploadService) generateFileURL(upload *models.Upload) string {
	ctx := context.TODO()
	url, err := s.storage.GetURL(ctx, upload.Path)
	if err != nil {
		return fmt.Sprintf("/api/v1/files/%s", upload.ID)
	}
	return url
}

// (getMimeTypeFromFilename - чистая функция, без изменений)
func getMimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".jpg": ".jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif",
		".webp": "image/webp", ".mp4": "video/mp4", ".mov": "video/quicktime",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// (getResizedPath - чистая функция, без изменений)
func getResizedPath(originalPath, size string) string {
	ext := filepath.Ext(originalPath)
	nameWithoutExt := strings.TrimSuffix(originalPath, ext)
	return fmt.Sprintf("%s_%s%s", nameWithoutExt, size, ext)
}

// (generateSecureRandomString - чистая функция, без изменений)
func generateSecureRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

// (handleUploadError - хелпер, без изменений)
func handleUploadError(err error) error {
	// (Предполагаем, что эти ошибки существуют в repositories)
	if errors.Is(err, gorm.ErrRecordNotFound) ||
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

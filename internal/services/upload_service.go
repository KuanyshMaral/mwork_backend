package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/config"
	"mwork_backend/internal/imageprocessor"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/storage"
)

// UploadService handles file upload operations
type UploadService interface {
	UploadFile(userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error)
	GetUpload(uploadID string) (*models.Upload, error)
	GetUserUploads(userID string) ([]*models.Upload, error)
	GetEntityUploads(entityType, entityID string) ([]*models.Upload, error)
	DeleteUpload(userID, uploadID string) error
	GetUserStorageUsage(userID string) (*dto.StorageUsageResponse, error)
	CleanOrphanedUploads() error
	GetPlatformUploadStats() (*dto.UploadStats, error)
}

type uploadService struct {
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
	profileRepo   repositories.ProfileRepository
	storage       storage.Storage
	imageProc     *imageprocessor.Processor
	fileConfig    dto.FileConfigPortfolio
}

// NewUploadService creates a new upload service
func NewUploadService(
	portfolioRepo repositories.PortfolioRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	storage storage.Storage,
) UploadService {
	return &uploadService{
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		storage:       storage,
		imageProc:     imageprocessor.NewProcessor(config.AppConfig.Upload.ImageQuality),
		fileConfig:    config.PortfolioFileConfig,
	}
}

// UploadFile handles file upload with validation and storage
func (s *uploadService) UploadFile(userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error) {
	// Validate user has access to the entity
	if err := s.validateEntityAccess(userID, req.EntityType, req.EntityID); err != nil {
		return nil, err
	}

	upload, err := s.processUpload(userID, file, req)
	if err != nil {
		return nil, err
	}

	return s.buildUploadResponse(upload), nil
}

// GetUpload retrieves upload metadata
func (s *uploadService) GetUpload(uploadID string) (*models.Upload, error) {
	return s.portfolioRepo.FindUploadByID(uploadID)
}

// GetUserUploads retrieves all uploads for a user
func (s *uploadService) GetUserUploads(userID string) ([]*models.Upload, error) {
	uploads, err := s.portfolioRepo.FindUploadsByUser(userID)
	if err != nil {
		return nil, err
	}

	var result []*models.Upload
	for i := range uploads {
		result = append(result, &uploads[i])
	}

	return result, nil
}

// GetEntityUploads retrieves all uploads for an entity
func (s *uploadService) GetEntityUploads(entityType, entityID string) ([]*models.Upload, error) {
	uploads, err := s.portfolioRepo.FindUploadsByEntity(entityType, entityID)
	if err != nil {
		return nil, err
	}

	var result []*models.Upload
	for i := range uploads {
		result = append(result, &uploads[i])
	}

	return result, nil
}

// DeleteUpload deletes an upload and its file
func (s *uploadService) DeleteUpload(userID, uploadID string) error {
	upload, err := s.portfolioRepo.FindUploadByID(uploadID)
	if err != nil {
		return err
	}

	if upload.UserID != userID {
		return errors.New("access denied")
	}

	// Delete file from storage
	ctx := context.Background()
	if err := s.storage.Delete(ctx, upload.Path); err != nil {
		// Log error but continue with database deletion
		fmt.Printf("Failed to delete file from storage: %v\n", err)
	}

	// Delete resized versions if it's an image
	if strings.HasPrefix(upload.MimeType, "image/") {
		s.deleteResizedVersions(upload.Path)
	}

	return s.portfolioRepo.DeleteUpload(uploadID)
}

// GetUserStorageUsage returns storage usage for a user
func (s *uploadService) GetUserStorageUsage(userID string) (*dto.StorageUsageResponse, error) {
	used, err := s.portfolioRepo.GetUserStorageUsage(userID)
	if err != nil {
		return nil, err
	}

	return &dto.StorageUsageResponse{
		Used:  used,
		Limit: s.fileConfig.MaxUserStorage,
	}, nil
}

// CleanOrphanedUploads removes uploads not associated with any entity
func (s *uploadService) CleanOrphanedUploads() error {
	return s.portfolioRepo.CleanOrphanedUploads()
}

// GetPlatformUploadStats returns platform-wide upload statistics
func (s *uploadService) GetPlatformUploadStats() (*dto.UploadStats, error) {
	// This would require additional repository methods
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

// processUpload handles the core upload logic
func (s *uploadService) processUpload(userID string, file *multipart.FileHeader, req *dto.UploadRequest) (*models.Upload, error) {
	// Validate file size
	if file.Size > s.fileConfig.MaxSize {
		return nil, appErrors.ErrFileTooLarge
	}

	// Validate file type
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = getMimeTypeFromFilename(file.Filename)
	}

	if !s.isValidFileType(mimeType) {
		return nil, appErrors.ErrInvalidFileType
	}

	// Validate usage
	if !s.isValidUsage(req.EntityType, req.Usage) {
		return nil, appErrors.ErrInvalidUploadUsage
	}

	// Check user storage limit
	currentUsage, err := s.portfolioRepo.GetUserStorageUsage(userID)
	if err != nil {
		return nil, err
	}

	if currentUsage+file.Size > s.fileConfig.MaxUserStorage {
		return nil, appErrors.ErrStorageLimitExceeded
	}

	// Generate file path
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), generateSecureRandomString(8), fileExt)
	filePath := filepath.Join(req.EntityType, fileName)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Save file to storage
	ctx := context.Background()
	if err := s.storage.Save(ctx, filePath, src, mimeType); err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}

	// If it's an image, generate resized versions asynchronously
	if strings.HasPrefix(mimeType, "image/") {
		go s.generateResizedVersions(filePath, file)
	}

	// Create upload record
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

	if err := s.portfolioRepo.CreateUpload(upload); err != nil {
		// Clean up uploaded file on database error
		s.storage.Delete(ctx, filePath)
		return nil, err
	}

	return upload, nil
}

// generateResizedVersions creates multiple sizes of an image
func (s *uploadService) generateResizedVersions(originalPath string, file *multipart.FileHeader) {
	ctx := context.Background()
	sizes := []imageprocessor.ImageSize{
		imageprocessor.SizeThumbnail,
		imageprocessor.SizeSmall,
		imageprocessor.SizeMedium,
	}

	for _, size := range sizes {
		// Open original file
		src, err := file.Open()
		if err != nil {
			continue
		}

		// Process image
		format := strings.TrimPrefix(filepath.Ext(originalPath), ".")
		resized, err := s.imageProc.ProcessImage(src, size, format)
		src.Close()

		if err != nil {
			continue
		}

		// Generate resized path
		resizedPath := getResizedPath(originalPath, size.Name)

		// Save resized version
		mimeType := "image/" + format
		s.storage.Save(ctx, resizedPath, resized, mimeType)
	}
}

// deleteResizedVersions removes all resized versions of an image
func (s *uploadService) deleteResizedVersions(originalPath string) {
	ctx := context.Background()
	sizes := []string{"thumbnail", "small", "medium"}

	for _, size := range sizes {
		resizedPath := getResizedPath(originalPath, size)
		s.storage.Delete(ctx, resizedPath)
	}
}

// Validation helpers

func (s *uploadService) isValidFileType(mimeType string) bool {
	for _, allowedType := range s.fileConfig.AllowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

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

func (s *uploadService) getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "file"
}

func (s *uploadService) validateEntityAccess(userID, entityType, entityID string) error {
	// Validate user has access to the entity
	switch entityType {
	case "portfolio":
		if entityID != "" {
			item, err := s.portfolioRepo.FindPortfolioItemByID(entityID)
			if err != nil {
				return errors.New("portfolio item not found")
			}
			modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
			if err != nil || modelProfile.ID != item.ModelID {
				return errors.New("access denied")
			}
		}
	case "model_profile":
		if entityID != "" {
			profile, err := s.profileRepo.FindModelProfileByID(entityID)
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

// Response builders

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

func (s *uploadService) generateFileURL(upload *models.Upload) string {
	ctx := context.Background()
	url, err := s.storage.GetURL(ctx, upload.Path)
	if err != nil {
		// Fallback to default URL format
		return fmt.Sprintf("/api/v1/files/%s", upload.ID)
	}
	return url
}

// Utility functions

func getMimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func getResizedPath(originalPath, size string) string {
	ext := filepath.Ext(originalPath)
	nameWithoutExt := strings.TrimSuffix(originalPath, ext)
	return fmt.Sprintf("%s_%s%s", nameWithoutExt, size, ext)
}

func generateSecureRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based generation
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

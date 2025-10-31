package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"mwork_backend/internal/models"
	"strings"
	"time"

	"mwork_backend/internal/config"
	"mwork_backend/internal/imageprocessor"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/storage"
)

type PortfolioService interface {
	// Portfolio operations
	CreatePortfolioItem(userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error)
	GetPortfolioItem(itemID string) (*dto.PortfolioResponse, error)
	GetModelPortfolio(modelID string) ([]*dto.PortfolioResponse, error)
	UpdatePortfolioItem(userID, itemID string, req *dto.UpdatePortfolioRequest) error
	UpdatePortfolioOrder(userID string, req *dto.ReorderPortfolioRequest) error
	DeletePortfolioItem(userID, itemID string) error
	GetPortfolioStats(modelID string) (*repositories.PortfolioStats, error)
	TogglePortfolioVisibility(userID, itemID string, req *dto.PortfolioVisibilityRequest) error

	// Upload operations
	UploadFile(userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error)
	GetUpload(uploadID string) (*models.Upload, error)
	GetUserUploads(userID string) ([]*models.Upload, error)
	GetEntityUploads(entityType, entityID string) ([]*models.Upload, error)
	DeleteUpload(userID, uploadID string) error
	GetUserStorageUsage(userID string) (*dto.StorageUsageResponse, error)

	// Combined operations
	CreatePortfolioWithUpload(userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error)
	DeletePortfolioWithUpload(userID, itemID string) error
	GetFeaturedPortfolio(limit int) (*dto.PortfolioListResponse, error)
	GetRecentPortfolio(limit int) (*dto.PortfolioListResponse, error)

	// Admin operations
	CleanOrphanedUploads() error
	GetPlatformUploadStats() (*dto.UploadStats, error)
}

type portfolioService struct {
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
	profileRepo   repositories.ProfileRepository
	fileConfig    dto.FileConfigPortfolio
	storage       storage.Storage           // Added storage
	imageProc     *imageprocessor.Processor // Added image processor
}

func NewPortfolioService(
	portfolioRepo repositories.PortfolioRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	storage storage.Storage,
) PortfolioService {
	return &portfolioService{
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		fileConfig:    config.PortfolioFileConfig,
		storage:       storage,
		imageProc:     imageprocessor.NewProcessor(config.AppConfig.Upload.ImageQuality),
	}
}

// Portfolio operations

func (s *portfolioService) CreatePortfolioItem(userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error) {
	// Validate user owns the model profile
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil {
		return nil, errors.New("model profile not found or access denied")
	}

	if modelProfile.ID != req.ModelID {
		return nil, errors.New("invalid model ID")
	}

	// Validate and process file
	upload, err := s.processUpload(userID, file, &dto.UploadRequest{
		EntityType: "portfolio",
		EntityID:   "", // Будет установлен после создания portfolio item
		FileType:   "image",
		Usage:      "portfolio_photo",
		IsPublic:   true,
	})
	if err != nil {
		return nil, err
	}

	// Create portfolio item
	portfolioItem := &models.PortfolioItem{
		ModelID:     req.ModelID,
		UploadID:    upload.ID,
		Title:       req.Title,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
	}

	if err := s.portfolioRepo.CreatePortfolioItem(portfolioItem); err != nil {
		// Clean up uploaded file if portfolio creation fails
		s.portfolioRepo.DeleteUpload(upload.ID)
		return nil, err
	}

	// Update upload with the actual portfolio item ID
	upload.EntityID = portfolioItem.ID
	if err := s.portfolioRepo.UpdateUpload(upload); err != nil {
		// If update fails, clean up both portfolio item and upload
		s.portfolioRepo.DeletePortfolioItem(portfolioItem.ID)
		s.portfolioRepo.DeleteUpload(upload.ID)
		return nil, err
	}

	return s.buildPortfolioResponse(portfolioItem, upload), nil
}

func (s *portfolioService) GetPortfolioItem(itemID string) (*dto.PortfolioResponse, error) {
	item, err := s.portfolioRepo.FindPortfolioItemByID(itemID)
	if err != nil {
		return nil, err
	}

	var upload *models.Upload
	if item.Upload != nil {
		upload = item.Upload
	} else {
		upload, _ = s.portfolioRepo.FindUploadByID(item.UploadID)
	}

	return s.buildPortfolioResponse(item, upload), nil
}

func (s *portfolioService) GetModelPortfolio(modelID string) ([]*dto.PortfolioResponse, error) {
	items, err := s.portfolioRepo.FindPortfolioByModel(modelID)
	if err != nil {
		return nil, err
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		var upload *models.Upload
		if item.Upload != nil {
			upload = item.Upload
		} else {
			upload, _ = s.portfolioRepo.FindUploadByID(item.UploadID)
		}
		responses = append(responses, s.buildPortfolioResponse(&item, upload))
	}

	return responses, nil
}

func (s *portfolioService) UpdatePortfolioItem(userID, itemID string, req *dto.UpdatePortfolioRequest) error {
	// Verify ownership
	item, err := s.portfolioRepo.FindPortfolioItemByID(itemID)
	if err != nil {
		return err
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	// Update fields
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.OrderIndex != nil {
		if err := s.portfolioRepo.UpdatePortfolioItemOrder(item, *req.OrderIndex); err != nil {
			return err
		}
	} else {
		if err := s.portfolioRepo.UpdatePortfolioItem(item); err != nil {
			return err
		}
	}

	return nil
}

func (s *portfolioService) UpdatePortfolioOrder(userID string, req *dto.ReorderPortfolioRequest) error {
	// Verify all items belong to user's model profile
	modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil {
		return errors.New("model profile not found")
	}

	for _, itemID := range req.ItemIDs {
		item, err := s.portfolioRepo.FindPortfolioItemByID(itemID)
		if err != nil {
			return err
		}
		if item.ModelID != modelProfile.ID {
			return errors.New("access denied for some items")
		}
	}

	return s.portfolioRepo.ReorderPortfolioItems(modelProfile.ID, req.ItemIDs)
}

func (s *portfolioService) DeletePortfolioItem(userID, itemID string) error {
	// Verify ownership
	item, err := s.portfolioRepo.FindPortfolioItemByID(itemID)
	if err != nil {
		return err
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	return s.portfolioRepo.DeletePortfolioItem(itemID)
}

func (s *portfolioService) GetPortfolioStats(modelID string) (*repositories.PortfolioStats, error) {
	return s.portfolioRepo.GetPortfolioStats(modelID)
}

func (s *portfolioService) TogglePortfolioVisibility(userID, itemID string, req *dto.PortfolioVisibilityRequest) error {
	// Verify ownership
	item, err := s.portfolioRepo.FindPortfolioItemByID(itemID)
	if err != nil {
		return err
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	// Get the upload associated with this portfolio item
	upload, err := s.portfolioRepo.FindUploadByID(item.UploadID)
	if err != nil {
		return err
	}

	// Update upload visibility
	upload.IsPublic = req.IsPublic
	if err := s.portfolioRepo.UpdateUpload(upload); err != nil {
		return err
	}

	return nil
}

// Upload operations

func (s *portfolioService) UploadFile(userID string, req *dto.UploadRequest, file *multipart.FileHeader) (*dto.UploadResponse, error) {
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

func (s *portfolioService) GetUpload(uploadID string) (*models.Upload, error) {
	return s.portfolioRepo.FindUploadByID(uploadID)
}

func (s *portfolioService) GetUserUploads(userID string) ([]*models.Upload, error) {
	uploads, err := s.portfolioRepo.FindUploadsByUser(userID)
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	var result []*models.Upload
	for i := range uploads {
		result = append(result, &uploads[i])
	}

	return result, nil
}

func (s *portfolioService) GetEntityUploads(entityType, entityID string) ([]*models.Upload, error) {
	uploads, err := s.portfolioRepo.FindUploadsByEntity(entityType, entityID)
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	var result []*models.Upload
	for i := range uploads {
		result = append(result, &uploads[i])
	}

	return result, nil
}

func (s *portfolioService) DeleteUpload(userID, uploadID string) error {
	upload, err := s.portfolioRepo.FindUploadByID(uploadID)
	if err != nil {
		return err
	}

	if upload.UserID != userID {
		return errors.New("access denied")
	}

	return s.portfolioRepo.DeleteUpload(uploadID)
}

func (s *portfolioService) GetUserStorageUsage(userID string) (*dto.StorageUsageResponse, error) {
	used, err := s.portfolioRepo.GetUserStorageUsage(userID)
	if err != nil {
		return nil, err
	}

	return &dto.StorageUsageResponse{
		Used:  used,
		Limit: s.fileConfig.MaxUserStorage,
	}, nil
}

// Combined operations

func (s *portfolioService) CreatePortfolioWithUpload(userID string, req *dto.CreatePortfolioRequest, file *multipart.FileHeader) (*dto.PortfolioResponse, error) {
	// This is essentially the same as CreatePortfolioItem
	return s.CreatePortfolioItem(userID, req, file)
}

func (s *portfolioService) DeletePortfolioWithUpload(userID, itemID string) error {
	// Verify ownership
	item, err := s.portfolioRepo.FindPortfolioItemByID(itemID)
	if err != nil {
		return err
	}

	modelProfile, err := s.profileRepo.FindModelProfileByUserID(userID)
	if err != nil || modelProfile.ID != item.ModelID {
		return errors.New("access denied")
	}

	return s.portfolioRepo.DeletePortfolioItemWithUpload(itemID)
}

func (s *portfolioService) GetFeaturedPortfolio(limit int) (*dto.PortfolioListResponse, error) {
	items, err := s.portfolioRepo.FindFeaturedPortfolioItems(limit)
	if err != nil {
		return nil, err
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		var upload *models.Upload
		if item.Upload != nil {
			upload = item.Upload
		} else {
			upload, _ = s.portfolioRepo.FindUploadByID(item.UploadID)
		}
		responses = append(responses, s.buildPortfolioResponse(&item, upload))
	}

	return &dto.PortfolioListResponse{
		Items: responses,
		Total: len(responses),
	}, nil
}

func (s *portfolioService) GetRecentPortfolio(limit int) (*dto.PortfolioListResponse, error) {
	items, err := s.portfolioRepo.FindRecentPortfolioItems(limit)
	if err != nil {
		return nil, err
	}

	var responses []*dto.PortfolioResponse
	for _, item := range items {
		var upload *models.Upload
		if item.Upload != nil {
			upload = item.Upload
		} else {
			upload, _ = s.portfolioRepo.FindUploadByID(item.UploadID)
		}
		responses = append(responses, s.buildPortfolioResponse(&item, upload))
	}

	return &dto.PortfolioListResponse{
		Items: responses,
		Total: len(responses),
	}, nil
}

// Admin operations

func (s *portfolioService) CleanOrphanedUploads() error {
	return s.portfolioRepo.CleanOrphanedUploads()
}

func (s *portfolioService) GetPlatformUploadStats() (*dto.UploadStats, error) {
	// This would require additional repository methods
	// For now, return placeholder stats
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

func (s *portfolioService) processUpload(userID string, file *multipart.FileHeader, req *dto.UploadRequest) (*models.Upload, error) {
	uploadSvc := NewUploadService(s.portfolioRepo, s.userRepo, s.profileRepo, s.storage)
	upload, err := uploadSvc.UploadFile(userID, req, file)
	if err != nil {
		return nil, err
	}

	// Convert DTO back to model
	return s.portfolioRepo.FindUploadByID(upload.ID)
}

func (s *portfolioService) isValidFileType(mimeType string) bool {
	for _, allowedType := range s.fileConfig.AllowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

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

func (s *portfolioService) getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "file"
}

func (s *portfolioService) validateEntityAccess(userID, entityType, entityID string) error {
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

func (s *portfolioService) buildPortfolioResponse(item *models.PortfolioItem, upload *models.Upload) *dto.PortfolioResponse {
	response := &dto.PortfolioResponse{
		ID:          item.ID,
		ModelID:     item.ModelID,
		Title:       item.Title,
		Description: item.Description,
		OrderIndex:  item.OrderIndex,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}

	if upload != nil {
		response.Upload = &dto.UploadResponse{
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

	return response
}

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

func (s *portfolioService) generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

func (s *portfolioService) generateFileURL(upload *models.Upload) string {
	ctx := context.Background()
	url, err := s.storage.GetURL(ctx, upload.Path)
	if err != nil {
		// Fallback to default URL format
		return fmt.Sprintf("/api/v1/files/%s", upload.ID)
	}
	return url
}

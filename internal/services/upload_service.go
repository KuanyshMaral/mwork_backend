package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/storage"
	"mwork_backend/internal/types"
	"mwork_backend/pkg/apperrors"

	"gorm.io/gorm"
)

// ============================================
// УНИВЕРСАЛЬНЫЙ UPLOAD SERVICE
// ============================================

type UploadService interface {
	// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
	// Универсальная загрузка файла для любого модуля
	UploadFile(ctx context.Context, db *gorm.DB, req *dto.UniversalUploadRequest) (*dto.UploadResponse, error)
	// ▲▲▲ ИЗМЕНЕНО (Проблема 4) ▲▲▲

	// Получение информации о файле
	GetUpload(db *gorm.DB, uploadID string) (*models.Upload, error)

	// Получение файлов пользователя
	GetUserUploads(db *gorm.DB, userID string, filters *types.UploadFilters) ([]*models.Upload, error)

	// Получение файлов сущности
	GetEntityUploads(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error)

	// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
	// Удаление файла
	DeleteUpload(ctx context.Context, db *gorm.DB, userID, uploadID string) error
	// ▲▲▲ ИЗМЕНЕНО (Проблема 4) ▲▲▲

	// Получение использования хранилища
	GetUserStorageUsage(db *gorm.DB, userID string) (*dto.StorageUsageResponse, error)

	// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
	// Административные функции
	// CleanOrphanedUploads(db *gorm.DB) error // Удалено
	GetPlatformUploadStats(db *gorm.DB) (*dto.UploadStats, error)
	// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲
}

type uploadService struct {
	uploadRepo repositories.UploadRepository
	storage    storage.Storage
	config     *UploadConfig
}

// ============================================
// КОНФИГУРАЦИЯ
// ============================================

type UploadConfig struct {
	// Общие настройки
	MaxFileSize    int64
	MaxUserStorage int64

	// Настройки для разных модулей
	Modules map[string]*ModuleConfig
}

type ModuleConfig struct {
	AllowedTypes  []string       // MIME-типы
	AllowedUsages []string       // Назначения файлов
	MaxFileSize   int64          // Переопределение размера
	ImageQuality  int            // Качество для изображений
	Validation    ValidationFunc // Кастомная валидация
}

type ValidationFunc func(db *gorm.DB, userID string, req *dto.UniversalUploadRequest) error

// ============================================
// КОНСТРУКТОР
// ============================================

func NewUploadService(
	uploadRepo repositories.UploadRepository,
	storage storage.Storage,
	config *UploadConfig,
) UploadService {
	if config == nil {
		config = GetDefaultUploadConfig()
	}

	return &uploadService{
		uploadRepo: uploadRepo,
		storage:    storage,
		config:     config,
	}
}

// ============================================
// ОСНОВНЫЕ МЕТОДЫ
// ============================================

// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
func (s *uploadService) UploadFile(ctx context.Context, db *gorm.DB, req *dto.UniversalUploadRequest) (*dto.UploadResponse, error) {
	// Валидация модуля
	moduleConfig, exists := s.config.Modules[req.Module]
	if !exists {
		return nil, apperrors.NewBadRequestError(fmt.Sprintf("unknown module: %s", req.Module))
	}

	// ⭐️ ГИБКОСТЬ ТРАНЗАКЦИЙ: Определяем, кто управляет коммитом
	// Если db.Statement.Context не nil, мы уже находимся в транзакции верхнего уровня.
	inTransaction := db.Statement.Context != nil

	// Если не в транзакции, начинаем новую, чтобы обеспечить атомарность
	if !inTransaction {
		db = db.Begin()
		if db.Error != nil {
			return nil, apperrors.InternalError(db.Error)
		}
		defer db.Rollback() // Откат, если что-то пойдет не так
	}
	// ⭐️ (db теперь используется как текущая рабочая транзакция/пул)

	// Валидация файла
	if err := s.validateFile(req.File, moduleConfig); err != nil {
		return nil, err
	}

	// Проверка лимитов хранилища
	if err := s.checkStorageLimits(db, req.UserID, req.File.Size); err != nil { // ИСПОЛЬЗУЕМ db
		return nil, err
	}

	// Кастомная валидация модуля
	if moduleConfig.Validation != nil {
		if err := moduleConfig.Validation(db, req.UserID, req); err != nil { // ИСПОЛЬЗУЕМ db
			return nil, err
		}
	}

	// Создаём запись в БД
	upload, err := s.createUploadRecord(db, req, moduleConfig) // ИСПОЛЬЗУЕМ db
	if err != nil {
		return nil, err
	}

	// Сохраняем файл в storage
	src, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	if err := s.storage.Save(ctx, upload.Path, src, upload.MimeType); err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}

	// ⭐️ КОММИТ ТОЛЬКО ЕСЛИ МЫ САМИ НАЧАЛИ ТРАНЗАКЦИЮ
	if !inTransaction {
		if err := db.Commit().Error; err != nil {
			// Откатываем файл из storage
			if delErr := s.storage.Delete(ctx, upload.Path); delErr != nil {
				// Логируем критическую ошибку, но не возвращаем её
				fmt.Printf("CRITICAL: failed to rollback file save: %v\n", delErr)
			}
			return nil, apperrors.InternalError(err)
		}
	}
	// ⭐️ КОНЕЦ КОММИТА

	// Асинхронная обработка
	if strings.HasPrefix(upload.MimeType, "image/") {
		go s.processImageAsync(upload, moduleConfig.ImageQuality)
	}

	return s.buildUploadResponse(ctx, upload), nil
}

func (s *uploadService) GetUpload(db *gorm.DB, uploadID string) (*models.Upload, error) {
	upload, err := s.uploadRepo.FindByID(db, uploadID)
	if err != nil {
		return nil, handleUploadError(err)
	}
	return upload, nil
}

func (s *uploadService) GetUserUploads(db *gorm.DB, userID string, filters *types.UploadFilters) ([]*models.Upload, error) {
	return s.uploadRepo.FindByUser(db, userID, filters)
}

func (s *uploadService) GetEntityUploads(db *gorm.DB, entityType, entityID string) ([]*models.Upload, error) {
	return s.uploadRepo.FindByEntity(db, entityType, entityID)
}

// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
func (s *uploadService) DeleteUpload(ctx context.Context, db *gorm.DB, userID, uploadID string) error {
	// ⭐️ ГИБКОСТЬ ТРАНЗАКЦИЙ: Определяем, кто управляет коммитом
	inTransaction := db.Statement.Context != nil
	if !inTransaction {
		db = db.Begin()
		if db.Error != nil {
			return apperrors.InternalError(db.Error)
		}
		defer db.Rollback()
	}
	// ⭐️ (db теперь используется как текущая рабочая транзакция/пул)

	upload, err := s.uploadRepo.FindByID(db, uploadID) // ИСПОЛЬЗУЕМ db
	if err != nil {
		return handleUploadError(err)
	}

	if upload.UserID != userID {
		return apperrors.NewForbiddenError("access denied")
	}

	if err := s.uploadRepo.Delete(db, uploadID); err != nil { // ИСПОЛЬЗУЕМ db
		return apperrors.InternalError(err)
	}

	// ⭐️ КОММИТ ТОЛЬКО ЕСЛИ МЫ САМИ НАЧАЛИ ТРАНЗАКЦИЮ
	if !inTransaction {
		if err := db.Commit().Error; err != nil {
			return apperrors.InternalError(err)
		}
	}
	// ⭐️ КОНЕЦ КОММИТА

	// Удаляем из storage (после успешного коммита БД)
	if err := s.storage.Delete(ctx, upload.Path); err != nil {
		// Логируем, но не возвращаем ошибку, т.к. запись в БД уже удалена
		log.Printf("Failed to delete file from storage: %v", err)
	}

	if strings.HasPrefix(upload.MimeType, "image/") {
		// Запускаем очистку в фоне
		go s.deleteResizedVersions(context.Background(), upload.Path)
	}

	return nil
}

func (s *uploadService) GetUserStorageUsage(db *gorm.DB, userID string) (*dto.StorageUsageResponse, error) {
	used, err := s.uploadRepo.GetUserStorageUsage(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &dto.StorageUsageResponse{
		Used:  used,
		Limit: s.config.MaxUserStorage,
	}, nil
}

// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
/*
func (s *uploadService) CleanOrphanedUploads(db *gorm.DB) error {
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// Эта логика должна быть заменена на DI, где каждый
	// модуль предоставляет свою функцию проверки
	if err := s.uploadRepo.CleanOrphaned(tx); err != nil {
		return apperrors.InternalError(err)
	}

	return tx.Commit().Error
}
*/
// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲

func (s *uploadService) GetPlatformUploadStats(db *gorm.DB) (*dto.UploadStats, error) {
	// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
	// Заменили s.uploadRepo.GetStats(db) на кастомную логику,
	// так как GetStats был удален из репозитория вместе с CleanOrphaned
	// (Если GetStats был в другом репозитории, верните как было)

	// ПРЕДПОЛАГАЯ, что GetStats - это отдельный метод, который вы хотите сохранить.
	// Если GetStats был в upload_repository.go, его нужно вернуть туда.
	// Я верну его в upload_repository.go, но БЕЗ CleanOrphaned.
	// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲

	// Предполагаем, что GetStats остался в репозитории
	statsMap, err := s.uploadRepo.GetStats(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	// Ручное преобразование map[string]interface{} в dto.UploadStats
	// Это хрупко; лучше, чтобы GetStats возвращал DTO
	stats := &dto.UploadStats{}
	if v, ok := statsMap["total_uploads"].(int64); ok {
		stats.TotalUploads = v
	}
	if v, ok := statsMap["total_size"].(int64); ok {
		stats.TotalSize = v
	}
	if v, ok := statsMap["by_module"].(map[string]int64); ok {
		stats.ByModule = v
	}
	if v, ok := statsMap["by_file_type"].(map[string]int64); ok {
		stats.ByFileType = v
	}
	if v, ok := statsMap["active_users"].(int64); ok {
		stats.ActiveUsers = v
	}
	if v, ok := statsMap["storage_used"].(int64); ok {
		stats.StorageUsed = v
	}
	// stats.StorageLimit = s.config.MaxUserStorage // Можно добавить

	return stats, nil
}

// ============================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ============================================

func (s *uploadService) validateFile(file *multipart.FileHeader, config *ModuleConfig) error {
	// Проверка размера
	maxSize := config.MaxFileSize
	if maxSize == 0 {
		maxSize = s.config.MaxFileSize
	}

	if file.Size > maxSize {
		return apperrors.ErrFileTooLarge
	}

	// Проверка MIME-типа
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = getMimeTypeFromFilename(file.Filename)
	}

	allowed := false
	for _, allowedType := range config.AllowedTypes {
		if mimeType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		return apperrors.ErrInvalidFileType
	}

	return nil
}

func (s *uploadService) checkStorageLimits(db *gorm.DB, userID string, fileSize int64) error {
	currentUsage, err := s.uploadRepo.GetUserStorageUsage(db, userID)
	if err != nil {
		return apperrors.InternalError(err)
	}

	if currentUsage+fileSize > s.config.MaxUserStorage {
		return apperrors.ErrStorageLimitExceeded
	}

	return nil
}

func (s *uploadService) createUploadRecord(db *gorm.DB, req *dto.UniversalUploadRequest, config *ModuleConfig) (*models.Upload, error) {
	// Проверка usage
	if !contains(config.AllowedUsages, req.Usage) {
		return nil, apperrors.ErrInvalidUploadUsage
	}

	// Генерация пути
	mimeType := req.File.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = getMimeTypeFromFilename(req.File.Filename)
	}

	fileExt := filepath.Ext(req.File.Filename)
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), generateSecureRandomString(8), fileExt)
	filePath := filepath.Join(req.Module, req.EntityType, fileName)

	// ▼▼▼ ИСПРАВЛЕНИЕ ОШИБКИ 1 ▼▼▼
	// Преобразуем map[string]string в models.JSONMap (map[string]interface{})
	var metadata models.JSONMap
	if req.Metadata != nil {
		metadata = make(models.JSONMap, len(req.Metadata))
		for k, v := range req.Metadata {
			metadata[k] = v // string неявно преобразуется в interface{}
		}
	}
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

	upload := &models.Upload{
		UserID:     req.UserID,
		Module:     req.Module,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		FileType:   getFileTypeFromMIME(mimeType),
		Usage:      req.Usage,
		Path:       filePath,
		MimeType:   mimeType,
		Size:       req.File.Size,
		IsPublic:   req.IsPublic,
		Metadata:   metadata, // <-- Используем преобразованную карту
	}

	if err := s.uploadRepo.Create(db, upload); err != nil {
		return nil, apperrors.InternalError(err)
	}

	return upload, nil
}

// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
func (s *uploadService) buildUploadResponse(ctx context.Context, upload *models.Upload) *dto.UploadResponse {
	url, err := s.storage.GetURL(ctx, upload.Path)
	if err != nil {
		url = fmt.Sprintf("/api/v1/files/%s", upload.ID)
	}

	// ▼▼▼ ИСПРАВЛЕНИЕ ОШИБКИ 2 ▼▼▼
	// Преобразуем models.JSONMap (map[string]interface{}) обратно в map[string]string
	var metadata map[string]string
	if upload.Metadata != nil {
		metadata = make(map[string]string, len(upload.Metadata))
		for k, v := range upload.Metadata {
			// Используем type assertion для безопасного преобразования
			if vStr, ok := v.(string); ok {
				metadata[k] = vStr
			}
			// (Примечание: если в JSONMap есть не-строковые значения, они будут проигнорированы)
		}
	}
	// ▲▲▲ ИСПРАВЛЕНО ▲▲▲

	return &dto.UploadResponse{
		ID:         upload.ID,
		UserID:     upload.UserID,
		Module:     upload.Module,
		EntityType: upload.EntityType,
		EntityID:   upload.EntityID,
		FileType:   upload.FileType,
		Usage:      upload.Usage,
		URL:        url,
		MimeType:   upload.MimeType,
		Size:       upload.Size,
		IsPublic:   upload.IsPublic,
		Metadata:   metadata, // <-- Используем преобразованную карту
		CreatedAt:  upload.CreatedAt,
	}
}

// ▼▼▼ ИМЕНЕНО (Проблема 3) ▼▼▼
func (s *uploadService) processImageAsync(upload *models.Upload, quality int) {
	// Асинхронная обработка изображений
	// Используем Background(), т.к. родительский запрос уже завершен.
	// В идеале - использовать очередь задач (Asynq, Machinery)
	ctx := context.Background()
	sizes := []string{"thumbnail", "small", "medium"}

	for _, size := range sizes {
		// Получаем оригинальный файл из storage
		src, err := s.storage.Get(ctx, upload.Path)
		if err != nil {
			log.Printf("ERROR: processImageAsync: failed to get file from storage: %v", err)
			return // Если не можем получить файл, прекращаем
		}

		// Здесь должна быть логика ресайза (src -> resized)
		// ...
		// resized := s.imageProc.ProcessImage(src, size, quality)
		// ...
		// src.Close() // Закрываем оригинальный ридер

		// Mock: Предполагаем, что 'resized' - это io.Reader с ресайзнутым изображением
		// s.storage.Save(ctx, resizedPath, resized, "image/jpeg")
		// resized.Close() // Закрываем ридер ресайзнутого изображения

		// Временная заглушка, т.к. нет библиотеки ресайза
		log.Printf("INFO: Simulating resize for %s to size %s", upload.Path, size)
		resizedPath := getResizedPath(upload.Path, size)
		log.Printf("INFO: Resized path would be %s", resizedPath)

		src.Close() // Закрываем в конце итерации
	}
}

func (s *uploadService) deleteResizedVersions(ctx context.Context, originalPath string) {
	// ▲▲▲ ИЗМЕНЕНО (Проблема 4) ▲▲▲
	// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
	// ctx := context.TODO() // Контекст из параметров
	// ▲▲▲ ИЗМЕНЕНО (Проблема 4) ▲▲▲
	sizes := []string{"thumbnail", "small", "medium"}

	for _, size := range sizes {
		resizedPath := getResizedPath(originalPath, size)
		if err := s.storage.Delete(ctx, resizedPath); err != nil {
			log.Printf("Failed to delete resized version %s: %v", resizedPath, err)
		}
	}
}

// ============================================
// КОНФИГУРАЦИИ ПО УМОЛЧАНИЮ
// ============================================

func GetDefaultUploadConfig() *UploadConfig {
	return &UploadConfig{
		MaxFileSize:    50 * 1024 * 1024,  // 50MB
		MaxUserStorage: 100 * 1024 * 1024, // 100MB
		Modules: map[string]*ModuleConfig{
			"portfolio": {
				AllowedTypes:  []string{"image/jpeg", "image/png", "image/gif", "video/mp4"},
				AllowedUsages: []string{"avatar", "portfolio_photo", "portfolio_video"},
				MaxFileSize:   50 * 1024 * 1024,
				ImageQuality:  85,
			},
			"chat": {
				AllowedTypes:  []string{"image/jpeg", "image/png", "image/gif", "video/mp4", "application/pdf", "application/msword"},
				AllowedUsages: []string{"message_attachment"},
				MaxFileSize:   20 * 1024 * 1024, // 20MB для чата
				ImageQuality:  80,
			},
			"casting": {
				AllowedTypes:  []string{"image/jpeg", "image/png", "application/pdf"},
				AllowedUsages: []string{"casting_attachment", "requirement_photo"},
				MaxFileSize:   10 * 1024 * 1024,
				ImageQuality:  85,
			},
			"profile": {
				AllowedTypes:  []string{"image/jpeg", "image/png"},
				AllowedUsages: []string{"avatar", "cover_photo"},
				MaxFileSize:   5 * 1024 * 1024,
				ImageQuality:  90,
			},
		},
	}
}

// ============================================
// УТИЛИТЫ
// ============================================

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
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func getFileTypeFromMIME(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		return "video"
	} else if strings.HasPrefix(mimeType, "application/pdf") {
		return "document"
	}
	return "file"
}

func getResizedPath(originalPath, size string) string {
	ext := filepath.Ext(originalPath)
	nameWithoutExt := strings.TrimSuffix(originalPath, ext)
	return fmt.Sprintf("%s_%s%s", nameWithoutExt, size, ext)
}

func generateSecureRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func handleUploadError(err error) error {
	if err == gorm.ErrRecordNotFound {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

package services

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"mwork_front_fn/internal/models"
	"mwork_front_fn/internal/repositories"
)

type UploadService interface {
	UploadFile(ctx context.Context, userID, entityType, entityID, usage string, fileHeader *multipart.FileHeader) (*models.Upload, error)
}

type uploadService struct {
	repo         repositories.UploadRepository
	uploadFolder string
	publicBase   string
}

func NewUploadService(repo repositories.UploadRepository, uploadFolder, publicBase string) UploadService {
	return &uploadService{
		repo:         repo,
		uploadFolder: uploadFolder, // Например: "./uploads"
		publicBase:   publicBase,   // Например: "/uploads"
	}
}

func (s *uploadService) UploadFile(ctx context.Context, userID, entityType, entityID, usage string, fileHeader *multipart.FileHeader) (*models.Upload, error) {
	fileID := uuid.New().String()

	// Открытие загруженного файла
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	defer src.Close()

	// Определение MIME-типа
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("unable to read file header: %w", err)
	}
	mimeType := http.DetectContentType(buffer)

	// Возвращаемся к началу
	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("unable to reset file reader: %w", err)
	}

	ext, _ := mime.ExtensionsByType(mimeType)
	fileExt := ".bin"
	if len(ext) > 0 {
		fileExt = ext[0]
	} else {
		// fallback к расширению по имени
		fileExt = filepath.Ext(fileHeader.Filename)
	}

	fileType := "unknown"
	if strings.HasPrefix(mimeType, "image/") {
		fileType = "image"
	} else if strings.HasPrefix(mimeType, "video/") {
		fileType = "video"
	} else if strings.HasPrefix(mimeType, "application/") {
		fileType = "document"
	}

	// Путь к файлу
	fileName := fmt.Sprintf("%s%s", fileID, fileExt)
	relativePath := fmt.Sprintf("%s/%s/%s", s.publicBase, entityType, fileName) // например: /uploads/model_profile/abc.jpg
	fullPath := filepath.Join(s.uploadFolder, entityType)

	// Убедиться, что папка существует
	err = os.MkdirAll(fullPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("unable to create upload folder: %w", err)
	}

	// Сохранение файла
	dstPath := filepath.Join(fullPath, fileName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return nil, fmt.Errorf("unable to save file: %w", err)
	}

	upload := &models.Upload{
		ID:         fileID,
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
		FileType:   fileType,
		Usage:      usage,
		Path:       relativePath,
		MimeType:   mimeType,
		Size:       fileHeader.Size,
		IsPublic:   true,
		CreatedAt:  time.Now(),
	}

	err = s.repo.Save(ctx, upload)
	if err != nil {
		return nil, fmt.Errorf("unable to save upload record: %w", err)
	}

	return upload, nil
}

package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/storage"
	"mwork_backend/pkg/apperrors"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	*BaseHandler
	storage       storage.Storage
	portfolioRepo repositories.PortfolioRepository
}

func NewFileHandler(base *BaseHandler, storage storage.Storage, portfolioRepo repositories.PortfolioRepository) *FileHandler {
	return &FileHandler{
		BaseHandler:   base,
		storage:       storage,
		portfolioRepo: portfolioRepo,
	}
}

func (h *FileHandler) RegisterRoutes(r *gin.RouterGroup) {
	files := r.Group("/files")
	{
		// Public file serving
		files.GET("/:uploadId", h.ServeFile)
		files.GET("/:uploadId/:size", h.ServeResizedImage)

		// Protected file operations
		files.GET("/:uploadId/signed-url", middleware.AuthMiddleware(), h.GetSignedURL)
		files.HEAD("/:uploadId", h.CheckFileExists)
	}
}

// ServeFile serves a file by upload ID
func (h *FileHandler) ServeFile(c *gin.Context) {
	uploadID := c.Param("uploadId")

	// Get upload metadata from database
	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	upload, err := h.portfolioRepo.FindUploadByID(h.GetDB(c), uploadID)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		apperrors.HandleError(c, apperrors.NewNotFoundError("File not found"))
		return
	}

	// Check if file is public or user has access
	if !upload.IsPublic {
		userID, exists := c.Get("userID")
		if !exists || userID.(string) != upload.UserID {
			// Check if user is admin
			userRole, roleExists := c.Get("userRole")
			if !roleExists || userRole.(string) != string(models.UserRoleAdmin) {
				apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied"))
				return
			}
		}
	}

	// Get file from storage
	reader, err := h.storage.Get(c.Request.Context(), upload.Path)
	if err != nil {
		apperrors.HandleError(c, apperrors.NewNotFoundError("File not found in storage"))
		return
	}
	defer reader.Close()

	// Set headers
	c.Header("Content-Type", upload.MimeType)
	c.Header("Content-Length", strconv.FormatInt(upload.Size, 10))
	c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	c.Header("ETag", fmt.Sprintf(`"%s"`, upload.ID))

	// Set Content-Disposition for downloads
	if c.Query("download") == "true" {
		filename := filepath.Base(upload.Path)
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	} else {
		c.Header("Content-Disposition", "inline")
	}

	// Stream file to client
	if _, err := io.Copy(c.Writer, reader); err != nil {
		// Log error but don't send response (headers already sent)
		c.Error(err)
	}
}

// ServeResizedImage serves a resized version of an image
func (h *FileHandler) ServeResizedImage(c *gin.Context) {
	uploadID := c.Param("uploadId")
	size := c.Param("size")

	// Validate size parameter
	validSizes := map[string]bool{
		"thumbnail": true,
		"small":     true,
		"medium":    true,
		"large":     true,
	}

	if !validSizes[size] {
		apperrors.HandleError(c, apperrors.NewBadRequestError("Invalid size parameter"))
		return
	}

	// Get upload metadata
	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	upload, err := h.portfolioRepo.FindUploadByID(h.GetDB(c), uploadID)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		apperrors.HandleError(c, apperrors.NewNotFoundError("File not found"))
		return
	}

	// Check if it's an image
	if !strings.HasPrefix(upload.MimeType, "image/") {
		apperrors.HandleError(c, apperrors.NewBadRequestError("File is not an image"))
		return
	}

	// Check access permissions
	if !upload.IsPublic {
		userID, exists := c.Get("userID")
		if !exists || userID.(string) != upload.UserID {
			userRole, roleExists := c.Get("userRole")
			if !roleExists || userRole.(string) != string(models.UserRoleAdmin) {
				apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied"))
				return
			}
		}
	}

	// Check if resized version exists in storage
	resizedPath := h.getResizedPath(upload.Path, size)
	exists, _ := h.storage.Exists(c.Request.Context(), resizedPath)

	var reader io.ReadCloser
	if exists {
		// Serve existing resized version
		reader, err = h.storage.Get(c.Request.Context(), resizedPath)
		if err != nil {
			apperrors.HandleError(c, apperrors.NewInternalServerError("Failed to retrieve resized image"))
			return
		}
	} else {
		// Serve original (resizing on-the-fly can be added later)
		reader, err = h.storage.Get(c.Request.Context(), upload.Path)
		if err != nil {
			apperrors.HandleError(c, apperrors.NewNotFoundError("File not found in storage"))
			return
		}
	}
	defer reader.Close()

	// Set headers
	c.Header("Content-Type", upload.MimeType)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("ETag", fmt.Sprintf(`"%s-%s"`, upload.ID, size))
	c.Header("Content-Disposition", "inline")

	// Stream file
	if _, err := io.Copy(c.Writer, reader); err != nil {
		c.Error(err)
	}
}

// GetSignedURL generates a temporary signed URL for private files
func (h *FileHandler) GetSignedURL(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	uploadID := c.Param("uploadId")

	// Get upload metadata
	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	upload, err := h.portfolioRepo.FindUploadByID(h.GetDB(c), uploadID)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		apperrors.HandleError(c, apperrors.NewNotFoundError("File not found"))
		return
	}

	// Check ownership
	if upload.UserID != userID {
		userRole, roleExists := c.Get("userRole")
		if !roleExists || userRole.(string) != string(models.UserRoleAdmin) {
			apperrors.HandleError(c, apperrors.NewForbiddenError("Access denied"))
			return
		}
	}

	// Parse expiry duration (default 1 hour)
	expiryStr := c.DefaultQuery("expiry", "1h")
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		expiry = time.Hour
	}

	// Generate signed URL
	signedURL, err := h.storage.GetSignedURL(c.Request.Context(), upload.Path, expiry)
	if err != nil {
		apperrors.HandleError(c, apperrors.NewInternalServerError("Failed to generate signed URL"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        signedURL,
		"expires_at": time.Now().Add(expiry).Unix(),
	})
}

// CheckFileExists checks if a file exists (HEAD request)
func (h *FileHandler) CheckFileExists(c *gin.Context) {
	uploadID := c.Param("uploadId")

	// Get upload metadata
	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	upload, err := h.portfolioRepo.FindUploadByID(h.GetDB(c), uploadID)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// Check if file exists in storage
	exists, err := h.storage.Exists(c.Request.Context(), upload.Path)
	if err != nil || !exists {
		c.Status(http.StatusNotFound)
		return
	}

	// Set headers
	c.Header("Content-Type", upload.MimeType)
	c.Header("Content-Length", strconv.FormatInt(upload.Size, 10))
	c.Header("ETag", fmt.Sprintf(`"%s"`, upload.ID))
	c.Status(http.StatusOK)
}

// Helper function to get resized image path
func (h *FileHandler) getResizedPath(originalPath, size string) string {
	ext := filepath.Ext(originalPath)
	nameWithoutExt := strings.TrimSuffix(originalPath, ext)
	return fmt.Sprintf("%s_%s%s", nameWithoutExt, size, ext)
}

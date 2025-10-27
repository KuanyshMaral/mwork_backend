package old_shit

import (
	"github.com/gin-gonic/gin"
	"mwork_backend/internal/services"
	"net/http"
)

type UploadHandler struct {
	service services.UploadService
}

func NewUploadHandler(service services.UploadService) *UploadHandler {
	return &UploadHandler{service: service}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	// Получение параметров формы
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	entityType := c.PostForm("entityType")
	entityID := c.PostForm("entityId")
	usage := c.PostForm("usage")

	if entityType == "" || entityID == "" || usage == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entityType, entityId and usage are required"})
		return
	}

	// Получение userID (предполагается, что он в контексте)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	upload, err := h.service.UploadFile(c.Request.Context(), userID.(string), entityType, entityID, usage, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       upload.ID,
		"url":      upload.Path,
		"usage":    upload.Usage,
		"mimeType": upload.MimeType,
		"size":     upload.Size,
	})
}

package handlers

import (
	"mwork_front_fn/backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "mwork_front_fn/backend/dto"
)

type UploadHandler struct {
	service services.UploadService
}

func NewUploadHandler(service services.UploadService) *UploadHandler {
	return &UploadHandler{service: service}
}

// Upload godoc
// @Summary Загрузить файл
// @Description Загружает файл и сохраняет его с привязкой к сущности (entityType, entityId, usage). Требуется авторизация.
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Файл для загрузки"
// @Param entityType formData string true "Тип сущности (например, 'model_profile', 'casting')"
// @Param entityId formData string true "ID сущности"
// @Param usage formData string true "Назначение файла (например, 'avatar', 'portfolio')"
// @Success 200 {object} dto.UploadResponse
// @Failure 400 {object} models.ErrorResponse "Некорректный ввод"
// @Failure 401 {object} models.ErrorResponse "Неавторизован"
// @Failure 500 {object} models.ErrorResponse "Ошибка сервера"
// @Router /upload [post]
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

package handlers

import (
	"log"
	"net/http"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/internal/types"
	"mwork_backend/pkg/apperrors"

	"github.com/gin-gonic/gin"
)

// ============================================
// UPLOAD HANDLER
// ============================================

type UploadHandler struct {
	*BaseHandler
	uploadService services.UploadService
}

func NewUploadHandler(base *BaseHandler, uploadService services.UploadService) *UploadHandler {
	return &UploadHandler{
		BaseHandler:   base,
		uploadService: uploadService,
	}
}

// ============================================
// ROUTES
// ============================================

func (h *UploadHandler) RegisterRoutes(r *gin.RouterGroup) {
	uploads := r.Group("/uploads")
	uploads.Use(middleware.AuthMiddleware())
	{
		// Загрузка файлов
		uploads.POST("", h.UploadFile)
		uploads.POST("/multi", h.UploadMultipleFiles)

		// Получение информации
		uploads.GET("/:uploadId", h.GetUpload)
		uploads.GET("/user/me", h.GetMyUploads)
		uploads.GET("/entity/:entityType/:entityId", h.GetEntityUploads)

		// Удаление
		uploads.DELETE("/:uploadId", h.DeleteUpload)

		// Статистика
		uploads.GET("/storage/usage", h.GetStorageUsage)

		// Админ-функции
		admin := uploads.Group("/admin")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/stats", h.GetPlatformStats)
			// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
			// admin.POST("/clean", h.CleanOrphanedUploads) // Удалено, т.к. требует DI от всех модулей
			// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲
		}
	}
}

// ============================================
// HANDLERS
// ============================================

// UploadFile - загрузка одного файла
func (h *UploadHandler) UploadFile(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// Парсим multipart form
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		apperrors.HandleError(c, apperrors.NewBadRequestError("failed to parse form: "+err.Error()))
		return
	}

	// ▼▼▼ ИЗМЕНЕНО (Проблема 6) ▼▼▼
	// Получаем метаданные из полей формы, а не JSON
	var req dto.UniversalUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		apperrors.HandleError(c, apperrors.NewBadRequestError("invalid form data: "+err.Error()))
		return
	}
	// ▲▲▲ ИЗМЕНЕНО (Проблема 6) ▲▲▲

	// Валидация
	if err := h.validator.Validate(&req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	// Получаем файл
	fileHeader, err := c.FormFile("file")
	if err != nil {
		apperrors.HandleError(c, apperrors.NewBadRequestError("no file provided"))
		return
	}

	req.UserID = userID
	req.File = fileHeader

	// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
	// Загружаем файл, передавая контекст запроса
	response, err := h.uploadService.UploadFile(c.Request.Context(), h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО (Проблема 4) ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// UploadMultipleFiles - загрузка нескольких файлов
func (h *UploadHandler) UploadMultipleFiles(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// Парсим форму
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		apperrors.HandleError(c, apperrors.NewBadRequestError("failed to parse form: "+err.Error()))
		return
	}

	// ▼▼▼ ИЗМЕНЕНО (Проблема 6) ▼▼▼
	// Получаем метаданные из полей формы
	var baseReq dto.UniversalUploadRequest
	if err := c.ShouldBind(&baseReq); err != nil {
		apperrors.HandleError(c, apperrors.NewBadRequestError("invalid form data: "+err.Error()))
		return
	}

	// Добавляем валидацию для согласованности с UploadFile
	if err := h.validator.Validate(&baseReq); err != nil {
		h.HandleValidationError(c, err)
		return
	}
	// ▲▲▲ ИЗМЕНЕНО (Проблема 6) ▲▲▲

	// Получаем файлы
	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		apperrors.HandleError(c, apperrors.NewBadRequestError("no files provided"))
		return
	}

	baseReq.UserID = userID

	// ▼▼▼ ИЗМЕНЕНО (Проблема 5) ▼▼▼
	// Собираем успешные и проваленные загрузки
	var successfulUploads []*dto.UploadResponse
	var failedUploads []gin.H

	ctx := c.Request.Context() // (Проблема 4)

	for _, fileHeader := range files {
		req := baseReq
		req.File = fileHeader

		response, err := h.uploadService.UploadFile(ctx, h.GetDB(c), &req)
		if err != nil {
			// Логируем ошибку и добавляем в список проваленных
			log.Printf("Failed to upload file %s: %v", fileHeader.Filename, err)
			failedUploads = append(failedUploads, gin.H{
				"file":  fileHeader.Filename,
				"error": err.Error(),
			})
			continue
		}
		successfulUploads = append(successfulUploads, response)
	}

	response := gin.H{
		"successful_uploads": successfulUploads,
		"failed_uploads":     failedUploads,
		"success_count":      len(successfulUploads),
		"failed_count":       len(failedUploads),
	}

	// Если были ошибки, возвращаем 207 Multi-Status
	if len(failedUploads) > 0 {
		c.JSON(http.StatusMultiStatus, response)
	} else {
		// Если все успешно, возвращаем 201 Created (или 200 OK, 201 лучше для *создания*)
		c.JSON(http.StatusCreated, response)
	}
	// ▲▲▲ ИЗМЕНЕНО (Проблема 5) ▲▲▲
}

// GetUpload - получение информации о файле
func (h *UploadHandler) GetUpload(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	uploadID := c.Param("uploadId")

	upload, err := h.uploadService.GetUpload(h.GetDB(c), uploadID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// Проверка доступа
	if !upload.IsPublic && upload.UserID != userID {
		apperrors.HandleError(c, apperrors.NewForbiddenError("access denied"))
		return
	}

	c.JSON(http.StatusOK, upload)
}

// GetMyUploads - получение файлов текущего пользователя
func (h *UploadHandler) GetMyUploads(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	filters := &types.UploadFilters{
		Module:     c.Query("module"),
		EntityType: c.Query("entity_type"),
		EntityID:   c.Query("entity_id"),
		Usage:      c.Query("usage"),
		Limit:      ParseQueryInt(c, "limit", 50),
		Offset:     ParseQueryInt(c, "offset", 0),
	}

	uploads, err := h.uploadService.GetUserUploads(h.GetDB(c), userID, filters)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": uploads,
		"count":   len(uploads),
	})
}

// GetEntityUploads - получение файлов сущности
func (h *UploadHandler) GetEntityUploads(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	entityType := c.Param("entityType")
	entityID := c.Param("entityId")

	uploads, err := h.uploadService.GetEntityUploads(h.GetDB(c), entityType, entityID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// Фильтруем приватные файлы
	var filteredUploads []*dto.UploadResponse
	for _, upload := range uploads {
		if upload.IsPublic || upload.UserID == userID {
			// Преобразуем в response
			response := &dto.UploadResponse{
				ID:         upload.ID,
				UserID:     upload.UserID,
				Module:     upload.Module,
				EntityType: upload.EntityType,
				EntityID:   upload.EntityID,
				FileType:   upload.FileType,
				Usage:      upload.Usage,
				MimeType:   upload.MimeType,
				Size:       upload.Size,
				IsPublic:   upload.IsPublic,
				CreatedAt:  upload.CreatedAt,
			}
			filteredUploads = append(filteredUploads, response)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": filteredUploads,
		"count":   len(filteredUploads),
	})
}

// DeleteUpload - удаление файла
func (h *UploadHandler) DeleteUpload(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	uploadID := c.Param("uploadId")

	// ▼▼▼ ИЗМЕНЕНО (Проблема 4) ▼▼▼
	if err := h.uploadService.DeleteUpload(c.Request.Context(), h.GetDB(c), userID, uploadID); err != nil {
		h.HandleServiceError(c, err)
		return
	}
	// ▲▲▲ ИЗМЕНЕНО (Проблема 4) ▲▲▲

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}

// GetStorageUsage - получение использования хранилища
func (h *UploadHandler) GetStorageUsage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	usage, err := h.uploadService.GetUserStorageUsage(h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetPlatformStats - статистика платформы (админ)
func (h *UploadHandler) GetPlatformStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	stats, err := h.uploadService.GetPlatformUploadStats(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ▼▼▼ ИЗМЕНЕНО (Проблема 2) ▼▼▼
// CleanOrphanedUploads - очистка осиротевших файлов (админ)
/*
func (h *UploadHandler) CleanOrphanedUploads(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	if err := h.uploadService.CleanOrphanedUploads(h.GetDB(c)); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "orphaned uploads cleaned"})
}
*/
//
// ▲▲▲ ИЗМЕНЕНО (Проблема 2) ▲▲▲

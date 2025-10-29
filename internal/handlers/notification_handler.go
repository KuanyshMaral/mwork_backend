package handlers

import (
	"net/http"
	"strconv"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationService services.NotificationService
}

func NewNotificationHandler(notificationService services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

func (h *NotificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Protected routes - All authenticated users
	notifications := r.Group("/notifications")
	notifications.Use(middleware.AuthMiddleware())
	{
		notifications.POST("", h.CreateNotification)
		notifications.GET("", h.GetUserNotifications)
		notifications.GET("/:notificationId", h.GetNotification)
		notifications.PUT("/:notificationId/read", h.MarkAsRead)
		notifications.PUT("/read-all", h.MarkAllAsRead)
		notifications.PUT("/read-multiple", h.MarkMultipleAsRead)
		notifications.DELETE("/:notificationId", h.DeleteNotification)
		notifications.DELETE("", h.DeleteUserNotifications)
		notifications.GET("/stats", h.GetUserNotificationStats)
		notifications.GET("/unread-count", h.GetUnreadCount)
	}

	// Admin routes
	admin := r.Group("/admin/notifications")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.POST("/templates", h.CreateTemplate)
		admin.GET("/templates/:templateId", h.GetTemplate)
		admin.GET("/templates/type/:type", h.GetTemplateByType)
		admin.PUT("/templates/:templateId", h.UpdateTemplate)
		admin.DELETE("/templates/:templateId", h.DeleteTemplate)
		admin.GET("/templates", h.GetAllTemplates)
		admin.GET("", h.GetAllNotifications)
		admin.GET("/stats/platform", h.GetPlatformNotificationStats)
		admin.POST("/bulk-send", h.SendBulkNotification)
		admin.DELETE("/cleanup", h.CleanOldNotifications)
	}
}

// User notification handlers

func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	notification, err := h.notificationService.CreateNotification(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *NotificationHandler) GetNotification(c *gin.Context) {
	notificationID := c.Param("notificationId")

	notification, err := h.notificationService.GetNotification(notificationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *NotificationHandler) GetUserNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page := 1
	pageSize := 20

	if pageParam := c.Query("page"); pageParam != "" {
		if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pageSizeParam := c.Query("page_size"); pageSizeParam != "" {
		if parsed, err := strconv.Atoi(pageSizeParam); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	criteria := dto.NotificationCriteria{
		Page:     page,
		PageSize: pageSize,
	}

	response, err := h.notificationService.GetUserNotifications(userID, criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	notificationID := c.Param("notificationId")

	if err := h.notificationService.MarkAsRead(userID, notificationID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.notificationService.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func (h *NotificationHandler) MarkMultipleAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		NotificationIDs []string `json:"notification_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.notificationService.MarkMultipleAsRead(userID, req.NotificationIDs); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied for some notifications" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifications marked as read"})
}

func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID := middleware.GetUserID(c)
	notificationID := c.Param("notificationId")

	if err := h.notificationService.DeleteNotification(userID, notificationID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted successfully"})
}

func (h *NotificationHandler) DeleteUserNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.notificationService.DeleteUserNotifications(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications deleted successfully"})
}

func (h *NotificationHandler) GetUserNotificationStats(c *gin.Context) {
	userID := middleware.GetUserID(c)

	stats, err := h.notificationService.GetUserNotificationStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)

	count, err := h.notificationService.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// Admin template handlers

func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	var req dto.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.notificationService.CreateTemplate(adminID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "invalid notification type" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Template created successfully"})
}

func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	templateID := c.Param("templateId")

	template, err := h.notificationService.GetTemplate(templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *NotificationHandler) GetTemplateByType(c *gin.Context) {
	notificationType := c.Param("type")

	template, err := h.notificationService.GetTemplateByType(notificationType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *NotificationHandler) UpdateTemplate(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	templateID := c.Param("templateId")

	var req dto.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.notificationService.UpdateTemplate(adminID, templateID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template updated successfully"})
}

func (h *NotificationHandler) DeleteTemplate(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	templateID := c.Param("templateId")

	if err := h.notificationService.DeleteTemplate(adminID, templateID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

func (h *NotificationHandler) GetAllTemplates(c *gin.Context) {
	templates, err := h.notificationService.GetAllTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"total":     len(templates),
	})
}

// Admin notification handlers

func (h *NotificationHandler) GetAllNotifications(c *gin.Context) {
	page := 1
	pageSize := 50

	if pageParam := c.Query("page"); pageParam != "" {
		if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pageSizeParam := c.Query("page_size"); pageSizeParam != "" {
		if parsed, err := strconv.Atoi(pageSizeParam); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	criteria := dto.AdminNotificationCriteria{
		Page:     page,
		PageSize: pageSize,
	}

	response, err := h.notificationService.GetAllNotifications(criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *NotificationHandler) GetPlatformNotificationStats(c *gin.Context) {
	stats, err := h.notificationService.GetPlatformNotificationStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *NotificationHandler) SendBulkNotification(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	var req dto.SendBulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.notificationService.SendBulkNotification(adminID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "invalid notification type" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bulk notification sent successfully"})
}

func (h *NotificationHandler) CleanOldNotifications(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsed, err := strconv.Atoi(daysParam); err == nil && parsed > 0 {
			days = parsed
		}
	}

	if err := h.notificationService.CleanOldNotifications(days); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Old notifications cleaned successfully"})
}

package handlers

import (
	"net/http"
	// "strconv" // <-- No longer needed

	"mwork_backend/internal/middleware" // <-- Still needed for RegisterRoutes
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	*BaseHandler        // <-- 1. Embed BaseHandler
	notificationService services.NotificationService
}

// 2. Update the constructor
func NewNotificationHandler(base *BaseHandler, notificationService services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		BaseHandler:         base, // <-- 3. Assign it
		notificationService: notificationService,
	}
}

// RegisterRoutes remains unchanged
func (h *NotificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Protected routes - All authenticated users
	notifications := r.Group("/notifications")
	notifications.Use(middleware.AuthMiddleware())
	{
		notifications.POST("", h.CreateNotification)
		notifications.GET("/my", h.GetUserNotifications)
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

// --- User notification handlers ---

func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	// 4. Use GetAndAuthorizeUserID
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateNotificationRequest
	// 5. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	notification, err := h.notificationService.CreateNotification(h.GetDB(c), userID, &req)
	if err != nil {
		// 6. Use HandleServiceError
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *NotificationHandler) GetNotification(c *gin.Context) {
	// This route is protected, so we must check auth
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	notificationID := c.Param("notificationId")

	// ✅ DB: Используем h.GetDB(c)
	notification, err := h.notificationService.GetNotification(h.GetDB(c), notificationID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *NotificationHandler) GetUserNotifications(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// 7. Use ParsePagination
	page, pageSize := ParsePagination(c)

	criteria := dto.NotificationCriteria{
		Page:     page,
		PageSize: pageSize,
		// TODO: Добавить парсинг фильтров из c.Query()
		Filters: make(map[string]interface{}),
	}
	if c.Query("unread_only") == "true" {
		criteria.Filters["unread_only"] = true
	}

	// ✅ DB: Используем h.GetDB(c)
	response, err := h.notificationService.GetUserNotifications(h.GetDB(c), userID, criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	notificationID := c.Param("notificationId")

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.MarkAsRead(h.GetDB(c), userID, notificationID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.MarkAllAsRead(h.GetDB(c), userID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func (h *NotificationHandler) MarkMultipleAsRead(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		NotificationIDs []string `json:"notification_ids" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.MarkMultipleAsRead(h.GetDB(c), userID, req.NotificationIDs); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifications marked as read"})
}

func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	notificationID := c.Param("notificationId")

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.DeleteNotification(h.GetDB(c), userID, notificationID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted successfully"})
}

func (h *NotificationHandler) DeleteUserNotifications(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.DeleteUserNotifications(h.GetDB(c), userID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications deleted successfully"})
}

func (h *NotificationHandler) GetUserNotificationStats(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.notificationService.GetUserNotificationStats(h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	count, err := h.notificationService.GetUnreadCount(h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// --- Admin template handlers ---

func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateTemplateRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.CreateTemplate(h.GetDB(c), adminID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Template created successfully"})
}

func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	templateID := c.Param("templateId")

	// ✅ DB: Используем h.GetDB(c)
	template, err := h.notificationService.GetTemplate(h.GetDB(c), templateID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *NotificationHandler) GetTemplateByType(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	notificationType := c.Param("type")

	// ✅ DB: Используем h.GetDB(c)
	template, err := h.notificationService.GetTemplateByType(h.GetDB(c), notificationType)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *NotificationHandler) UpdateTemplate(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	templateID := c.Param("templateId")

	var req dto.UpdateTemplateRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.UpdateTemplate(h.GetDB(c), adminID, templateID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template updated successfully"})
}

func (h *NotificationHandler) DeleteTemplate(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	templateID := c.Param("templateId")

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.DeleteTemplate(h.GetDB(c), adminID, templateID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

func (h *NotificationHandler) GetAllTemplates(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	templates, err := h.notificationService.GetAllTemplates(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"total":     len(templates),
	})
}

// --- Admin notification handlers ---

func (h *NotificationHandler) GetAllNotifications(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	page, pageSize := ParsePagination(c)

	criteria := dto.AdminNotificationCriteria{
		Page:     page,
		PageSize: pageSize,
		// TODO: Добавить парсинг фильтров из c.Query()
		Filters: make(map[string]interface{}),
	}
	if c.Query("user_id") != "" {
		criteria.Filters["user_id"] = c.Query("user_id")
	}

	// ✅ DB: Используем h.GetDB(c)
	response, err := h.notificationService.GetAllNotifications(h.GetDB(c), criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *NotificationHandler) GetPlatformNotificationStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.notificationService.GetPlatformNotificationStats(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *NotificationHandler) SendBulkNotification(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.SendBulkNotificationRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Использу... h.GetDB(c)
	if err := h.notificationService.SendBulkNotification(h.GetDB(c), adminID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bulk notification sent successfully"})
}

func (h *NotificationHandler) CleanOldNotifications(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// 8. Use ParseQueryInt
	days := ParseQueryInt(c, "days", 30)
	if days <= 0 {
		days = 30
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.notificationService.CleanOldNotifications(h.GetDB(c), days); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Old notifications cleaned successfully"})
}

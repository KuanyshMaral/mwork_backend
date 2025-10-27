package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mwork_backend/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrNotificationNotFound    = errors.New("notification not found")
	ErrInvalidNotificationData = errors.New("invalid notification data")
)

// Константы типов уведомлений
const (
	NotificationTypeNewResponse          = "new_response"
	NotificationTypeNewMessage           = "new_message"
	NotificationTypeCastingMatch         = "casting_match"
	NotificationTypeResponseStatus       = "response_status"
	NotificationTypeSubscriptionExpiring = "subscription_expiring"
	NotificationTypeNewCasting           = "new_casting"
	NotificationTypeProfileView          = "profile_view"
)

type NotificationRepository interface {
	// Notification operations
	CreateNotification(notification *models.Notification) error
	CreateBulkNotifications(notifications []*models.Notification) error
	FindNotificationByID(id string) (*models.Notification, error)
	FindUserNotifications(userID string, criteria NotificationCriteria) ([]models.Notification, int64, error)
	MarkAsRead(notificationID string) error
	MarkAllAsRead(userID string) error
	MarkMultipleAsRead(notificationIDs []string) error
	DeleteNotification(id string) error
	DeleteUserNotifications(userID string) error
	DeleteReadNotifications(userID string, olderThan time.Time) error

	// Notification stats
	GetUserNotificationStats(userID string) (*NotificationStats, error)
	GetUnreadCount(userID string) (int64, error)

	// Template operations
	CreateNotificationTemplate(template *NotificationTemplate) error
	FindTemplateByType(notificationType string) (*NotificationTemplate, error)
	UpdateTemplate(template *NotificationTemplate) error

	// Admin operations
	FindAllNotifications(criteria AdminNotificationCriteria) ([]models.Notification, int64, error)
	GetPlatformNotificationStats() (*PlatformNotificationStats, error)
	CleanOldNotifications(days int) error

	// Factory methods for common notification types
	CreateNewResponseNotification(employerID, castingID, responseID, modelName string) error
	CreateResponseStatusNotification(modelID, castingTitle string, status models.ResponseStatus) error
	CreateCastingMatchNotification(modelID string, castingTitle string, score float64) error
	CreateNewMessageNotification(recipientID, senderName string, dialogID string) error
	CreateSubscriptionExpiringNotification(userID, planName string, daysRemaining int) error
	CreateBulkResponseNotifications(employerID string, responses []ResponseNotificationData) error
	CreateBulkCastingMatchNotifications(matches []CastingMatchNotificationData) error
}

type NotificationRepositoryImpl struct {
	db *gorm.DB
}

// Search criteria for notifications
type NotificationCriteria struct {
	UnreadOnly bool      `form:"unread_only"`
	Type       string    `form:"type"`
	DateFrom   time.Time `form:"date_from"`
	DateTo     time.Time `form:"date_to"`
	Page       int       `form:"page" binding:"min=1"`
	PageSize   int       `form:"page_size" binding:"min=1,max=100"`
}

// Admin search criteria
type AdminNotificationCriteria struct {
	UserID     string    `form:"user_id"`
	Type       string    `form:"type"`
	UnreadOnly bool      `form:"unread_only"`
	DateFrom   time.Time `form:"date_from"`
	DateTo     time.Time `form:"date_to"`
	Page       int       `form:"page" binding:"min=1"`
	PageSize   int       `form:"page_size" binding:"min=1,max=100"`
}

// Notification template
type NotificationTemplate struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"uniqueIndex" json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Variables []string  `json:"variables" gorm:"type:jsonb"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Notification statistics
type NotificationStats struct {
	TotalNotifications int64            `json:"total_notifications"`
	UnreadCount        int64            `json:"unread_count"`
	ReadCount          int64            `json:"read_count"`
	ByType             map[string]int64 `json:"by_type"`
	TodayCount         int64            `json:"today_count"`
	ThisWeekCount      int64            `json:"this_week_count"`
}

// Platform notification statistics
type PlatformNotificationStats struct {
	TotalNotifications int64                   `json:"total_notifications"`
	UnreadCount        int64                   `json:"unread_count"`
	TodayCount         int64                   `json:"today_count"`
	ThisWeekCount      int64                   `json:"this_week_count"`
	ByType             map[string]int64        `json:"by_type"`
	MostActiveUsers    []UserNotificationStats `json:"most_active_users"`
}

type UserNotificationStats struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Count  int64  `json:"count"`
	Unread int64  `json:"unread"`
}

// Additional structures for batch operations
type ResponseNotificationData struct {
	CastingID  string
	ResponseID string
	ModelName  string
}

type CastingMatchNotificationData struct {
	ModelID      string
	CastingTitle string
	Score        float64
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &NotificationRepositoryImpl{db: db}
}

// Notification operations

func (r *NotificationRepositoryImpl) CreateNotification(notification *models.Notification) error {
	// Validate notification data
	if err := r.validateNotification(notification); err != nil {
		return err
	}

	return r.db.Create(notification).Error
}

func (r *NotificationRepositoryImpl) CreateBulkNotifications(notifications []*models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Validate all notifications
	for _, notification := range notifications {
		if err := r.validateNotification(notification); err != nil {
			return err
		}
	}

	return r.db.CreateInBatches(notifications, 100).Error
}

func (r *NotificationRepositoryImpl) FindNotificationByID(id string) (*models.Notification, error) {
	var notification models.Notification
	err := r.db.First(&notification, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	return &notification, nil
}

func (r *NotificationRepositoryImpl) FindUserNotifications(userID string, criteria NotificationCriteria) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	query := r.db.Where("user_id = ?", userID)

	// Apply filters
	if criteria.UnreadOnly {
		query = query.Where("is_read = ?", false)
	}

	if criteria.Type != "" {
		query = query.Where("type = ?", criteria.Type)
	}

	if !criteria.DateFrom.IsZero() {
		query = query.Where("created_at >= ?", criteria.DateFrom)
	}

	if !criteria.DateTo.IsZero() {
		query = query.Where("created_at <= ?", criteria.DateTo)
	}

	// Get total count
	var total int64
	if err := query.Model(&models.Notification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *NotificationRepositoryImpl) MarkAsRead(notificationID string) error {
	result := r.db.Model(&models.Notification{}).Where("id = ?", notificationID).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepositoryImpl) MarkAllAsRead(userID string) error {
	result := r.db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	})

	return result.Error
}

func (r *NotificationRepositoryImpl) MarkMultipleAsRead(notificationIDs []string) error {
	if len(notificationIDs) == 0 {
		return nil
	}

	result := r.db.Model(&models.Notification{}).Where("id IN ?", notificationIDs).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	})

	return result.Error
}

func (r *NotificationRepositoryImpl) DeleteNotification(id string) error {
	result := r.db.Where("id = ?", id).Delete(&models.Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepositoryImpl) DeleteUserNotifications(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}

func (r *NotificationRepositoryImpl) DeleteReadNotifications(userID string, olderThan time.Time) error {
	return r.db.Where("user_id = ? AND is_read = ? AND created_at < ?", userID, true, olderThan).
		Delete(&models.Notification{}).Error
}

// Notification stats

func (r *NotificationRepositoryImpl) GetUserNotificationStats(userID string) (*NotificationStats, error) {
	var stats NotificationStats
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))

	// Total notifications
	if err := r.db.Model(&models.Notification{}).Where("user_id = ?", userID).
		Count(&stats.TotalNotifications).Error; err != nil {
		return nil, err
	}

	// Unread count
	if err := r.db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).
		Count(&stats.UnreadCount).Error; err != nil {
		return nil, err
	}

	// Read count
	stats.ReadCount = stats.TotalNotifications - stats.UnreadCount

	// Today count
	if err := r.db.Model(&models.Notification{}).Where("user_id = ? AND created_at >= ?", userID, todayStart).
		Count(&stats.TodayCount).Error; err != nil {
		return nil, err
	}

	// This week count
	if err := r.db.Model(&models.Notification{}).Where("user_id = ? AND created_at >= ?", userID, weekStart).
		Count(&stats.ThisWeekCount).Error; err != nil {
		return nil, err
	}

	// Count by type
	stats.ByType = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}

	err := r.db.Model(&models.Notification{}).Where("user_id = ?", userID).
		Select("type, COUNT(*) as count").
		Group("type").Scan(&typeStats).Error

	if err != nil {
		return nil, err
	}

	for _, ts := range typeStats {
		stats.ByType[ts.Type] = ts.Count
	}

	return &stats, nil
}

func (r *NotificationRepositoryImpl) GetUnreadCount(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// Template operations

func (r *NotificationRepositoryImpl) CreateNotificationTemplate(template *NotificationTemplate) error {
	return r.db.Create(template).Error
}

func (r *NotificationRepositoryImpl) FindTemplateByType(notificationType string) (*NotificationTemplate, error) {
	var template NotificationTemplate
	err := r.db.Where("type = ? AND is_active = ?", notificationType, true).First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}
	return &template, nil
}

func (r *NotificationRepositoryImpl) UpdateTemplate(template *NotificationTemplate) error {
	result := r.db.Model(template).Updates(map[string]interface{}{
		"title":      template.Title,
		"message":    template.Message,
		"variables":  template.Variables,
		"is_active":  template.IsActive,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("template not found")
	}
	return nil
}

// Admin operations

func (r *NotificationRepositoryImpl) FindAllNotifications(criteria AdminNotificationCriteria) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	query := r.db.Model(&models.Notification{})

	// Apply filters
	if criteria.UserID != "" {
		query = query.Where("user_id = ?", criteria.UserID)
	}

	if criteria.Type != "" {
		query = query.Where("type = ?", criteria.Type)
	}

	if criteria.UnreadOnly {
		query = query.Where("is_read = ?", false)
	}

	if !criteria.DateFrom.IsZero() {
		query = query.Where("created_at >= ?", criteria.DateFrom)
	}

	if !criteria.DateTo.IsZero() {
		query = query.Where("created_at <= ?", criteria.DateTo)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	err := query.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *NotificationRepositoryImpl) GetPlatformNotificationStats() (*PlatformNotificationStats, error) {
	var stats PlatformNotificationStats
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))

	// Total notifications
	if err := r.db.Model(&models.Notification{}).Count(&stats.TotalNotifications).Error; err != nil {
		return nil, err
	}

	// Unread count
	if err := r.db.Model(&models.Notification{}).Where("is_read = ?", false).
		Count(&stats.UnreadCount).Error; err != nil {
		return nil, err
	}

	// Today count
	if err := r.db.Model(&models.Notification{}).Where("created_at >= ?", todayStart).
		Count(&stats.TodayCount).Error; err != nil {
		return nil, err
	}

	// This week count
	if err := r.db.Model(&models.Notification{}).Where("created_at >= ?", weekStart).
		Count(&stats.ThisWeekCount).Error; err != nil {
		return nil, err
	}

	// Count by type
	stats.ByType = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}

	err := r.db.Model(&models.Notification{}).
		Select("type, COUNT(*) as count").
		Group("type").Scan(&typeStats).Error

	if err != nil {
		return nil, err
	}

	for _, ts := range typeStats {
		stats.ByType[ts.Type] = ts.Count
	}

	// Most active users (top 10)
	var userStats []UserNotificationStats
	err = r.db.Model(&models.Notification{}).
		Select("user_id, COUNT(*) as count, SUM(CASE WHEN is_read = false THEN 1 ELSE 0 END) as unread").
		Group("user_id").
		Order("count DESC").
		Limit(10).
		Scan(&userStats).Error

	if err != nil {
		return nil, err
	}

	// Get user emails for the stats
	for i := range userStats {
		var user models.User
		if err := r.db.Select("email").First(&user, "id = ?", userStats[i].UserID).Error; err == nil {
			userStats[i].Email = user.Email
		}
	}

	stats.MostActiveUsers = userStats

	return &stats, nil
}

func (r *NotificationRepositoryImpl) CleanOldNotifications(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	return r.db.Where("created_at < ?", cutoffDate).Delete(&models.Notification{}).Error
}

// Factory methods for common notification types

func (r *NotificationRepositoryImpl) CreateNewResponseNotification(employerID, castingID, responseID, modelName string) error {
	data := map[string]interface{}{
		"casting_id":  castingID,
		"response_id": responseID,
		"model_name":  modelName,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notification := &models.Notification{
		UserID:  employerID,
		Type:    NotificationTypeNewResponse,
		Title:   "Новый отклик на кастинг",
		Message: fmt.Sprintf("Модель %s откликнулась на ваш кастинг", modelName),
		Data:    datatypes.JSON(jsonData),
	}

	return r.CreateNotification(notification)
}

func (r *NotificationRepositoryImpl) CreateResponseStatusNotification(modelID, castingTitle string, status models.ResponseStatus) error {
	var title, message string

	switch status {
	case models.ResponseStatusAccepted:
		title = "Отклик принят"
		message = fmt.Sprintf("Ваш отклик на кастинг '%s' был принят", castingTitle)
	case models.ResponseStatusRejected:
		title = "Отклик отклонен"
		message = fmt.Sprintf("Ваш отклик на кастинг '%s' был отклонен", castingTitle)
	default:
		return errors.New("unsupported status for notification")
	}

	notification := &models.Notification{
		UserID:  modelID,
		Type:    NotificationTypeResponseStatus,
		Title:   title,
		Message: message,
	}

	return r.CreateNotification(notification)
}

func (r *NotificationRepositoryImpl) CreateCastingMatchNotification(modelID string, castingTitle string, score float64) error {
	notification := &models.Notification{
		UserID:  modelID,
		Type:    NotificationTypeCastingMatch,
		Title:   "Новый подходящий кастинг",
		Message: fmt.Sprintf("Мы нашли для вас подходящий кастинг '%s' (совпадение: %.0f%%)", castingTitle, score),
	}

	return r.CreateNotification(notification)
}

func (r *NotificationRepositoryImpl) CreateNewMessageNotification(recipientID, senderName string, dialogID string) error {
	data := map[string]interface{}{
		"dialog_id": dialogID,
		"sender":    senderName,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	notification := &models.Notification{
		UserID:  recipientID,
		Type:    NotificationTypeNewMessage,
		Title:   "Новое сообщение",
		Message: fmt.Sprintf("У вас новое сообщение от %s", senderName),
		Data:    datatypes.JSON(jsonData),
	}

	return r.CreateNotification(notification)
}

func (r *NotificationRepositoryImpl) CreateSubscriptionExpiringNotification(userID, planName string, daysRemaining int) error {
	notification := &models.Notification{
		UserID:  userID,
		Type:    NotificationTypeSubscriptionExpiring,
		Title:   "Подписка скоро истекает",
		Message: fmt.Sprintf("Ваша подписка '%s' истекает через %d дней", planName, daysRemaining),
	}

	return r.CreateNotification(notification)
}

// Batch operations for performance

func (r *NotificationRepositoryImpl) CreateBulkResponseNotifications(employerID string, responses []ResponseNotificationData) error {
	var notifications []*models.Notification

	for _, response := range responses {
		data := map[string]interface{}{
			"casting_id":  response.CastingID,
			"response_id": response.ResponseID,
			"model_name":  response.ModelName,
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		notification := &models.Notification{
			UserID:  employerID,
			Type:    NotificationTypeNewResponse,
			Title:   "Новый отклик на кастинг",
			Message: fmt.Sprintf("Модель %s откликнулась на ваш кастинг", response.ModelName),
			Data:    datatypes.JSON(jsonData),
		}

		notifications = append(notifications, notification)
	}

	return r.CreateBulkNotifications(notifications)
}

func (r *NotificationRepositoryImpl) CreateBulkCastingMatchNotifications(matches []CastingMatchNotificationData) error {
	var notifications []*models.Notification

	for _, match := range matches {
		notification := &models.Notification{
			UserID:  match.ModelID,
			Type:    NotificationTypeCastingMatch,
			Title:   "Новый подходящий кастинг",
			Message: fmt.Sprintf("Мы нашли для вас подходящий кастинг '%s' (совпадение: %.0f%%)", match.CastingTitle, match.Score),
		}

		notifications = append(notifications, notification)
	}

	return r.CreateBulkNotifications(notifications)
}

// Helper methods

func (r *NotificationRepositoryImpl) validateNotification(notification *models.Notification) error {
	if notification.UserID == "" {
		return errors.New("user ID is required")
	}

	if notification.Type == "" {
		return errors.New("notification type is required")
	}

	if notification.Title == "" {
		return errors.New("notification title is required")
	}

	// Validate notification type
	validTypes := map[string]bool{
		NotificationTypeNewResponse:          true,
		NotificationTypeNewMessage:           true,
		NotificationTypeCastingMatch:         true,
		NotificationTypeResponseStatus:       true,
		NotificationTypeSubscriptionExpiring: true,
		NotificationTypeNewCasting:           true,
		NotificationTypeProfileView:          true,
	}

	if !validTypes[notification.Type] {
		return fmt.Errorf("invalid notification type: %s", notification.Type)
	}

	// Validate JSON data if present
	if len(notification.Data) > 0 {
		if !json.Valid(notification.Data) {
			return ErrInvalidNotificationData
		}
	}

	return nil
}

package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"mwork_backend/internal/models"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrNotificationNotFound    = errors.New("notification not found")
	ErrInvalidNotificationData = errors.New("invalid notification data")
	ErrTemplateNotFound        = errors.New("notification template not found")
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
	NotificationTypePasswordReset        = "password_reset"
	NotificationTypeAnnouncement         = "announcement"
)

type NotificationRepository interface {
	// Notification operations
	CreateNotification(db *gorm.DB, notification *models.Notification) error
	CreateBulkNotifications(db *gorm.DB, notifications []*models.Notification) error
	FindNotificationByID(db *gorm.DB, id string) (*models.Notification, error)
	FindUserNotifications(db *gorm.DB, userID string, criteria NotificationCriteria) ([]models.Notification, int64, error)
	MarkAsRead(db *gorm.DB, notificationID string) error
	MarkAllAsRead(db *gorm.DB, userID string) error
	MarkMultipleAsRead(db *gorm.DB, notificationIDs []string) error
	DeleteNotification(db *gorm.DB, id string) error
	DeleteUserNotifications(db *gorm.DB, userID string) error
	DeleteReadNotifications(db *gorm.DB, userID string, olderThan time.Time) error

	// Notification stats
	GetUserNotificationStats(db *gorm.DB, userID string) (*NotificationStats, error)
	GetUnreadCount(db *gorm.DB, userID string) (int64, error)

	// Template operations
	CreateNotificationTemplate(db *gorm.DB, template *NotificationTemplate) error
	FindTemplateByID(db *gorm.DB, templateID string) (*NotificationTemplate, error)
	FindAllTemplates(db *gorm.DB) ([]*NotificationTemplate, error)
	FindTemplateByType(db *gorm.DB, notificationType string) (*NotificationTemplate, error)
	UpdateTemplate(db *gorm.DB, template *NotificationTemplate) error
	DeleteTemplate(db *gorm.DB, templateID string) error

	// Admin operations
	FindAllNotifications(db *gorm.DB, criteria AdminNotificationCriteria) ([]models.Notification, int64, error)
	GetPlatformNotificationStats(db *gorm.DB) (*PlatformNotificationStats, error)
	CleanOldNotifications(db *gorm.DB, days int) error

	// Factory methods for common notification types
	CreateNewResponseNotification(db *gorm.DB, employerID, castingID, responseID, modelName string) error
	CreateResponseStatusNotification(db *gorm.DB, modelID, castingTitle string, status models.ResponseStatus) error
	CreateCastingMatchNotification(db *gorm.DB, modelID string, castingTitle string, score float64) error
	CreateNewMessageNotification(db *gorm.DB, recipientID, senderName string, dialogID string) error
	CreateSubscriptionExpiringNotification(db *gorm.DB, userID, planName string, daysRemaining int) error
	CreateBulkResponseNotifications(db *gorm.DB, employerID string, responses []ResponseNotificationData) error
	CreateBulkCastingMatchNotifications(db *gorm.DB, matches []CastingMatchNotificationData) error
}

type NotificationRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
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
	ID        string         `gorm:"primaryKey" json:"id"`
	Type      string         `gorm:"uniqueIndex" json:"type"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Variables datatypes.JSON `json:"variables" gorm:"type:jsonb"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
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

// ✅ Конструктор не принимает db
func NewNotificationRepository() NotificationRepository {
	return &NotificationRepositoryImpl{}
}

// Notification operations

func (r *NotificationRepositoryImpl) CreateNotification(db *gorm.DB, notification *models.Notification) error {
	// Validate notification data
	// ✅ Передаем db
	if err := r.validateNotification(db, notification); err != nil {
		return err
	}
	// ✅ Используем 'db' из параметра
	return db.Create(notification).Error
}

func (r *NotificationRepositoryImpl) CreateBulkNotifications(db *gorm.DB, notifications []*models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Validate all notifications
	for _, notification := range notifications {
		// ✅ Передаем db
		if err := r.validateNotification(db, notification); err != nil {
			return err
		}
	}
	// ✅ Используем 'db' из параметра
	return db.CreateInBatches(notifications, 100).Error
}

func (r *NotificationRepositoryImpl) FindNotificationByID(db *gorm.DB, id string) (*models.Notification, error) {
	var notification models.Notification
	// ✅ Используем 'db' из параметра
	err := db.First(&notification, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	return &notification, nil
}

func (r *NotificationRepositoryImpl) FindUserNotifications(db *gorm.DB, userID string, criteria NotificationCriteria) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	// ✅ Используем 'db' из параметра
	query := db.Where("user_id = ?", userID)

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
	// ✅ Используем 'db' (query)
	if err := query.Model(&models.Notification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	// ✅ Используем 'db' (query)
	err := query.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *NotificationRepositoryImpl) MarkAsRead(db *gorm.DB, notificationID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.Notification{}).Where("id = ?", notificationID).Updates(map[string]interface{}{
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

func (r *NotificationRepositoryImpl) MarkAllAsRead(db *gorm.DB, userID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	})

	return result.Error
}

func (r *NotificationRepositoryImpl) MarkMultipleAsRead(db *gorm.DB, notificationIDs []string) error {
	if len(notificationIDs) == 0 {
		return nil
	}
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.Notification{}).Where("id IN ?", notificationIDs).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	})

	return result.Error
}

func (r *NotificationRepositoryImpl) DeleteNotification(db *gorm.DB, id string) error {
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepositoryImpl) DeleteUserNotifications(db *gorm.DB, userID string) error {
	// ✅ Используем 'db' из параметра
	return db.Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}

func (r *NotificationRepositoryImpl) DeleteReadNotifications(db *gorm.DB, userID string, olderThan time.Time) error {
	// ✅ Используем 'db' из параметра
	return db.Where("user_id = ? AND is_read = ? AND created_at < ?", userID, true, olderThan).
		Delete(&models.Notification{}).Error
}

// Notification stats

func (r *NotificationRepositoryImpl) GetUserNotificationStats(db *gorm.DB, userID string) (*NotificationStats, error) {
	var stats NotificationStats
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))

	// Total notifications
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("user_id = ?", userID).
		Count(&stats.TotalNotifications).Error; err != nil {
		return nil, err
	}

	// Unread count
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).
		Count(&stats.UnreadCount).Error; err != nil {
		return nil, err
	}

	// Read count
	stats.ReadCount = stats.TotalNotifications - stats.UnreadCount

	// Today count
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND created_at >= ?", userID, todayStart).
		Count(&stats.TodayCount).Error; err != nil {
		return nil, err
	}

	// This week count
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND created_at >= ?", userID, weekStart).
		Count(&stats.ThisWeekCount).Error; err != nil {
		return nil, err
	}

	// Count by type
	stats.ByType = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Notification{}).Where("user_id = ?", userID).
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

func (r *NotificationRepositoryImpl) GetUnreadCount(db *gorm.DB, userID string) (int64, error) {
	var count int64
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// Template operations

func (r *NotificationRepositoryImpl) CreateNotificationTemplate(db *gorm.DB, template *NotificationTemplate) error {
	// ✅ Используем 'db' из параметра
	return db.Create(template).Error
}

func (r *NotificationRepositoryImpl) FindTemplateByID(db *gorm.DB, templateID string) (*NotificationTemplate, error) {
	var template NotificationTemplate
	err := db.First(&template, "id = ?", templateID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return &template, nil
}

func (r *NotificationRepositoryImpl) FindAllTemplates(db *gorm.DB) ([]*NotificationTemplate, error) {
	var templates []*NotificationTemplate
	// GORM сам найдет таблицу "notification_templates" в схеме "public"
	err := db.Find(&templates).Error
	return templates, err
}

func (r *NotificationRepositoryImpl) FindTemplateByType(db *gorm.DB, notificationType string) (*NotificationTemplate, error) {
	var template NotificationTemplate
	// ✅ Используем 'db' из параметра
	err := db.Where("type = ? AND is_active = ?", notificationType, true).First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}
	return &template, nil
}

func (r *NotificationRepositoryImpl) UpdateTemplate(db *gorm.DB, template *NotificationTemplate) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(template).Updates(map[string]interface{}{
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

func (r *NotificationRepositoryImpl) FindAllNotifications(db *gorm.DB, criteria AdminNotificationCriteria) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	// ✅ Используем 'db' из параметра
	query := db.Model(&models.Notification{})

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
	// ✅ Используем 'db' (query)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	limit := criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	// ✅ Используем 'db' (query)
	err := query.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *NotificationRepositoryImpl) GetPlatformNotificationStats(db *gorm.DB) (*PlatformNotificationStats, error) {
	var stats PlatformNotificationStats
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))

	// Total notifications
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Count(&stats.TotalNotifications).Error; err != nil {
		return nil, err
	}

	// Unread count
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("is_read = ?", false).
		Count(&stats.UnreadCount).Error; err != nil {
		return nil, err
	}

	// Today count
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("created_at >= ?", todayStart).
		Count(&stats.TodayCount).Error; err != nil {
		return nil, err
	}

	// This week count
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.Notification{}).Where("created_at >= ?", weekStart).
		Count(&stats.ThisWeekCount).Error; err != nil {
		return nil, err
	}

	// Count by type
	stats.ByType = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}
	// ✅ Используем 'db' из параметра
	err := db.Model(&models.Notification{}).
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
	// ✅ Используем 'db' из параметра
	err = db.Model(&models.Notification{}).
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
		// ✅ Используем 'db' из параметра
		if err := db.Select("email").First(&user, "id = ?", userStats[i].UserID).Error; err == nil {
			userStats[i].Email = user.Email
		}
	}

	stats.MostActiveUsers = userStats

	return &stats, nil
}

func (r *NotificationRepositoryImpl) CleanOldNotifications(db *gorm.DB, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	// ✅ Используем 'db' из параметра
	return db.Where("created_at < ?", cutoffDate).Delete(&models.Notification{}).Error
}

// Factory methods for common notification types

func (r *NotificationRepositoryImpl) CreateNewResponseNotification(db *gorm.DB, employerID, castingID, responseID, modelName string) error {
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
	// ✅ Передаем db
	return r.CreateNotification(db, notification)
}

func (r *NotificationRepositoryImpl) CreateResponseStatusNotification(db *gorm.DB, modelID, castingTitle string, status models.ResponseStatus) error {
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
	// ✅ Передаем db
	return r.CreateNotification(db, notification)
}

func (r *NotificationRepositoryImpl) CreateCastingMatchNotification(db *gorm.DB, modelID string, castingTitle string, score float64) error {
	notification := &models.Notification{
		UserID:  modelID,
		Type:    NotificationTypeCastingMatch,
		Title:   "Новый подходящий кастинг",
		Message: fmt.Sprintf("Мы нашли для вас подходящий кастинг '%s' (совпадение: %.0f%%)", castingTitle, score),
	}
	// ✅ Передаем db
	return r.CreateNotification(db, notification)
}

func (r *NotificationRepositoryImpl) CreateNewMessageNotification(db *gorm.DB, recipientID, senderName string, dialogID string) error {
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
	// ✅ Передаем db
	return r.CreateNotification(db, notification)
}

func (r *NotificationRepositoryImpl) CreateSubscriptionExpiringNotification(db *gorm.DB, userID, planName string, daysRemaining int) error {
	notification := &models.Notification{
		UserID:  userID,
		Type:    NotificationTypeSubscriptionExpiring,
		Title:   "Подписка скоро истекает",
		Message: fmt.Sprintf("Ваша подписка '%s' истекает через %d дней", planName, daysRemaining),
	}
	// ✅ Передаем db
	return r.CreateNotification(db, notification)
}

// Batch operations for performance

func (r *NotificationRepositoryImpl) CreateBulkResponseNotifications(db *gorm.DB, employerID string, responses []ResponseNotificationData) error {
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
	// ✅ Передаем db
	return r.CreateBulkNotifications(db, notifications)
}

func (r *NotificationRepositoryImpl) CreateBulkCastingMatchNotifications(db *gorm.DB, matches []CastingMatchNotificationData) error {
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
	// ✅ Передаем db
	return r.CreateBulkNotifications(db, notifications)
}

// Helper methods

// ✅ Метод теперь принимает 'db' (хотя и не использует), для согласованности интерфейса
func (r *NotificationRepositoryImpl) validateNotification(db *gorm.DB, notification *models.Notification) error {
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
		NotificationTypePasswordReset:        true,
		NotificationTypeAnnouncement:         true,
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

func (r *NotificationRepositoryImpl) DeleteTemplate(db *gorm.DB, templateID string) error {
	result := db.Where("id = ?", templateID).Delete(&NotificationTemplate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

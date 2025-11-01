package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type NotificationService interface {
	// Notification operations
	CreateNotification(db *gorm.DB, userID string, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error)
	CreateBulkNotifications(db *gorm.DB, req *dto.CreateBulkNotificationsRequest) error
	GetNotification(db *gorm.DB, notificationID string) (*dto.NotificationResponse, error)
	GetUserNotifications(db *gorm.DB, userID string, criteria dto.NotificationCriteria) (*dto.NotificationListResponse, error)
	MarkAsRead(db *gorm.DB, userID, notificationID string) error
	MarkAllAsRead(db *gorm.DB, userID string) error
	MarkMultipleAsRead(db *gorm.DB, userID string, notificationIDs []string) error
	DeleteNotification(db *gorm.DB, userID, notificationID string) error
	DeleteUserNotifications(db *gorm.DB, userID string) error
	CleanOldNotifications(db *gorm.DB, days int) error

	// Notification stats
	GetUserNotificationStats(db *gorm.DB, userID string) (*repositories.NotificationStats, error)
	GetUnreadCount(db *gorm.DB, userID string) (int64, error)

	// Template operations
	CreateTemplate(db *gorm.DB, adminID string, req *dto.CreateTemplateRequest) error
	GetTemplate(db *gorm.DB, templateID string) (*repositories.NotificationTemplate, error)
	GetTemplateByType(db *gorm.DB, notificationType string) (*repositories.NotificationTemplate, error)
	UpdateTemplate(db *gorm.DB, adminID, templateID string, req *dto.UpdateTemplateRequest) error
	DeleteTemplate(db *gorm.DB, adminID, templateID string) error
	GetAllTemplates(db *gorm.DB) ([]*repositories.NotificationTemplate, error)

	// Factory methods for common notification types
	NotifyNewResponse(db *gorm.DB, employerID, castingID, responseID, modelName string) error
	NotifyResponseStatus(db *gorm.DB, modelID, castingTitle string, status models.ResponseStatus) error
	NotifyCastingMatch(db *gorm.DB, modelID string, castingTitle string, score float64) error
	NotifyNewMessage(db *gorm.DB, recipientID, senderName, dialogID string) error
	NotifySubscriptionExpiring(db *gorm.DB, userID, planName string, daysRemaining int) error
	NotifyNewCasting(db *gorm.DB, modelID string, castingTitle string) error
	NotifyProfileView(db *gorm.DB, modelID string, viewerName string) error

	// Batch operations
	NotifyBulkResponses(db *gorm.DB, employerID string, responses []dto.ResponseNotificationData) error
	NotifyBulkCastingMatches(db *gorm.DB, matches []dto.CastingMatchNotificationData) error

	// Admin operations
	GetAllNotifications(db *gorm.DB, criteria dto.AdminNotificationCriteria) (*dto.NotificationListResponse, error)
	GetPlatformNotificationStats(db *gorm.DB) (*repositories.PlatformNotificationStats, error)
	SendBulkNotification(db *gorm.DB, adminID string, req *dto.SendBulkNotificationRequest) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type notificationService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	notificationRepo repositories.NotificationRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
}

// ✅ Конструктор обновлен (db убран)
func NewNotificationService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	notificationRepo repositories.NotificationRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
) NotificationService {
	return &notificationService{
		// ❌ 'db: db,' УДАЛЕНО
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
	}
}

// ---------------- Notification operations ----------------

// CreateNotification - 'db' добавлен
func (s *notificationService) CreateNotification(db *gorm.DB, userID string, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if _, err := s.userRepo.FindByID(tx, req.UserID); err != nil {
		return nil, errors.New("user not found")
	}

	var dataJSON datatypes.JSON
	if req.Data != nil {
		jsonData, err := json.Marshal(req.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal notification data: %w", err)
		}
		dataJSON = datatypes.JSON(jsonData)
	}

	notification := &models.Notification{
		UserID:  req.UserID,
		Type:    req.Type,
		Title:   req.Title,
		Message: req.Message,
		Data:    dataJSON,
		IsRead:  false,
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateNotification(tx, notification); err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Коммитим транзакцию
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	return s.buildNotificationResponse(notification), nil
}

// CreateBulkNotifications - 'db' добавлен
func (s *notificationService) CreateBulkNotifications(db *gorm.DB, req *dto.CreateBulkNotificationsRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	var notifications []*models.Notification

	for _, notificationReq := range req.Notifications {
		// ✅ Передаем tx
		if _, err := s.userRepo.FindByID(tx, notificationReq.UserID); err != nil {
			return fmt.Errorf("user not found: %s", notificationReq.UserID)
		}

		var dataJSON datatypes.JSON
		if notificationReq.Data != nil {
			jsonData, err := json.Marshal(notificationReq.Data)
			if err != nil {
				return fmt.Errorf("failed to marshal notification data: %w", err)
			}
			dataJSON = datatypes.JSON(jsonData)
		}

		notifications = append(notifications, &models.Notification{
			UserID:  notificationReq.UserID,
			Type:    notificationReq.Type,
			Title:   notificationReq.Title,
			Message: notificationReq.Message,
			Data:    dataJSON,
			IsRead:  false,
		})
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateBulkNotifications(tx, notifications); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetNotification - 'db' добавлен
func (s *notificationService) GetNotification(db *gorm.DB, notificationID string) (*dto.NotificationResponse, error) {
	// ✅ Используем 'db' из параметра
	notification, err := s.notificationRepo.FindNotificationByID(db, notificationID)
	if err != nil {
		return nil, handleNotificationError(err)
	}
	return s.buildNotificationResponse(notification), nil
}

// GetUserNotifications - 'db' добавлен
func (s *notificationService) GetUserNotifications(db *gorm.DB, userID string, criteria dto.NotificationCriteria) (*dto.NotificationListResponse, error) {

	// 1. Инициализируем критерии репозитория только общими полями
	repoCriteria := repositories.NotificationCriteria{
		Page:     criteria.Page,
		PageSize: criteria.PageSize,
	}

	// 2. Безопасно извлекаем значения из карты Filters
	if criteria.Filters != nil {
		if val, ok := criteria.Filters["unread_only"]; ok {
			if boolVal, ok := val.(bool); ok {
				repoCriteria.UnreadOnly = boolVal
			}
		}
		if val, ok := criteria.Filters["type"]; ok {
			if strVal, ok := val.(string); ok {
				repoCriteria.Type = strVal
			}
		}
		if val, ok := criteria.Filters["date_from"]; ok {
			if timeVal, ok := val.(time.Time); ok {
				repoCriteria.DateFrom = timeVal
			}
		}
		if val, ok := criteria.Filters["date_to"]; ok {
			if timeVal, ok := val.(time.Time); ok {
				repoCriteria.DateTo = timeVal
			}
		}
	}

	// ✅ Используем 'db' из параметра и обновленные repoCriteria
	notifications, total, err := s.notificationRepo.FindUserNotifications(db, userID, repoCriteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var notificationResponses []*dto.NotificationResponse
	for _, notification := range notifications {
		notificationResponses = append(notificationResponses, s.buildNotificationResponse(&notification))
	}

	return &dto.NotificationListResponse{
		Notifications: notificationResponses,
		Total:         total,
		Page:          criteria.Page,
		PageSize:      criteria.PageSize,
		TotalPages:    calculateTotalPages(total, criteria.PageSize),
	}, nil
}

// MarkAsRead - 'db' добавлен
func (s *notificationService) MarkAsRead(db *gorm.DB, userID, notificationID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	notification, err := s.notificationRepo.FindNotificationByID(tx, notificationID)
	if err != nil {
		return handleNotificationError(err)
	}
	if notification.UserID != userID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.MarkAsRead(tx, notificationID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// MarkAllAsRead - 'db' добавлен
func (s *notificationService) MarkAllAsRead(db *gorm.DB, userID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.MarkAllAsRead(tx, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// MarkMultipleAsRead - 'db' добавлен
func (s *notificationService) MarkMultipleAsRead(db *gorm.DB, userID string, notificationIDs []string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	for _, notificationID := range notificationIDs {
		// ✅ Передаем tx
		notification, err := s.notificationRepo.FindNotificationByID(tx, notificationID)
		if err != nil {
			return handleNotificationError(err)
		}
		if notification.UserID != userID {
			return errors.New("access denied for some notifications")
		}
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.MarkMultipleAsRead(tx, notificationIDs); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteNotification - 'db' добавлен
func (s *notificationService) DeleteNotification(db *gorm.DB, userID, notificationID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	notification, err := s.notificationRepo.FindNotificationByID(tx, notificationID)
	if err != nil {
		return handleNotificationError(err)
	}
	if notification.UserID != userID {
		return errors.New("access denied")
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.DeleteNotification(tx, notificationID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteUserNotifications - 'db' добавлен
func (s *notificationService) DeleteUserNotifications(db *gorm.DB, userID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.DeleteUserNotifications(tx, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// CleanOldNotifications - 'db' добавлен
func (s *notificationService) CleanOldNotifications(db *gorm.DB, days int) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CleanOldNotifications(tx, days); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ---------------- Notification stats ----------------

// GetUserNotificationStats - 'db' добавлен
func (s *notificationService) GetUserNotificationStats(db *gorm.DB, userID string) (*repositories.NotificationStats, error) {
	// ✅ Используем 'db' из параметра
	return s.notificationRepo.GetUserNotificationStats(db, userID)
}

// GetUnreadCount - 'db' добавлен
func (s *notificationService) GetUnreadCount(db *gorm.DB, userID string) (int64, error) {
	// ✅ Используем 'db' из параметра
	return s.notificationRepo.GetUnreadCount(db, userID)
}

// ---------------- Template operations ----------------

// CreateTemplate - 'db' добавлен
func (s *notificationService) CreateTemplate(db *gorm.DB, adminID string, req *dto.CreateTemplateRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleNotificationError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}
	if !isValidNotificationType(req.Type) {
		return errors.New("invalid notification type")
	}

	template := &repositories.NotificationTemplate{
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Variables: req.Variables,
		IsActive:  req.IsActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateNotificationTemplate(tx, template); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetTemplate - 'db' добавлен
func (s *notificationService) GetTemplate(db *gorm.DB, templateID string) (*repositories.NotificationTemplate, error) {
	// ✅ Используем 'db' из параметра
	template, err := s.notificationRepo.FindTemplateByID(db, templateID)
	if err != nil {
		return nil, handleNotificationError(err)
	}
	return template, nil
}

// GetTemplateByType - 'db' добавлен
func (s *notificationService) GetTemplateByType(db *gorm.DB, notificationType string) (*repositories.NotificationTemplate, error) {
	// ✅ Используем 'db' из параметра
	template, err := s.notificationRepo.FindTemplateByType(db, notificationType)
	if err != nil {
		return nil, handleNotificationError(err)
	}
	return template, nil
}

// UpdateTemplate - 'db' добавлен
func (s *notificationService) UpdateTemplate(db *gorm.DB, adminID, templateID string, req *dto.UpdateTemplateRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleNotificationError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}

	// ✅ Передаем tx
	template, err := s.notificationRepo.FindTemplateByID(tx, templateID)
	if err != nil {
		return handleNotificationError(err)
	}

	if req.Title != nil {
		template.Title = *req.Title
	}
	if req.Message != nil {
		template.Message = *req.Message
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}
	template.UpdatedAt = time.Now()

	// ✅ Передаем tx
	if err := s.notificationRepo.UpdateTemplate(tx, template); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// DeleteTemplate - 'db' добавлен
func (s *notificationService) DeleteTemplate(db *gorm.DB, adminID, templateID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleNotificationError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.DeleteTemplate(tx, templateID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// GetAllTemplates - 'db' добавлен
func (s *notificationService) GetAllTemplates(db *gorm.DB) ([]*repositories.NotificationTemplate, error) {
	// ✅ Используем 'db' из параметра
	templates, err := s.notificationRepo.FindAllTemplates(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return templates, nil
}

// ---------------- Factory methods ----------------

// NotifyNewResponse - 'db' добавлен
func (s *notificationService) NotifyNewResponse(db *gorm.DB, employerID, castingID, responseID, modelName string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateNewResponseNotification(tx, employerID, castingID, responseID, modelName); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifyResponseStatus - 'db' добавлен
func (s *notificationService) NotifyResponseStatus(db *gorm.DB, modelID, castingTitle string, status models.ResponseStatus) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateResponseStatusNotification(tx, modelID, castingTitle, status); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifyCastingMatch - 'db' добавлен
func (s *notificationService) NotifyCastingMatch(db *gorm.DB, modelID string, castingTitle string, score float64) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateCastingMatchNotification(tx, modelID, castingTitle, score); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifyNewMessage - 'db' добавлен
func (s *notificationService) NotifyNewMessage(db *gorm.DB, recipientID, senderName, dialogID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateNewMessageNotification(tx, recipientID, senderName, dialogID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifySubscriptionExpiring - 'db' добавлен
func (s *notificationService) NotifySubscriptionExpiring(db *gorm.DB, userID, planName string, daysRemaining int) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateSubscriptionExpiringNotification(tx, userID, planName, daysRemaining); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifyNewCasting - 'db' добавлен
func (s *notificationService) NotifyNewCasting(db *gorm.DB, modelID string, castingTitle string) error {
	notification := &models.Notification{
		UserID:  modelID,
		Type:    repositories.NotificationTypeNewCasting,
		Title:   "Новый кастинг в вашем городе",
		Message: fmt.Sprintf("Появился новый кастинг '%s', который может вас заинтересовать", castingTitle),
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateNotification(tx, notification); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifyProfileView - 'db' добавлен
func (s *notificationService) NotifyProfileView(db *gorm.DB, modelID string, viewerName string) error {
	notification := &models.Notification{
		UserID:  modelID,
		Type:    repositories.NotificationTypeProfileView,
		Title:   "Ваш профиль просмотрели",
		Message: fmt.Sprintf("Ваш профиль просмотрел(а) %s", viewerName),
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateNotification(tx, notification); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ---------------- Batch operations ----------------

// NotifyBulkResponses - 'db' добавлен
func (s *notificationService) NotifyBulkResponses(db *gorm.DB, employerID string, responses []dto.ResponseNotificationData) error {
	repoResponses := make([]repositories.ResponseNotificationData, len(responses))
	for i, r := range responses {
		repoResponses[i] = repositories.ResponseNotificationData{
			CastingID:  r.CastingID,
			ResponseID: r.ResponseID,
			ModelName:  r.ModelName,
		}
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateBulkResponseNotifications(tx, employerID, repoResponses); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// NotifyBulkCastingMatches - 'db' добавлен
func (s *notificationService) NotifyBulkCastingMatches(db *gorm.DB, matches []dto.CastingMatchNotificationData) error {
	repoMatches := make([]repositories.CastingMatchNotificationData, len(matches))
	for i, m := range matches {
		repoMatches[i] = repositories.CastingMatchNotificationData{
			ModelID:      m.ModelID,
			CastingTitle: m.CastingTitle,
			Score:        m.Score,
		}
	}

	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateBulkCastingMatchNotifications(tx, repoMatches); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ---------------- Admin operations ----------------

// GetAllNotifications - 'db' добавлен
func (s *notificationService) GetAllNotifications(db *gorm.DB, criteria dto.AdminNotificationCriteria) (*dto.NotificationListResponse, error) {

	// 1. Инициализируем критерии репозитория только общими полями
	repoCriteria := repositories.AdminNotificationCriteria{
		Page:     criteria.Page,
		PageSize: criteria.PageSize,
	}

	// 2. Безопасно извлекаем значения из карты Filters
	if criteria.Filters != nil {
		if val, ok := criteria.Filters["user_id"]; ok {
			if strVal, ok := val.(string); ok {
				repoCriteria.UserID = strVal
			}
		}
		if val, ok := criteria.Filters["type"]; ok {
			if strVal, ok := val.(string); ok {
				repoCriteria.Type = strVal
			}
		}
		if val, ok := criteria.Filters["unread_only"]; ok {
			if boolVal, ok := val.(bool); ok {
				repoCriteria.UnreadOnly = boolVal
			}
		}
		if val, ok := criteria.Filters["date_from"]; ok {
			if timeVal, ok := val.(time.Time); ok {
				repoCriteria.DateFrom = timeVal
			}
		}
		if val, ok := criteria.Filters["date_to"]; ok {
			if timeVal, ok := val.(time.Time); ok {
				repoCriteria.DateTo = timeVal
			}
		}
	}

	// ✅ Используем 'db' из параметра и обновленные repoCriteria
	notifications, total, err := s.notificationRepo.FindAllNotifications(db, repoCriteria)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	var notificationResponses []*dto.NotificationResponse
	for _, notification := range notifications {
		notificationResponses = append(notificationResponses, s.buildNotificationResponse(&notification))
	}

	return &dto.NotificationListResponse{
		Notifications: notificationResponses,
		Total:         total,
		Page:          criteria.Page,
		PageSize:      criteria.PageSize,
		TotalPages:    calculateTotalPages(total, criteria.PageSize),
	}, nil
}

// GetPlatformNotificationStats - 'db' добавлен
func (s *notificationService) GetPlatformNotificationStats(db *gorm.DB) (*repositories.PlatformNotificationStats, error) {
	// ✅ Используем 'db' из параметра
	return s.notificationRepo.GetPlatformNotificationStats(db)
}

// SendBulkNotification - 'db' добавлен
func (s *notificationService) SendBulkNotification(db *gorm.DB, adminID string, req *dto.SendBulkNotificationRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleNotificationError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}
	if !isValidNotificationType(req.Type) {
		return errors.New("invalid notification type")
	}

	var notifications []*models.Notification
	for _, userID := range req.UserIDs {
		// ✅ Передаем tx
		if _, err := s.userRepo.FindByID(tx, userID); err != nil {
			return fmt.Errorf("user not found: %s", userID)
		}

		var dataJSON datatypes.JSON
		if req.Data != nil {
			jsonData, err := json.Marshal(req.Data)
			if err != nil {
				return fmt.Errorf("failed to marshal notification data: %w", err)
			}
			dataJSON = datatypes.JSON(jsonData)
		}

		notifications = append(notifications, &models.Notification{
			UserID:  userID,
			Type:    req.Type,
			Title:   req.Title,
			Message: req.Message,
			Data:    dataJSON,
			IsRead:  false,
		})
	}

	// ✅ Передаем tx
	if err := s.notificationRepo.CreateBulkNotifications(tx, notifications); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// ---------------- Helpers ----------------

// (buildNotificationResponse - чистая функция, без изменений)
func (s *notificationService) buildNotificationResponse(notification *models.Notification) *dto.NotificationResponse {
	response := &dto.NotificationResponse{
		ID:        notification.ID,
		UserID:    notification.UserID,
		Type:      notification.Type,
		Title:     notification.Title,
		Message:   notification.Message,
		IsRead:    notification.IsRead,
		ReadAt:    notification.ReadAt,
		CreatedAt: notification.CreatedAt,
		UpdatedAt: notification.UpdatedAt,
	}

	if len(notification.Data) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(notification.Data, &data); err == nil {
			response.Data = data
		}
	}

	return response
}

// (isValidNotificationType - чистая функция, без изменений)
func isValidNotificationType(notificationType string) bool {
	validTypes := map[string]bool{
		repositories.NotificationTypeNewResponse:          true,
		repositories.NotificationTypeNewMessage:           true,
		repositories.NotificationTypeCastingMatch:         true,
		repositories.NotificationTypeResponseStatus:       true,
		repositories.NotificationTypeSubscriptionExpiring: true,
		repositories.NotificationTypeNewCasting:           true,
		repositories.NotificationTypeProfileView:          true,
	}
	return validTypes[notificationType]
}

// (calculateTotalPages - чистая функция, добавлена)
func calculateTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}
	return totalPages
}

// (handleNotificationError - хелпер, без изменений)
func handleNotificationError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrNotificationNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

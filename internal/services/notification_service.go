package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/datatypes"
	"time"

	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services/dto"
)

type NotificationService interface {
	// Notification operations
	CreateNotification(userID string, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error)
	CreateBulkNotifications(req *dto.CreateBulkNotificationsRequest) error
	GetNotification(notificationID string) (*dto.NotificationResponse, error)
	GetUserNotifications(userID string, criteria dto.NotificationCriteria) (*dto.NotificationListResponse, error)
	MarkAsRead(userID, notificationID string) error
	MarkAllAsRead(userID string) error
	MarkMultipleAsRead(userID string, notificationIDs []string) error
	DeleteNotification(userID, notificationID string) error
	DeleteUserNotifications(userID string) error
	CleanOldNotifications(days int) error

	// Notification stats
	GetUserNotificationStats(userID string) (*repositories.NotificationStats, error)
	GetUnreadCount(userID string) (int64, error)

	// Template operations
	CreateTemplate(adminID string, req *dto.CreateTemplateRequest) error
	GetTemplate(templateID string) (*repositories.NotificationTemplate, error)
	GetTemplateByType(notificationType string) (*repositories.NotificationTemplate, error)
	UpdateTemplate(adminID, templateID string, req *dto.UpdateTemplateRequest) error
	DeleteTemplate(adminID, templateID string) error
	GetAllTemplates() ([]*repositories.NotificationTemplate, error)

	// Factory methods for common notification types
	NotifyNewResponse(employerID, castingID, responseID, modelName string) error
	NotifyResponseStatus(modelID, castingTitle string, status models.ResponseStatus) error
	NotifyCastingMatch(modelID string, castingTitle string, score float64) error
	NotifyNewMessage(recipientID, senderName, dialogID string) error
	NotifySubscriptionExpiring(userID, planName string, daysRemaining int) error
	NotifyNewCasting(modelID string, castingTitle string) error
	NotifyProfileView(modelID string, viewerName string) error

	// Batch operations
	NotifyBulkResponses(employerID string, responses []dto.ResponseNotificationData) error
	NotifyBulkCastingMatches(matches []dto.CastingMatchNotificationData) error

	// Admin operations
	GetAllNotifications(criteria dto.AdminNotificationCriteria) (*dto.NotificationListResponse, error)
	GetPlatformNotificationStats() (*repositories.PlatformNotificationStats, error)
	SendBulkNotification(adminID string, req *dto.SendBulkNotificationRequest) error
}

type notificationService struct {
	notificationRepo repositories.NotificationRepository
	userRepo         repositories.UserRepository
	profileRepo      repositories.ProfileRepository
}

func NewNotificationService(
	notificationRepo repositories.NotificationRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
	}
}

// ---------------- Notification operations ----------------

func (s *notificationService) CreateNotification(userID string, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
	if _, err := s.userRepo.FindByID(req.UserID); err != nil {
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

	if err := s.notificationRepo.CreateNotification(notification); err != nil {
		return nil, err
	}

	return s.buildNotificationResponse(notification), nil
}

func (s *notificationService) CreateBulkNotifications(req *dto.CreateBulkNotificationsRequest) error {
	var notifications []*models.Notification

	for _, notificationReq := range req.Notifications {
		if _, err := s.userRepo.FindByID(notificationReq.UserID); err != nil {
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

	return s.notificationRepo.CreateBulkNotifications(notifications)
}

func (s *notificationService) GetNotification(notificationID string) (*dto.NotificationResponse, error) {
	notification, err := s.notificationRepo.FindNotificationByID(notificationID)
	if err != nil {
		return nil, err
	}
	return s.buildNotificationResponse(notification), nil
}

func (s *notificationService) GetUserNotifications(userID string, criteria dto.NotificationCriteria) (*dto.NotificationListResponse, error) {
	// конвертируем DTO → репозиторий
	repoCriteria := repositories.NotificationCriteria{
		Page:     criteria.Page,
		PageSize: criteria.PageSize,
	}

	notifications, total, err := s.notificationRepo.FindUserNotifications(userID, repoCriteria)
	if err != nil {
		return nil, err
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

func (s *notificationService) MarkAsRead(userID, notificationID string) error {
	notification, err := s.notificationRepo.FindNotificationByID(notificationID)
	if err != nil {
		return err
	}
	if notification.UserID != userID {
		return errors.New("access denied")
	}
	return s.notificationRepo.MarkAsRead(notificationID)
}

func (s *notificationService) MarkAllAsRead(userID string) error {
	return s.notificationRepo.MarkAllAsRead(userID)
}

func (s *notificationService) MarkMultipleAsRead(userID string, notificationIDs []string) error {
	for _, notificationID := range notificationIDs {
		notification, err := s.notificationRepo.FindNotificationByID(notificationID)
		if err != nil {
			return err
		}
		if notification.UserID != userID {
			return errors.New("access denied for some notifications")
		}
	}
	return s.notificationRepo.MarkMultipleAsRead(notificationIDs)
}

func (s *notificationService) DeleteNotification(userID, notificationID string) error {
	notification, err := s.notificationRepo.FindNotificationByID(notificationID)
	if err != nil {
		return err
	}
	if notification.UserID != userID {
		return errors.New("access denied")
	}
	return s.notificationRepo.DeleteNotification(notificationID)
}

func (s *notificationService) DeleteUserNotifications(userID string) error {
	return s.notificationRepo.DeleteUserNotifications(userID)
}

func (s *notificationService) CleanOldNotifications(days int) error {
	return s.notificationRepo.CleanOldNotifications(days)
}

// ---------------- Notification stats ----------------

func (s *notificationService) GetUserNotificationStats(userID string) (*repositories.NotificationStats, error) {
	return s.notificationRepo.GetUserNotificationStats(userID)
}

func (s *notificationService) GetUnreadCount(userID string) (int64, error) {
	return s.notificationRepo.GetUnreadCount(userID)
}

// ---------------- Template operations ----------------

func (s *notificationService) CreateTemplate(adminID string, req *dto.CreateTemplateRequest) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
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

	return s.notificationRepo.CreateNotificationTemplate(template)
}

func (s *notificationService) GetTemplate(templateID string) (*repositories.NotificationTemplate, error) {
	return nil, errors.New("not implemented")
}

func (s *notificationService) GetTemplateByType(notificationType string) (*repositories.NotificationTemplate, error) {
	return s.notificationRepo.FindTemplateByType(notificationType)
}

func (s *notificationService) UpdateTemplate(adminID, templateID string, req *dto.UpdateTemplateRequest) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}

	template, err := s.GetTemplate(templateID)
	if err != nil {
		return err
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

	return s.notificationRepo.UpdateTemplate(template)
}

func (s *notificationService) DeleteTemplate(adminID, templateID string) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}
	return errors.New("not implemented")
}

func (s *notificationService) GetAllTemplates() ([]*repositories.NotificationTemplate, error) {
	return []*repositories.NotificationTemplate{}, nil
}

// ---------------- Factory methods ----------------

func (s *notificationService) NotifyNewResponse(employerID, castingID, responseID, modelName string) error {
	return s.notificationRepo.CreateNewResponseNotification(employerID, castingID, responseID, modelName)
}

func (s *notificationService) NotifyResponseStatus(modelID, castingTitle string, status models.ResponseStatus) error {
	return s.notificationRepo.CreateResponseStatusNotification(modelID, castingTitle, status)
}

func (s *notificationService) NotifyCastingMatch(modelID string, castingTitle string, score float64) error {
	return s.notificationRepo.CreateCastingMatchNotification(modelID, castingTitle, score)
}

func (s *notificationService) NotifyNewMessage(recipientID, senderName, dialogID string) error {
	return s.notificationRepo.CreateNewMessageNotification(recipientID, senderName, dialogID)
}

func (s *notificationService) NotifySubscriptionExpiring(userID, planName string, daysRemaining int) error {
	return s.notificationRepo.CreateSubscriptionExpiringNotification(userID, planName, daysRemaining)
}

func (s *notificationService) NotifyNewCasting(modelID string, castingTitle string) error {
	notification := &models.Notification{
		UserID:  modelID,
		Type:    repositories.NotificationTypeNewCasting,
		Title:   "Новый кастинг в вашем городе",
		Message: fmt.Sprintf("Появился новый кастинг '%s', который может вас заинтересовать", castingTitle),
	}
	return s.notificationRepo.CreateNotification(notification)
}

func (s *notificationService) NotifyProfileView(modelID string, viewerName string) error {
	notification := &models.Notification{
		UserID:  modelID,
		Type:    repositories.NotificationTypeProfileView,
		Title:   "Ваш профиль просмотрели",
		Message: fmt.Sprintf("Ваш профиль просмотрел(а) %s", viewerName),
	}
	return s.notificationRepo.CreateNotification(notification)
}

// ---------------- Batch operations ----------------

func (s *notificationService) NotifyBulkResponses(employerID string, responses []dto.ResponseNotificationData) error {
	repoResponses := make([]repositories.ResponseNotificationData, len(responses))
	for i, r := range responses {
		repoResponses[i] = repositories.ResponseNotificationData{
			CastingID:  r.CastingID, // убедись, что поле существует в репозитории
			ResponseID: r.ResponseID,
			ModelName:  r.ModelName, // может быть ModelID вместо ModelName
		}
	}
	return s.notificationRepo.CreateBulkResponseNotifications(employerID, repoResponses)
}

func (s *notificationService) NotifyBulkCastingMatches(matches []dto.CastingMatchNotificationData) error {
	repoMatches := make([]repositories.CastingMatchNotificationData, len(matches))
	for i, m := range matches {
		repoMatches[i] = repositories.CastingMatchNotificationData{
			ModelID: m.ModelID,
			Score:   m.Score,
		}
	}
	return s.notificationRepo.CreateBulkCastingMatchNotifications(repoMatches)
}

// ---------------- Admin operations ----------------

func (s *notificationService) GetAllNotifications(criteria dto.AdminNotificationCriteria) (*dto.NotificationListResponse, error) {
	repoCriteria := repositories.AdminNotificationCriteria{
		Page:     criteria.Page,
		PageSize: criteria.PageSize,
	}

	notifications, total, err := s.notificationRepo.FindAllNotifications(repoCriteria)
	if err != nil {
		return nil, err
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

func (s *notificationService) GetPlatformNotificationStats() (*repositories.PlatformNotificationStats, error) {
	return s.notificationRepo.GetPlatformNotificationStats()
}

func (s *notificationService) SendBulkNotification(adminID string, req *dto.SendBulkNotificationRequest) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}
	if admin.Role != models.UserRoleAdmin {
		return errors.New("insufficient permissions")
	}
	if !isValidNotificationType(req.Type) {
		return errors.New("invalid notification type")
	}

	var notifications []*dto.CreateNotificationRequest
	for _, userID := range req.UserIDs {
		notifications = append(notifications, &dto.CreateNotificationRequest{
			UserID:  userID,
			Type:    req.Type,
			Title:   req.Title,
			Message: req.Message,
			Data:    req.Data,
		})
	}

	bulkRequest := &dto.CreateBulkNotificationsRequest{Notifications: notifications}
	return s.CreateBulkNotifications(bulkRequest)
}

// ---------------- Helpers ----------------

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

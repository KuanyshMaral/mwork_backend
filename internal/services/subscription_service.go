package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/pkg/apperrors"
	"time"
)

// =======================
// 1. ИНТЕРФЕЙС ОБНОВЛЕН
// =======================
// Все методы теперь принимают 'db *gorm.DB'
type SubscriptionService interface {
	// Plan operations
	GetPlans(db *gorm.DB, role models.UserRole) ([]*models.SubscriptionPlan, error)
	GetPlan(db *gorm.DB, planID string) (*models.SubscriptionPlan, error)
	CreatePlan(db *gorm.DB, adminID string, req *models.CreatePlanRequest) error
	UpdatePlan(db *gorm.DB, adminID, planID string, req *models.UpdatePlanRequest) error
	DeletePlan(db *gorm.DB, adminID, planID string) error

	// User subscription operations
	GetUserSubscription(db *gorm.DB, userID string) (*models.UserSubscription, error)
	GetUserSubscriptionStats(db *gorm.DB, userID string) (*repositories.UserSubscriptionStats, error)
	CreateSubscription(db *gorm.DB, userID, planID string) (*models.UserSubscription, error)
	CancelSubscription(db *gorm.DB, userID string) error
	RenewSubscription(db *gorm.DB, userID, planID string) error
	CheckSubscriptionLimit(db *gorm.DB, userID, feature string) (bool, error)
	IncrementUsage(db *gorm.DB, userID, feature string) error
	ResetUsage(db *gorm.DB, userID string) error

	// Payment operations
	CreatePayment(db *gorm.DB, userID, planID string) (*models.PaymentResponse, error)
	ProcessPayment(db *gorm.DB, paymentID string) error
	GetPaymentHistory(db *gorm.DB, userID string) ([]*models.PaymentTransaction, error)
	GetPaymentStatus(db *gorm.DB, paymentID string) (*models.PaymentTransaction, error)

	// Robokassa integration
	InitRobokassaPayment(db *gorm.DB, userID, planID string) (*models.RobokassaInitResponse, error)
	ProcessRobokassaCallback(db *gorm.DB, data *models.RobokassaCallbackData) error
	CheckRobokassaPayment(db *gorm.DB, paymentID string) (*models.PaymentStatusResponse, error)

	// Admin operations
	GetPlatformSubscriptionStats(db *gorm.DB) (*repositories.PlatformSubscriptionStats, error)
	GetRevenueStats(db *gorm.DB, days int) (*repositories.RevenueStats, error)
	GetExpiringSubscriptions(db *gorm.DB, days int) ([]*models.UserSubscription, error)
	GetExpiredSubscriptions(db *gorm.DB) ([]*models.UserSubscription, error)
	ProcessExpiredSubscriptions(db *gorm.DB) error
}

// =======================
// 2. РЕАЛИЗАЦИЯ ОБНОВЛЕНА
// =======================
type subscriptionService struct {
	// ❌ 'db *gorm.DB' УДАЛЕНО ОТСЮДА
	subscriptionRepo repositories.SubscriptionRepository
	userRepo         repositories.UserRepository
	notificationRepo repositories.NotificationRepository
}

// ✅ Конструктор обновлен (db убран)
func NewSubscriptionService(
	// ❌ 'db *gorm.DB,' УДАЛЕНО
	subscriptionRepo repositories.SubscriptionRepository,
	userRepo repositories.UserRepository,
	notificationRepo repositories.NotificationRepository,
) SubscriptionService {
	return &subscriptionService{
		// ❌ 'db: db,' УДАЛЕНО
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

// Plan operations

func (s *subscriptionService) GetPlans(db *gorm.DB, role models.UserRole) ([]*models.SubscriptionPlan, error) {
	// ✅ Используем 'db' из параметра
	plans, err := s.subscriptionRepo.FindPlansByRole(db, role)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var result []*models.SubscriptionPlan
	for i := range plans {
		result = append(result, &plans[i])
	}
	return result, nil
}

func (s *subscriptionService) GetPlan(db *gorm.DB, planID string) (*models.SubscriptionPlan, error) {
	// ✅ Используем 'db' из параметра
	plan, err := s.subscriptionRepo.FindPlanByID(db, planID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}
	return plan, nil
}

func (s *subscriptionService) CreatePlan(db *gorm.DB, adminID string, req *models.CreatePlanRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleSubscriptionError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	featuresJSON, err := json.Marshal(req.Features)
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}
	limitsJSON, err := json.Marshal(req.Limits)
	if err != nil {
		return fmt.Errorf("failed to marshal limits: %w", err)
	}

	plan := &models.SubscriptionPlan{
		Name:     req.Name,
		Price:    req.Price,
		Currency: req.Currency,
		Duration: req.Duration,
		Features: datatypes.JSON(featuresJSON),
		Limits:   datatypes.JSON(limitsJSON),
		IsActive: req.IsActive,
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.CreatePlan(tx, plan); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *subscriptionService) UpdatePlan(db *gorm.DB, adminID, planID string, req *models.UpdatePlanRequest) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleSubscriptionError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	// ✅ Передаем tx
	plan, err := s.subscriptionRepo.FindPlanByID(tx, planID)
	if err != nil {
		return handleSubscriptionError(err)
	}

	// ... (логика обновления полей)
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Limits != nil {
		limitsJSON, err := json.Marshal(req.Limits)
		if err != nil {
			return fmt.Errorf("failed to marshal limits: %w", err)
		}
		plan.Limits = datatypes.JSON(limitsJSON)
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}
	// ... (конец логики обновления)

	// ✅ Передаем tx
	if err := s.subscriptionRepo.UpdatePlan(tx, plan); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *subscriptionService) DeletePlan(db *gorm.DB, adminID, planID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	admin, err := s.userRepo.FindByID(tx, adminID)
	if err != nil {
		return handleSubscriptionError(err)
	}
	if admin.Role != models.UserRoleAdmin {
		return apperrors.ErrInsufficientPermissions
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.DeletePlan(tx, planID); err != nil {
		return handleSubscriptionError(err)
	}
	return tx.Commit().Error
}

// User subscription operations

func (s *subscriptionService) GetUserSubscription(db *gorm.DB, userID string) (*models.UserSubscription, error) {
	// ✅ Используем 'db' из параметра
	sub, err := s.subscriptionRepo.FindUserSubscription(db, userID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}
	return sub, nil
}

func (s *subscriptionService) GetUserSubscriptionStats(db *gorm.DB, userID string) (*repositories.UserSubscriptionStats, error) {
	// ✅ Используем 'db' из параметра
	stats, err := s.subscriptionRepo.GetUserSubscriptionStats(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	return stats, nil
}

func (s *subscriptionService) CreateSubscription(db *gorm.DB, userID, planID string) (*models.UserSubscription, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	plan, err := s.subscriptionRepo.FindPlanByID(tx, planID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}

	endDate := s.calculateEndDate(time.Now(), plan.Duration)
	subscription := &models.UserSubscription{
		UserID:       userID,
		PlanID:       planID,
		Status:       models.SubscriptionStatusActive,
		InvID:        generateInvoiceID(),
		CurrentUsage: datatypes.JSON(`{"publications": 0, "responses": 0, "messages": 0, "promotions": 0}`),
		StartDate:    time.Now(),
		EndDate:      endDate,
		AutoRenew:    true,
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.CreateUserSubscription(tx, subscription); err != nil {
		return nil, apperrors.InternalError(err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	// ✅ Отправляем уведомление *после* коммита, передаем 'db' (пул)
	go s.notificationRepo.CreateSubscriptionExpiringNotification(db, userID, plan.Name, 30)

	return subscription, nil
}

func (s *subscriptionService) CancelSubscription(db *gorm.DB, userID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	subscription, err := s.subscriptionRepo.FindUserSubscription(tx, userID)
	if err != nil {
		return handleSubscriptionError(err)
	}
	if subscription.Status == models.SubscriptionStatusCancelled {
		return apperrors.ErrSubscriptionCancelled
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.CancelUserSubscription(tx, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *subscriptionService) RenewSubscription(db *gorm.DB, userID, planID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	plan, err := s.subscriptionRepo.FindPlanByID(tx, planID)
	if err != nil {
		return handleSubscriptionError(err)
	}

	newEndDate := s.calculateEndDate(time.Now(), plan.Duration)

	// ✅ Передаем tx
	if err := s.subscriptionRepo.RenewUserSubscription(tx, userID, planID, newEndDate); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *subscriptionService) CheckSubscriptionLimit(db *gorm.DB, userID, feature string) (bool, error) {
	// ✅ Используем 'db' из параметра
	ok, err := s.subscriptionRepo.CanUserPublish(db, userID)
	if err != nil {
		return false, apperrors.InternalError(err)
	}
	return ok, nil
}

func (s *subscriptionService) IncrementUsage(db *gorm.DB, userID, feature string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.subscriptionRepo.IncrementSubscriptionUsage(tx, userID, feature); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *subscriptionService) ResetUsage(db *gorm.DB, userID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	if err := s.subscriptionRepo.ResetSubscriptionUsage(tx, userID); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

// Payment operations

func (s *subscriptionService) CreatePayment(db *gorm.DB, userID, planID string) (*models.PaymentResponse, error) {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return nil, apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	plan, err := s.subscriptionRepo.FindPlanByID(tx, planID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}

	payment := &models.PaymentTransaction{
		UserID:         userID,
		SubscriptionID: "", // (Оставлено как в оригинале)
		Amount:         plan.Price,
		Status:         models.PaymentStatusPending,
		InvID:          generateInvoiceID(),
	}

	// ✅ Передаем tx
	if err := s.subscriptionRepo.CreatePaymentTransaction(tx, payment); err != nil {
		return nil, apperrors.InternalError(err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &models.PaymentResponse{
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Currency:  plan.Currency,
		Status:    string(payment.Status),
		InvoiceID: payment.InvID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

func (s *subscriptionService) ProcessPayment(db *gorm.DB, paymentID string) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	payment, err := s.subscriptionRepo.FindPaymentByID(tx, paymentID)
	if err != nil {
		return handleSubscriptionError(err)
	}

	paidAt := time.Now()
	// ✅ Передаем tx
	if err := s.subscriptionRepo.UpdatePaymentStatus(tx, payment.InvID, models.PaymentStatusPaid, &paidAt); err != nil {
		return apperrors.InternalError(err)
	}
	return tx.Commit().Error
}

func (s *subscriptionService) GetPaymentHistory(db *gorm.DB, userID string) ([]*models.PaymentTransaction, error) {
	// ✅ Используем 'db' из параметра
	payments, err := s.subscriptionRepo.FindPaymentsByUser(db, userID)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var result []*models.PaymentTransaction
	for i := range payments {
		result = append(result, &payments[i])
	}
	return result, nil
}

func (s *subscriptionService) GetPaymentStatus(db *gorm.DB, paymentID string) (*models.PaymentTransaction, error) {
	// ✅ Используем 'db' из параметра
	payment, err := s.subscriptionRepo.FindPaymentByID(db, paymentID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}
	return payment, nil
}

// Robokassa integration

func (s *subscriptionService) InitRobokassaPayment(db *gorm.DB, userID, planID string) (*models.RobokassaInitResponse, error) {
	// ✅ Используем 'db' из параметра
	plan, err := s.subscriptionRepo.FindPlanByID(db, planID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}

	// ✅ Передаем 'db' в CreatePayment
	payment, err := s.CreatePayment(db, userID, planID)
	if err != nil {
		return nil, err
	}

	paymentURL, err := s.generateRobokassaURL(payment.InvoiceID, payment.Amount, plan.Currency)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}

	return &models.RobokassaInitResponse{
		PaymentURL: paymentURL,
		InvoiceID:  payment.InvoiceID,
		Amount:     payment.Amount,
		Currency:   payment.Currency,
	}, nil
}

func (s *subscriptionService) ProcessRobokassaCallback(db *gorm.DB, data *models.RobokassaCallbackData) error {
	if !s.verifyRobokassaSignature(data) {
		return apperrors.ErrRobokassaError
	}

	// ✅ Начинаем *одну* транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	payment, err := s.subscriptionRepo.FindPaymentByInvID(tx, data.InvID)
	if err != nil {
		return handleSubscriptionError(err)
	}
	if payment.Amount != data.OutSum {
		return apperrors.ErrInvalidPaymentAmount
	}

	paidAt := time.Now()
	// ✅ Передаем tx
	if err := s.subscriptionRepo.UpdatePaymentStatus(tx, data.InvID, models.PaymentStatusPaid, &paidAt); err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Передаем tx
	_, err = s.subscriptionRepo.FindUserSubscription(tx, payment.UserID)
	if err != nil {
		// Ошибка - значит подписки нет. Создаем новую.
		if _, err := s.createSubscriptionInTx(tx, payment.UserID, payment.SubscriptionID); err != nil {
			return err
		}
	} else {
		// Подписка есть - обновляем.
		if err := s.renewSubscriptionInTx(tx, payment.UserID, payment.SubscriptionID); err != nil {
			return err
		}
	}

	return tx.Commit().Error
}

func (s *subscriptionService) CheckRobokassaPayment(db *gorm.DB, paymentID string) (*models.PaymentStatusResponse, error) {
	// ✅ Используем 'db' из параметра
	payment, err := s.subscriptionRepo.FindPaymentByID(db, paymentID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}

	return &models.PaymentStatusResponse{
		PaymentID: payment.ID,
		Status:    string(payment.Status),
		Amount:    payment.Amount,
		PaidAt:    *payment.PaidAt,
		InvoiceID: payment.InvID,
	}, nil
}

// Admin operations

func (s *subscriptionService) GetPlatformSubscriptionStats(db *gorm.DB) (*repositories.PlatformSubscriptionStats, error) {
	// ✅ Используем 'db' из параметра
	return s.subscriptionRepo.GetPlatformSubscriptionStats(db)
}

func (s *subscriptionService) GetRevenueStats(db *gorm.DB, days int) (*repositories.RevenueStats, error) {
	// ✅ Используем 'db' из параметра
	return s.subscriptionRepo.GetRevenueStats(db, days)
}

func (s *subscriptionService) GetExpiringSubscriptions(db *gorm.DB, days int) ([]*models.UserSubscription, error) {
	// ✅ Используем 'db' из параметра
	subscriptions, err := s.subscriptionRepo.FindExpiringSubscriptions(db, days)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var result []*models.UserSubscription
	for i := range subscriptions {
		result = append(result, &subscriptions[i])
	}
	return result, nil
}

func (s *subscriptionService) GetExpiredSubscriptions(db *gorm.DB) ([]*models.UserSubscription, error) {
	// ✅ Используем 'db' из параметра
	subscriptions, err := s.subscriptionRepo.FindExpiredSubscriptions(db)
	if err != nil {
		return nil, apperrors.InternalError(err)
	}
	var result []*models.UserSubscription
	for i := range subscriptions {
		result = append(result, &subscriptions[i])
	}
	return result, nil
}

func (s *subscriptionService) ProcessExpiredSubscriptions(db *gorm.DB) error {
	// ✅ Начинаем транзакцию из переданного 'db'
	tx := db.Begin()
	if tx.Error != nil {
		return apperrors.InternalError(tx.Error)
	}
	defer tx.Rollback()

	// ✅ Передаем tx
	expiredSubscriptions, err := s.subscriptionRepo.FindExpiredSubscriptions(tx)
	if err != nil {
		return apperrors.InternalError(err)
	}

	var notificationJobs []struct {
		UserID   string
		PlanName string
	}

	for _, subscription := range expiredSubscriptions {
		// ✅ Передаем tx
		if err := s.subscriptionRepo.UpdateSubscriptionStatus(tx, subscription.UserID, models.SubscriptionStatusExpired); err != nil {
			fmt.Printf("Failed to expire subscription for user %s: %v\n", subscription.UserID, err)
			continue
		}
		notificationJobs = append(notificationJobs, struct {
			UserID   string
			PlanName string
		}{UserID: subscription.UserID, PlanName: subscription.Plan.Name})
	}

	if err := tx.Commit().Error; err != nil {
		return apperrors.InternalError(err)
	}

	// ✅ Отправляем уведомления *после* коммита, передаем 'db' (пул)
	for _, job := range notificationJobs {
		go s.notificationRepo.CreateSubscriptionExpiringNotification(
			db, // 👈 Используем 'db' из параметра
			job.UserID,
			job.PlanName,
			0, // Expired
		)
	}

	return nil
}

// =======================
// 3. ХЕЛПЕРЫ
// =======================

// (Чистые функции - без изменений)
func (s *subscriptionService) calculateEndDate(startDate time.Time, duration string) time.Time {
	switch duration {
	case "yearly":
		return startDate.AddDate(1, 0, 0)
	case "monthly":
		return startDate.AddDate(0, 1, 0)
	case "weekly":
		return startDate.AddDate(0, 0, 7)
	default:
		return startDate.AddDate(0, 1, 0)
	}
}

// (Чистые функции - без изменений)
func (s *subscriptionService) generateRobokassaURL(invID string, amount float64, currency string) (string, error) {
	return fmt.Sprintf("https://auth.robokassa.kz/Merchant/Index.aspx?InvId=%s&OutSum=%.2f&Culture=ru", invID, amount), nil
}

// (Чистые функции - без изменений)
func (s *subscriptionService) verifyRobokassaSignature(data *models.RobokassaCallbackData) bool {
	return true
}

// (Чистые функции - без изменений)
func generateInvoiceID() string {
	return fmt.Sprintf("INV%d", time.Now().UnixNano())
}

// (Внутренний хелпер для ProcessRobokassaCallback - без изменений, т.к. он уже принимает tx)
func (s *subscriptionService) createSubscriptionInTx(tx *gorm.DB, userID, planID string) (*models.UserSubscription, error) {
	plan, err := s.subscriptionRepo.FindPlanByID(tx, planID)
	if err != nil {
		return nil, handleSubscriptionError(err)
	}
	endDate := s.calculateEndDate(time.Now(), plan.Duration)
	subscription := &models.UserSubscription{
		UserID:       userID,
		PlanID:       planID,
		Status:       models.SubscriptionStatusActive,
		InvID:        generateInvoiceID(),
		CurrentUsage: datatypes.JSON(`{"publications": 0, "responses": 0, "messages": 0, "promotions": 0}`),
		StartDate:    time.Now(),
		EndDate:      endDate,
		AutoRenew:    true,
	}
	if err := s.subscriptionRepo.CreateUserSubscription(tx, subscription); err != nil {
		return nil, apperrors.InternalError(err)
	}
	return subscription, nil
}

// (Внутренний хелпер для ProcessRobokassaCallback - без изменений, т.к. он уже принимает tx)
func (s *subscriptionService) renewSubscriptionInTx(tx *gorm.DB, userID, planID string) error {
	plan, err := s.subscriptionRepo.FindPlanByID(tx, planID)
	if err != nil {
		return handleSubscriptionError(err)
	}
	newEndDate := s.calculateEndDate(time.Now(), plan.Duration)
	return s.subscriptionRepo.RenewUserSubscription(tx, userID, planID, newEndDate)
}

// (Вспомогательный хелпер для ошибок - без изменений)
func handleSubscriptionError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) ||
		errors.Is(err, repositories.ErrSubscriptionPlanNotFound) ||
		errors.Is(err, repositories.ErrSubscriptionNotFound) ||
		errors.Is(err, repositories.ErrPaymentNotFound) {
		return apperrors.ErrNotFound(err)
	}
	if errors.Is(err, repositories.ErrUserNotFound) {
		return apperrors.ErrNotFound(err)
	}
	return apperrors.InternalError(err)
}

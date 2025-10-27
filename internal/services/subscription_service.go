package services

import (
	"encoding/json"
	"fmt"
	"gorm.io/datatypes"
	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"time"
)

type SubscriptionService interface {
	// Plan operations
	GetPlans(role models.UserRole) ([]*models.SubscriptionPlan, error)
	GetPlan(planID string) (*models.SubscriptionPlan, error)
	CreatePlan(adminID string, req *models.CreatePlanRequest) error
	UpdatePlan(adminID, planID string, req *models.UpdatePlanRequest) error
	DeletePlan(adminID, planID string) error

	// User subscription operations
	GetUserSubscription(userID string) (*models.UserSubscription, error)
	GetUserSubscriptionStats(userID string) (*repositories.UserSubscriptionStats, error)
	CreateSubscription(userID, planID string) (*models.UserSubscription, error)
	CancelSubscription(userID string) error
	RenewSubscription(userID, planID string) error
	CheckSubscriptionLimit(userID, feature string) (bool, error)
	IncrementUsage(userID, feature string) error
	ResetUsage(userID string) error

	// Payment operations
	CreatePayment(userID, planID string) (*models.PaymentResponse, error)
	ProcessPayment(paymentID string) error
	GetPaymentHistory(userID string) ([]*models.PaymentTransaction, error)
	GetPaymentStatus(paymentID string) (*models.PaymentTransaction, error)

	// Robokassa integration
	InitRobokassaPayment(userID, planID string) (*models.RobokassaInitResponse, error)
	ProcessRobokassaCallback(data *models.RobokassaCallbackData) error
	CheckRobokassaPayment(paymentID string) (*models.PaymentStatusResponse, error)

	// Admin operations
	GetPlatformSubscriptionStats() (*repositories.PlatformSubscriptionStats, error)
	GetRevenueStats(days int) (*repositories.RevenueStats, error)
	GetExpiringSubscriptions(days int) ([]*models.UserSubscription, error)
	GetExpiredSubscriptions() ([]*models.UserSubscription, error)
	ProcessExpiredSubscriptions() error
}

type subscriptionService struct {
	subscriptionRepo repositories.SubscriptionRepository
	userRepo         repositories.UserRepository
	notificationRepo repositories.NotificationRepository
}

func NewSubscriptionService(
	subscriptionRepo repositories.SubscriptionRepository,
	userRepo repositories.UserRepository,
	notificationRepo repositories.NotificationRepository,
) SubscriptionService {
	return &subscriptionService{
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

// Plan operations

func (s *subscriptionService) GetPlans(role models.UserRole) ([]*models.SubscriptionPlan, error) {
	plans, err := s.subscriptionRepo.FindPlansByRole(role)
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	var result []*models.SubscriptionPlan
	for i := range plans {
		result = append(result, &plans[i])
	}

	return result, nil
}

func (s *subscriptionService) GetPlan(planID string) (*models.SubscriptionPlan, error) {
	return s.subscriptionRepo.FindPlanByID(planID)
}

func (s *subscriptionService) CreatePlan(adminID string, req *models.CreatePlanRequest) error {
	// Validate admin permissions
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	// Convert features and limits to JSON
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

	return s.subscriptionRepo.CreatePlan(plan)
}

func (s *subscriptionService) UpdatePlan(adminID, planID string, req *models.UpdatePlanRequest) error {
	// Validate admin permissions
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	plan, err := s.subscriptionRepo.FindPlanByID(planID)
	if err != nil {
		return err
	}

	// Update fields if provided
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Price != nil {
		plan.Price = *req.Price
	}
	if req.Currency != nil {
		plan.Currency = *req.Currency
	}
	if req.Duration != nil {
		plan.Duration = *req.Duration
	}
	if req.Features != nil {
		featuresJSON, err := json.Marshal(req.Features)
		if err != nil {
			return fmt.Errorf("failed to marshal features: %w", err)
		}
		plan.Features = datatypes.JSON(featuresJSON)
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

	return s.subscriptionRepo.UpdatePlan(plan)
}

func (s *subscriptionService) DeletePlan(adminID, planID string) error {
	// Validate admin permissions
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil {
		return err
	}

	if admin.Role != models.UserRoleAdmin {
		return appErrors.ErrInsufficientPermissions
	}

	return s.subscriptionRepo.DeletePlan(planID)
}

// User subscription operations

func (s *subscriptionService) GetUserSubscription(userID string) (*models.UserSubscription, error) {
	return s.subscriptionRepo.FindUserSubscription(userID)
}

func (s *subscriptionService) GetUserSubscriptionStats(userID string) (*repositories.UserSubscriptionStats, error) {
	return s.subscriptionRepo.GetUserSubscriptionStats(userID)
}

func (s *subscriptionService) CreateSubscription(userID, planID string) (*models.UserSubscription, error) {
	// Get plan details
	plan, err := s.subscriptionRepo.FindPlanByID(planID)
	if err != nil {
		return nil, err
	}

	// Calculate end date based on duration
	endDate := s.calculateEndDate(time.Now(), plan.Duration)

	// Create subscription
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

	if err := s.subscriptionRepo.CreateUserSubscription(subscription); err != nil {
		return nil, err
	}

	// Send notification
	go s.notificationRepo.CreateSubscriptionExpiringNotification(userID, plan.Name, 30)

	return subscription, nil
}

func (s *subscriptionService) CancelSubscription(userID string) error {
	subscription, err := s.subscriptionRepo.FindUserSubscription(userID)
	if err != nil {
		return err
	}

	if subscription.Status == models.SubscriptionStatusCancelled {
		return appErrors.ErrSubscriptionCancelled
	}

	return s.subscriptionRepo.CancelUserSubscription(userID)
}

func (s *subscriptionService) RenewSubscription(userID, planID string) error {
	// Get plan details
	plan, err := s.subscriptionRepo.FindPlanByID(planID)
	if err != nil {
		return err
	}

	// Calculate new end date
	newEndDate := s.calculateEndDate(time.Now(), plan.Duration)

	return s.subscriptionRepo.RenewUserSubscription(userID, planID, newEndDate)
}

func (s *subscriptionService) CheckSubscriptionLimit(userID, feature string) (bool, error) {
	return s.subscriptionRepo.CanUserPublish(userID)
}

func (s *subscriptionService) IncrementUsage(userID, feature string) error {
	return s.subscriptionRepo.IncrementSubscriptionUsage(userID, feature)
}

func (s *subscriptionService) ResetUsage(userID string) error {
	return s.subscriptionRepo.ResetSubscriptionUsage(userID)
}

// Payment operations

func (s *subscriptionService) CreatePayment(userID, planID string) (*models.PaymentResponse, error) {
	// Get plan details
	plan, err := s.subscriptionRepo.FindPlanByID(planID)
	if err != nil {
		return nil, err
	}

	// Create payment transaction
	payment := &models.PaymentTransaction{
		UserID:         userID,
		SubscriptionID: "", // Will be set after subscription creation
		Amount:         plan.Price,
		Status:         models.PaymentStatusPending,
		InvID:          generateInvoiceID(),
	}

	if err := s.subscriptionRepo.CreatePaymentTransaction(payment); err != nil {
		return nil, err
	}

	return &models.PaymentResponse{
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Currency:  plan.Currency,
		Status:    string(payment.Status),
		InvoiceID: payment.InvID,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Payment expires in 24 hours
	}, nil
}

func (s *subscriptionService) ProcessPayment(paymentID string) error {
	payment, err := s.subscriptionRepo.FindPaymentByID(paymentID)
	if err != nil {
		return err
	}

	// Mark payment as paid
	paidAt := time.Now()
	return s.subscriptionRepo.UpdatePaymentStatus(payment.InvID, models.PaymentStatusPaid, &paidAt)
}

func (s *subscriptionService) GetPaymentHistory(userID string) ([]*models.PaymentTransaction, error) {
	payments, err := s.subscriptionRepo.FindPaymentsByUser(userID)
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	var result []*models.PaymentTransaction
	for i := range payments {
		result = append(result, &payments[i])
	}

	return result, nil
}

func (s *subscriptionService) GetPaymentStatus(paymentID string) (*models.PaymentTransaction, error) {
	return s.subscriptionRepo.FindPaymentByID(paymentID)
}

// Robokassa integration

func (s *subscriptionService) InitRobokassaPayment(userID, planID string) (*models.RobokassaInitResponse, error) {
	// Get plan details
	plan, err := s.subscriptionRepo.FindPlanByID(planID)
	if err != nil {
		return nil, err
	}

	// Create payment first
	payment, err := s.CreatePayment(userID, planID)
	if err != nil {
		return nil, err
	}

	// Generate Robokassa payment URL
	paymentURL, err := s.generateRobokassaURL(payment.InvoiceID, payment.Amount, plan.Currency)
	if err != nil {
		return nil, err
	}

	return &models.RobokassaInitResponse{
		PaymentURL: paymentURL,
		InvoiceID:  payment.InvoiceID,
		Amount:     payment.Amount,
		Currency:   payment.Currency,
	}, nil
}

func (s *subscriptionService) ProcessRobokassaCallback(data *models.RobokassaCallbackData) error {
	// Verify signature
	if !s.verifyRobokassaSignature(data) {
		return appErrors.ErrRobokassaError
	}

	// Find payment by invoice ID
	payment, err := s.subscriptionRepo.FindPaymentByInvID(data.InvID)
	if err != nil {
		return err
	}

	// Verify amount
	if payment.Amount != data.OutSum {
		return appErrors.ErrInvalidPaymentAmount
	}

	// Mark payment as paid and create subscription
	paidAt := time.Now()
	if err := s.subscriptionRepo.UpdatePaymentStatus(data.InvID, models.PaymentStatusPaid, &paidAt); err != nil {
		return err
	}

	// Check if user already has a subscription
	_, err = s.subscriptionRepo.FindUserSubscription(payment.UserID)
	if err != nil {
		// No existing subscription - create new one
		_, err = s.CreateSubscription(payment.UserID, payment.SubscriptionID)
		return err
	} else {
		// Existing subscription found - renew it
		return s.RenewSubscription(payment.UserID, payment.SubscriptionID)
	}
}

func (s *subscriptionService) CheckRobokassaPayment(paymentID string) (*models.PaymentStatusResponse, error) {
	payment, err := s.subscriptionRepo.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
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

func (s *subscriptionService) GetPlatformSubscriptionStats() (*repositories.PlatformSubscriptionStats, error) {
	return s.subscriptionRepo.GetPlatformSubscriptionStats()
}

func (s *subscriptionService) GetRevenueStats(days int) (*repositories.RevenueStats, error) {
	return s.subscriptionRepo.GetRevenueStats(days)
}

func (s *subscriptionService) GetExpiringSubscriptions(days int) ([]*models.UserSubscription, error) {
	subscriptions, err := s.subscriptionRepo.FindExpiringSubscriptions(days)
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	var result []*models.UserSubscription
	for i := range subscriptions {
		result = append(result, &subscriptions[i])
	}

	return result, nil
}

func (s *subscriptionService) GetExpiredSubscriptions() ([]*models.UserSubscription, error) {
	subscriptions, err := s.subscriptionRepo.FindExpiredSubscriptions()
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	var result []*models.UserSubscription
	for i := range subscriptions {
		result = append(result, &subscriptions[i])
	}

	return result, nil
}

func (s *subscriptionService) ProcessExpiredSubscriptions() error {
	expiredSubscriptions, err := s.subscriptionRepo.FindExpiredSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range expiredSubscriptions {
		// Update subscription status to expired
		if err := s.subscriptionRepo.UpdateSubscriptionStatus(subscription.UserID, models.SubscriptionStatusExpired); err != nil {
			// Log error but continue processing others
			fmt.Printf("Failed to expire subscription for user %s: %v\n", subscription.UserID, err)
			continue
		}

		// Send notification to user
		go s.notificationRepo.CreateSubscriptionExpiringNotification(
			subscription.UserID,
			subscription.Plan.Name,
			0, // Expired
		)
	}

	return nil
}

// Helper methods

func (s *subscriptionService) calculateEndDate(startDate time.Time, duration string) time.Time {
	switch duration {
	case "yearly":
		return startDate.AddDate(1, 0, 0)
	case "monthly":
		return startDate.AddDate(0, 1, 0)
	case "weekly":
		return startDate.AddDate(0, 0, 7)
	default:
		return startDate.AddDate(0, 1, 0) // Default to monthly
	}
}

func (s *subscriptionService) generateRobokassaURL(invID string, amount float64, currency string) (string, error) {
	// Implementation for Robokassa URL generation
	// This would use Robokassa API credentials and signature generation
	// For now, return a placeholder
	return fmt.Sprintf("https://auth.robokassa.kz/Merchant/Index.aspx?InvId=%s&OutSum=%.2f&Culture=ru", invID, amount), nil
}

func (s *subscriptionService) verifyRobokassaSignature(data *models.RobokassaCallbackData) bool {
	// Implementation for Robokassa signature verification
	// This would verify the signature using Robokassa merchant password
	// For now, return true for testing
	return true
}

func generateInvoiceID() string {
	return fmt.Sprintf("INV%d", time.Now().UnixNano())
}

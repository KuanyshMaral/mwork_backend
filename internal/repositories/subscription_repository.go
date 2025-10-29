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
	ErrSubscriptionNotFound     = errors.New("subscription not found")
	ErrSubscriptionPlanNotFound = errors.New("subscription plan not found")
	ErrPaymentNotFound          = errors.New("payment transaction not found")
	ErrSubscriptionLimit        = errors.New("subscription limit reached")
)

type SubscriptionRepository interface {
	// SubscriptionPlan operations
	CreatePlan(plan *models.SubscriptionPlan) error
	FindPlanByID(id string) (*models.SubscriptionPlan, error)
	FindPlanByName(name string) (*models.SubscriptionPlan, error)
	FindActivePlans() ([]models.SubscriptionPlan, error)
	FindPlansByRole(role models.UserRole) ([]models.SubscriptionPlan, error)
	UpdatePlan(plan *models.SubscriptionPlan) error
	DeletePlan(id string) error

	// UserSubscription operations
	CreateUserSubscription(subscription *models.UserSubscription) error
	FindUserSubscription(userID string) (*models.UserSubscription, error)
	FindUserSubscriptionByInvID(invID string) (*models.UserSubscription, error)
	UpdateUserSubscription(subscription *models.UserSubscription) error
	UpdateSubscriptionStatus(userID string, status models.SubscriptionStatus) error
	CancelUserSubscription(userID string) error
	RenewUserSubscription(userID string, newPlanID string, newEndDate time.Time) error
	IncrementSubscriptionUsage(userID string, feature string) error
	DecrementSubscriptionUsage(userID string, feature string) error
	ResetSubscriptionUsage(userID string) error
	FindExpiringSubscriptions(days int) ([]models.UserSubscription, error)
	FindExpiredSubscriptions() ([]models.UserSubscription, error)

	// PaymentTransaction operations
	CreatePaymentTransaction(payment *models.PaymentTransaction) error
	FindPaymentByID(id string) (*models.PaymentTransaction, error)
	FindPaymentByInvID(invID string) (*models.PaymentTransaction, error)
	FindPaymentsByUser(userID string) ([]models.PaymentTransaction, error)
	UpdatePaymentStatus(invID string, status models.PaymentStatus, paidAt *time.Time) error
	DeletePayment(id string) error

	// Usage and limits operations
	GetUserUsage(userID string) (map[string]int, error)
	GetUserLimits(userID string) (map[string]int, error)
	CanUserPublish(userID string) (bool, error)
	CanUserRespond(userID string) (bool, error)
	GetUserSubscriptionStats(userID string) (*UserSubscriptionStats, error)

	// Admin operations
	GetPlatformSubscriptionStats() (*PlatformSubscriptionStats, error)
	GetRevenueStats(days int) (*RevenueStats, error)

	GetSubscriptionMetrics(dateFrom, dateTo time.Time) (*SubscriptionMetrics, error)
}

type SubscriptionRepositoryImpl struct {
	db *gorm.DB
}

// Usage and limits structures
type Usage struct {
	Publications int `json:"publications"`
	Responses    int `json:"responses"`
	Messages     int `json:"messages"`
	Promotions   int `json:"promotions"`
}

type Limits struct {
	Publications int `json:"publications"`
	Responses    int `json:"responses"`
	Messages     int `json:"messages"`
	Promotions   int `json:"promotions"`
}

type SubscriptionMetrics struct {
	TotalSubscribers int64   `json:"totalSubscribers"`
	TotalRevenue     float64 `json:"totalRevenue"`
	MRR              float64 `json:"mrr"`       // Monthly Recurring Revenue
	ARPU             float64 `json:"arpu"`      // Average Revenue Per User
	ChurnRate        float64 `json:"churnRate"` // в процентах
}

// Statistics structures
type UserSubscriptionStats struct {
	PlanName       string                    `json:"plan_name"`
	Status         models.SubscriptionStatus `json:"status"`
	StartDate      time.Time                 `json:"start_date"`
	EndDate        time.Time                 `json:"end_date"`
	CurrentUsage   Usage                     `json:"current_usage"`
	PlanLimits     Limits                    `json:"plan_limits"`
	DaysRemaining  int                       `json:"days_remaining"`
	IsExpiringSoon bool                      `json:"is_expiring_soon"`
	CanPublish     bool                      `json:"can_publish"`
	CanRespond     bool                      `json:"can_respond"`
}

type PlatformSubscriptionStats struct {
	TotalSubscriptions      int64            `json:"total_subscriptions"`
	ActiveSubscriptions     int64            `json:"active_subscriptions"`
	ExpiredSubscriptions    int64            `json:"expired_subscriptions"`
	CanceledSubscriptions   int64            `json:"canceled_subscriptions"`
	TotalRevenue            float64          `json:"total_revenue"`
	MonthlyRecurringRevenue float64          `json:"monthly_recurring_revenue"`
	ByPlan                  map[string]int64 `json:"by_plan"`
}

type RevenueStats struct {
	TotalRevenue     float64            `json:"total_revenue"`
	TodayRevenue     float64            `json:"today_revenue"`
	ThisWeekRevenue  float64            `json:"this_week_revenue"`
	ThisMonthRevenue float64            `json:"this_month_revenue"`
	ByPlan           map[string]float64 `json:"by_plan"`
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &SubscriptionRepositoryImpl{db: db}
}

// SubscriptionPlan operations

func (r *SubscriptionRepositoryImpl) CreatePlan(plan *models.SubscriptionPlan) error {
	return r.db.Create(plan).Error
}

func (r *SubscriptionRepositoryImpl) FindPlanByID(id string) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	err := r.db.First(&plan, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *SubscriptionRepositoryImpl) FindPlanByName(name string) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	err := r.db.Where("name = ?", name).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *SubscriptionRepositoryImpl) FindActivePlans() ([]models.SubscriptionPlan, error) {
	var plans []models.SubscriptionPlan
	err := r.db.Where("is_active = ?", true).Order("price ASC").Find(&plans).Error
	return plans, err
}

func (r *SubscriptionRepositoryImpl) FindPlansByRole(role models.UserRole) ([]models.SubscriptionPlan, error) {
	var plans []models.SubscriptionPlan

	// Определяем префикс плана по роли
	rolePrefix := getPlanPrefixByRole(role)

	err := r.db.Where("is_active = ? AND name LIKE ?", true, rolePrefix+"%").
		Order("price ASC").Find(&plans).Error
	return plans, err
}

func (r *SubscriptionRepositoryImpl) UpdatePlan(plan *models.SubscriptionPlan) error {
	result := r.db.Model(plan).Updates(map[string]interface{}{
		"name":       plan.Name,
		"price":      plan.Price,
		"currency":   plan.Currency,
		"duration":   plan.Duration,
		"features":   plan.Features,
		"limits":     plan.Limits,
		"is_active":  plan.IsActive,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionPlanNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) DeletePlan(id string) error {
	result := r.db.Where("id = ?", id).Delete(&models.SubscriptionPlan{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionPlanNotFound
	}
	return nil
}

// UserSubscription operations

func (r *SubscriptionRepositoryImpl) CreateUserSubscription(subscription *models.UserSubscription) error {
	return r.db.Create(subscription).Error
}

func (r *SubscriptionRepositoryImpl) FindUserSubscription(userID string) (*models.UserSubscription, error) {
	var subscription models.UserSubscription
	err := r.db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepositoryImpl) FindUserSubscriptionByInvID(invID string) (*models.UserSubscription, error) {
	var subscription models.UserSubscription
	err := r.db.Preload("Plan").Where("inv_id = ?", invID).First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepositoryImpl) UpdateUserSubscription(subscription *models.UserSubscription) error {
	result := r.db.Model(subscription).Updates(map[string]interface{}{
		"plan_id":       subscription.PlanID,
		"status":        subscription.Status,
		"current_usage": subscription.CurrentUsage,
		"start_date":    subscription.StartDate,
		"end_date":      subscription.EndDate,
		"auto_renew":    subscription.AutoRenew,
		"cancelled_at":  subscription.CancelledAt,
		"updated_at":    time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) UpdateSubscriptionStatus(userID string, status models.SubscriptionStatus) error {
	result := r.db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) CancelUserSubscription(userID string) error {
	result := r.db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"status":       models.SubscriptionStatusCancelled,
		"cancelled_at": time.Now(),
		"updated_at":   time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) RenewUserSubscription(userID string, newPlanID string, newEndDate time.Time) error {
	// Используем транзакцию для обновления подписки
	return r.db.Transaction(func(tx *gorm.DB) error {
		var subscription models.UserSubscription
		if err := tx.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
			return ErrSubscriptionNotFound
		}

		// Обновляем подписку
		updates := map[string]interface{}{
			"plan_id":       newPlanID,
			"status":        models.SubscriptionStatusActive,
			"current_usage": datatypes.JSON(`{"publications": 0, "responses": 0, "messages": 0, "promotions": 0}`),
			"start_date":    time.Now(),
			"end_date":      newEndDate,
			"cancelled_at":  nil,
			"updated_at":    time.Now(),
		}

		if err := tx.Model(&subscription).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *SubscriptionRepositoryImpl) IncrementSubscriptionUsage(userID string, feature string) error {
	// Используем транзакцию для атомарного обновления
	return r.db.Transaction(func(tx *gorm.DB) error {
		var subscription models.UserSubscription
		if err := tx.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
			return ErrSubscriptionNotFound
		}

		// Проверяем лимиты
		if !r.canUseFeature(&subscription, feature) {
			return ErrSubscriptionLimit
		}

		// Обновляем usage - ИСПРАВЛЕННЫЙ КОД
		var usage map[string]int
		if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
			return fmt.Errorf("failed to unmarshal usage: %w", err)
		}

		usage[feature]++

		newUsage, err := json.Marshal(usage)
		if err != nil {
			return fmt.Errorf("failed to marshal usage: %w", err)
		}

		subscription.CurrentUsage = datatypes.JSON(newUsage)
		return tx.Save(&subscription).Error
	})
}

func (r *SubscriptionRepositoryImpl) DecrementSubscriptionUsage(userID string, feature string) error {
	// Используем транзакцию для атомарного обновления
	return r.db.Transaction(func(tx *gorm.DB) error {
		var subscription models.UserSubscription
		if err := tx.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
			return ErrSubscriptionNotFound
		}

		// Обновляем usage
		var usage map[string]int
		if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
			return fmt.Errorf("failed to unmarshal usage: %w", err)
		}

		// Декрементируем, но не ниже 0
		if usage[feature] > 0 {
			usage[feature]--
		}

		newUsage, err := json.Marshal(usage)
		if err != nil {
			return fmt.Errorf("failed to marshal usage: %w", err)
		}

		subscription.CurrentUsage = datatypes.JSON(newUsage)
		return tx.Save(&subscription).Error
	})
}

func (r *SubscriptionRepositoryImpl) ResetSubscriptionUsage(userID string) error {
	result := r.db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Update("current_usage",
		datatypes.JSON(`{"publications": 0, "responses": 0, "messages": 0, "promotions": 0}`))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) FindExpiringSubscriptions(days int) ([]models.UserSubscription, error) {
	var subscriptions []models.UserSubscription
	expiryDate := time.Now().AddDate(0, 0, days)

	err := r.db.Preload("Plan").Where("status = ? AND end_date <= ? AND end_date > ?",
		models.SubscriptionStatusActive, expiryDate, time.Now()).
		Order("end_date ASC").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *SubscriptionRepositoryImpl) FindExpiredSubscriptions() ([]models.UserSubscription, error) {
	var subscriptions []models.UserSubscription

	err := r.db.Preload("Plan").Where("status = ? AND end_date < ?",
		models.SubscriptionStatusActive, time.Now()).
		Order("end_date ASC").
		Find(&subscriptions).Error
	return subscriptions, err
}

// PaymentTransaction operations

func (r *SubscriptionRepositoryImpl) CreatePaymentTransaction(payment *models.PaymentTransaction) error {
	return r.db.Create(payment).Error
}

func (r *SubscriptionRepositoryImpl) FindPaymentByID(id string) (*models.PaymentTransaction, error) {
	var payment models.PaymentTransaction
	err := r.db.Preload("Subscription").Preload("Subscription.Plan").
		First(&payment, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *SubscriptionRepositoryImpl) FindPaymentByInvID(invID string) (*models.PaymentTransaction, error) {
	var payment models.PaymentTransaction
	err := r.db.Preload("Subscription").Preload("Subscription.Plan").
		Where("inv_id = ?", invID).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *SubscriptionRepositoryImpl) FindPaymentsByUser(userID string) ([]models.PaymentTransaction, error) {
	var payments []models.PaymentTransaction
	err := r.db.Preload("Subscription").Preload("Subscription.Plan").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&payments).Error
	return payments, err
}

func (r *SubscriptionRepositoryImpl) UpdatePaymentStatus(invID string, status models.PaymentStatus, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if paidAt != nil {
		updates["paid_at"] = paidAt
	}

	result := r.db.Model(&models.PaymentTransaction{}).Where("inv_id = ?", invID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) DeletePayment(id string) error {
	result := r.db.Where("id = ?", id).Delete(&models.PaymentTransaction{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

// Usage and limits operations

func (r *SubscriptionRepositoryImpl) GetUserUsage(userID string) (map[string]int, error) {
	var subscription models.UserSubscription
	err := r.db.Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	var usage map[string]int
	if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage: %w", err)
	}

	return usage, nil
}

func (r *SubscriptionRepositoryImpl) GetUserLimits(userID string) (map[string]int, error) {
	var subscription models.UserSubscription
	err := r.db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	var limits map[string]int
	if err := json.Unmarshal(subscription.Plan.Limits, &limits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
	}

	return limits, nil
}

func (r *SubscriptionRepositoryImpl) CanUserPublish(userID string) (bool, error) {
	return r.canUseFeatureByUserID(userID, "publications")
}

func (r *SubscriptionRepositoryImpl) CanUserRespond(userID string) (bool, error) {
	return r.canUseFeatureByUserID(userID, "responses")
}

func (r *SubscriptionRepositoryImpl) GetUserSubscriptionStats(userID string) (*UserSubscriptionStats, error) {
	var subscription models.UserSubscription
	err := r.db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	// Получаем usage - ИСПРАВЛЕННЫЙ КОД
	var usage Usage
	if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage: %w", err)
	}

	// Получаем limits - ИСПРАВЛЕННЫЙ КОД
	var limits Limits
	if err := json.Unmarshal(subscription.Plan.Limits, &limits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
	}

	// Рассчитываем оставшиеся дни
	daysRemaining := int(subscription.EndDate.Sub(time.Now()).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	stats := &UserSubscriptionStats{
		PlanName:       subscription.Plan.Name,
		Status:         subscription.Status,
		StartDate:      subscription.StartDate,
		EndDate:        subscription.EndDate,
		CurrentUsage:   usage,
		PlanLimits:     limits,
		DaysRemaining:  daysRemaining,
		IsExpiringSoon: daysRemaining <= 7,
		CanPublish:     usage.Publications < limits.Publications,
		CanRespond:     usage.Responses < limits.Responses,
	}

	return stats, nil
}

// Admin operations

func (r *SubscriptionRepositoryImpl) GetPlatformSubscriptionStats() (*PlatformSubscriptionStats, error) {
	var stats PlatformSubscriptionStats

	// Total subscriptions
	if err := r.db.Model(&models.UserSubscription{}).Count(&stats.TotalSubscriptions).Error; err != nil {
		return nil, err
	}

	// Active subscriptions
	if err := r.db.Model(&models.UserSubscription{}).Where("status = ?", models.SubscriptionStatusActive).
		Count(&stats.ActiveSubscriptions).Error; err != nil {
		return nil, err
	}

	// Expired subscriptions
	if err := r.db.Model(&models.UserSubscription{}).Where("status = ?", models.SubscriptionStatusExpired).
		Count(&stats.ExpiredSubscriptions).Error; err != nil {
		return nil, err
	}

	// Canceled subscriptions
	if err := r.db.Model(&models.UserSubscription{}).Where("status = ?", models.SubscriptionStatusCancelled).
		Count(&stats.CanceledSubscriptions).Error; err != nil {
		return nil, err
	}

	// Total revenue
	if err := r.db.Model(&models.PaymentTransaction{}).Where("status = ?", models.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue).Error; err != nil {
		return nil, err
	}

	// Monthly recurring revenue (last 30 days)
	monthAgo := time.Now().AddDate(0, -1, 0)
	if err := r.db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND paid_at >= ?", models.PaymentStatusPaid, monthAgo).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.MonthlyRecurringRevenue).Error; err != nil {
		return nil, err
	}

	// Subscriptions by plan
	stats.ByPlan = make(map[string]int64)
	var planStats []struct {
		PlanName string
		Count    int64
	}

	err := r.db.Model(&models.UserSubscription{}).
		Select("p.name as plan_name, COUNT(*) as count").
		Joins("LEFT JOIN subscription_plans p ON user_subscriptions.plan_id = p.id").
		Group("p.name").Scan(&planStats).Error

	if err != nil {
		return nil, err
	}

	for _, ps := range planStats {
		stats.ByPlan[ps.PlanName] = ps.Count
	}

	return &stats, nil
}

func (r *SubscriptionRepositoryImpl) GetRevenueStats(days int) (*RevenueStats, error) {
	var stats RevenueStats
	now := time.Now()

	// Total revenue
	if err := r.db.Model(&models.PaymentTransaction{}).Where("status = ?", models.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue).Error; err != nil {
		return nil, err
	}

	// Today revenue
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := r.db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND paid_at >= ?", models.PaymentStatusPaid, todayStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TodayRevenue).Error; err != nil {
		return nil, err
	}

	// This week revenue
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))
	if err := r.db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND paid_at >= ?", models.PaymentStatusPaid, weekStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.ThisWeekRevenue).Error; err != nil {
		return nil, err
	}

	// This month revenue
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := r.db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND paid_at >= ?", models.PaymentStatusPaid, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.ThisMonthRevenue).Error; err != nil {
		return nil, err
	}

	// Revenue by plan
	stats.ByPlan = make(map[string]float64)
	var revenueStats []struct {
		PlanName string
		Revenue  float64
	}

	err := r.db.Model(&models.PaymentTransaction{}).
		Select("p.name as plan_name, COALESCE(SUM(pt.amount), 0) as revenue").
		Joins("LEFT JOIN user_subscriptions us ON pt.subscription_id = us.id").
		Joins("LEFT JOIN subscription_plans p ON us.plan_id = p.id").
		Where("pt.status = ?", models.PaymentStatusPaid).
		Group("p.name").Scan(&revenueStats).Error

	if err != nil {
		return nil, err
	}

	for _, rs := range revenueStats {
		stats.ByPlan[rs.PlanName] = rs.Revenue
	}

	return &stats, nil
}

// Helper methods

func (r *SubscriptionRepositoryImpl) canUseFeature(subscription *models.UserSubscription, feature string) bool {
	if subscription.Status != models.SubscriptionStatusActive {
		return false
	}

	var usage map[string]int
	var limits map[string]int

	// ИСПРАВЛЕННЫЙ КОД - используем json.Unmarshal вместо .Unmarshal()
	if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
		return false
	}
	if err := json.Unmarshal(subscription.Plan.Limits, &limits); err != nil {
		return false
	}

	return usage[feature] < limits[feature]
}

func (r *SubscriptionRepositoryImpl) canUseFeatureByUserID(userID string, feature string) (bool, error) {
	subscription, err := r.FindUserSubscription(userID)
	if err != nil {
		return false, err
	}

	return r.canUseFeature(subscription, feature), nil
}

func getPlanPrefixByRole(role models.UserRole) string {
	switch role {
	case models.UserRoleModel:
		return "MWork"
	case models.UserRoleEmployer:
		return "Pro"
	default:
		return "Free"
	}
}

func (r *SubscriptionRepositoryImpl) GetSubscriptionMetrics(dateFrom, dateTo time.Time) (*SubscriptionMetrics, error) {
	var metrics SubscriptionMetrics

	// Total active subscribers
	if err := r.db.Model(&models.UserSubscription{}).
		Where("status = ? AND created_at BETWEEN ? AND ?",
			models.SubscriptionStatusActive, dateFrom, dateTo).
		Count(&metrics.TotalSubscribers).Error; err != nil {
		return nil, err
	}

	// Calculate total revenue (сумма всех успешных платежей)
	if err := r.db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND created_at BETWEEN ? AND ?",
			models.PaymentStatusPaid, dateFrom, dateTo).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&metrics.TotalRevenue).Error; err != nil {
		return nil, err
	}

	// Calculate MRR (Monthly Recurring Revenue)
	// Упрощенный расчет - среднемесячный доход
	days := dateTo.Sub(dateFrom).Hours() / 24
	if days > 0 {
		metrics.MRR = metrics.TotalRevenue / (days / 30.0)
	}

	// Calculate ARPU (Average Revenue Per User)
	if metrics.TotalSubscribers > 0 {
		metrics.ARPU = metrics.TotalRevenue / float64(metrics.TotalSubscribers)
	}

	// Calculate churn rate (упрощенный расчет)
	var cancelledSubs int64
	if err := r.db.Model(&models.UserSubscription{}).
		Where("status = ? AND created_at BETWEEN ? AND ?",
			models.SubscriptionStatusCancelled, dateFrom, dateTo).
		Count(&cancelledSubs).Error; err != nil {
		return nil, err
	}

	totalSubsInPeriod := metrics.TotalSubscribers + cancelledSubs
	if totalSubsInPeriod > 0 {
		metrics.ChurnRate = (float64(cancelledSubs) / float64(totalSubsInPeriod)) * 100
	}

	return &metrics, nil
}

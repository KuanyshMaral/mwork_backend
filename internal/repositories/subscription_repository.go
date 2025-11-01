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
	ErrSubscriptionNotFound     = errors.New("subscription not found")
	ErrSubscriptionPlanNotFound = errors.New("subscription plan not found")
	ErrPaymentNotFound          = errors.New("payment transaction not found")
	ErrSubscriptionLimit        = errors.New("subscription limit reached")
)

type SubscriptionRepository interface {
	// SubscriptionPlan operations
	CreatePlan(db *gorm.DB, plan *models.SubscriptionPlan) error
	FindPlanByID(db *gorm.DB, id string) (*models.SubscriptionPlan, error)
	FindPlanByName(db *gorm.DB, name string) (*models.SubscriptionPlan, error)
	FindActivePlans(db *gorm.DB) ([]models.SubscriptionPlan, error)
	FindPlansByRole(db *gorm.DB, role models.UserRole) ([]models.SubscriptionPlan, error)
	UpdatePlan(db *gorm.DB, plan *models.SubscriptionPlan) error
	DeletePlan(db *gorm.DB, id string) error

	// UserSubscription operations
	CreateUserSubscription(db *gorm.DB, subscription *models.UserSubscription) error
	FindUserSubscription(db *gorm.DB, userID string) (*models.UserSubscription, error)
	FindUserSubscriptionByInvID(db *gorm.DB, invID string) (*models.UserSubscription, error)
	UpdateUserSubscription(db *gorm.DB, subscription *models.UserSubscription) error
	UpdateSubscriptionStatus(db *gorm.DB, userID string, status models.SubscriptionStatus) error
	CancelUserSubscription(db *gorm.DB, userID string) error
	RenewUserSubscription(db *gorm.DB, userID string, newPlanID string, newEndDate time.Time) error
	IncrementSubscriptionUsage(db *gorm.DB, userID string, feature string) error
	DecrementSubscriptionUsage(db *gorm.DB, userID string, feature string) error
	ResetSubscriptionUsage(db *gorm.DB, userID string) error
	FindExpiringSubscriptions(db *gorm.DB, days int) ([]models.UserSubscription, error)
	FindExpiredSubscriptions(db *gorm.DB) ([]models.UserSubscription, error)

	// PaymentTransaction operations
	CreatePaymentTransaction(db *gorm.DB, payment *models.PaymentTransaction) error
	FindPaymentByID(db *gorm.DB, id string) (*models.PaymentTransaction, error)
	FindPaymentByInvID(db *gorm.DB, invID string) (*models.PaymentTransaction, error)
	FindPaymentsByUser(db *gorm.DB, userID string) ([]models.PaymentTransaction, error)
	UpdatePaymentStatus(db *gorm.DB, invID string, status models.PaymentStatus, paidAt *time.Time) error
	DeletePayment(db *gorm.DB, id string) error

	// Usage and limits operations
	GetUserUsage(db *gorm.DB, userID string) (map[string]int, error)
	GetUserLimits(db *gorm.DB, userID string) (map[string]int, error)
	CanUserPublish(db *gorm.DB, userID string) (bool, error)
	CanUserRespond(db *gorm.DB, userID string) (bool, error)
	GetUserSubscriptionStats(db *gorm.DB, userID string) (*UserSubscriptionStats, error)

	// Admin operations
	GetPlatformSubscriptionStats(db *gorm.DB) (*PlatformSubscriptionStats, error)
	GetRevenueStats(db *gorm.DB, days int) (*RevenueStats, error)
	GetSubscriptionMetrics(db *gorm.DB, dateFrom, dateTo time.Time) (*SubscriptionMetrics, error)
}

type SubscriptionRepositoryImpl struct {
	// ✅ Пусто! db *gorm.DB больше не хранится здесь
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

// ✅ Конструктор не принимает db
func NewSubscriptionRepository() SubscriptionRepository {
	return &SubscriptionRepositoryImpl{}
}

// SubscriptionPlan operations

func (r *SubscriptionRepositoryImpl) CreatePlan(db *gorm.DB, plan *models.SubscriptionPlan) error {
	// ✅ Используем 'db' из параметра
	return db.Create(plan).Error
}

func (r *SubscriptionRepositoryImpl) FindPlanByID(db *gorm.DB, id string) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	// ✅ Используем 'db' из параметра
	err := db.First(&plan, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *SubscriptionRepositoryImpl) FindPlanByName(db *gorm.DB, name string) (*models.SubscriptionPlan, error) {
	var plan models.SubscriptionPlan
	// ✅ Используем 'db' из параметра
	err := db.Where("name = ?", name).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *SubscriptionRepositoryImpl) FindActivePlans(db *gorm.DB) ([]models.SubscriptionPlan, error) {
	var plans []models.SubscriptionPlan
	// ✅ Используем 'db' из параметра
	err := db.Where("is_active = ?", true).Order("price ASC").Find(&plans).Error
	return plans, err
}

func (r *SubscriptionRepositoryImpl) FindPlansByRole(db *gorm.DB, role models.UserRole) ([]models.SubscriptionPlan, error) {
	var plans []models.SubscriptionPlan

	// Определяем префикс плана по роли
	rolePrefix := getPlanPrefixByRole(role)

	// ✅ Используем 'db' из параметра
	err := db.Where("is_active = ? AND name LIKE ?", true, rolePrefix+"%").
		Order("price ASC").Find(&plans).Error
	return plans, err
}

func (r *SubscriptionRepositoryImpl) UpdatePlan(db *gorm.DB, plan *models.SubscriptionPlan) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(plan).Updates(map[string]interface{}{
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

func (r *SubscriptionRepositoryImpl) DeletePlan(db *gorm.DB, id string) error {
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.SubscriptionPlan{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionPlanNotFound
	}
	return nil
}

// UserSubscription operations

func (r *SubscriptionRepositoryImpl) CreateUserSubscription(db *gorm.DB, subscription *models.UserSubscription) error {
	// ✅ Используем 'db' из параметра
	return db.Create(subscription).Error
}

func (r *SubscriptionRepositoryImpl) FindUserSubscription(db *gorm.DB, userID string) (*models.UserSubscription, error) {
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	err := db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepositoryImpl) FindUserSubscriptionByInvID(db *gorm.DB, invID string) (*models.UserSubscription, error) {
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	err := db.Preload("Plan").Where("inv_id = ?", invID).First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepositoryImpl) UpdateUserSubscription(db *gorm.DB, subscription *models.UserSubscription) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(subscription).Updates(map[string]interface{}{
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

func (r *SubscriptionRepositoryImpl) UpdateSubscriptionStatus(db *gorm.DB, userID string, status models.SubscriptionStatus) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
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

func (r *SubscriptionRepositoryImpl) CancelUserSubscription(db *gorm.DB, userID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
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

func (r *SubscriptionRepositoryImpl) RenewUserSubscription(db *gorm.DB, userID string, newPlanID string, newEndDate time.Time) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	if err := db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
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

	// ✅ Используем 'db' из параметра
	if err := db.Model(&subscription).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}

func (r *SubscriptionRepositoryImpl) IncrementSubscriptionUsage(db *gorm.DB, userID string, feature string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	// ❗️ Мы должны использовать Preload("Plan"), чтобы canUseFeature работал
	if err := db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		return ErrSubscriptionNotFound
	}

	// Проверяем лимиты
	// ✅ canUseFeature теперь безопасно вызывать
	if !r.canUseFeature(&subscription, feature) {
		return ErrSubscriptionLimit
	}

	// Обновляем usage
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
	// ✅ Используем 'db' из параметра
	return db.Save(&subscription).Error
}

func (r *SubscriptionRepositoryImpl) DecrementSubscriptionUsage(db *gorm.DB, userID string, feature string) error {
	// ✅ Вложенная транзакция удалена. Используем 'db' из параметра.
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	if err := db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
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
	// ✅ Используем 'db' из параметра
	return db.Save(&subscription).Error
}

func (r *SubscriptionRepositoryImpl) ResetSubscriptionUsage(db *gorm.DB, userID string) error {
	// ✅ Используем 'db' из параметра
	result := db.Model(&models.UserSubscription{}).Where("user_id = ?", userID).Update("current_usage",
		datatypes.JSON(`{"publications": 0, "responses": 0, "messages": 0, "promotions": 0}`))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) FindExpiringSubscriptions(db *gorm.DB, days int) ([]models.UserSubscription, error) {
	var subscriptions []models.UserSubscription
	expiryDate := time.Now().AddDate(0, 0, days)

	// ✅ Используем 'db' из параметра
	err := db.Preload("Plan").Where("status = ? AND end_date <= ? AND end_date > ?",
		models.SubscriptionStatusActive, expiryDate, time.Now()).
		Order("end_date ASC").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *SubscriptionRepositoryImpl) FindExpiredSubscriptions(db *gorm.DB) ([]models.UserSubscription, error) {
	var subscriptions []models.UserSubscription

	// ✅ Используем 'db' из параметра
	err := db.Preload("Plan").Where("status = ? AND end_date < ?",
		models.SubscriptionStatusActive, time.Now()).
		Order("end_date ASC").
		Find(&subscriptions).Error
	return subscriptions, err
}

// PaymentTransaction operations

func (r *SubscriptionRepositoryImpl) CreatePaymentTransaction(db *gorm.DB, payment *models.PaymentTransaction) error {
	// ✅ Используем 'db' из параметра
	return db.Create(payment).Error
}

func (r *SubscriptionRepositoryImpl) FindPaymentByID(db *gorm.DB, id string) (*models.PaymentTransaction, error) {
	var payment models.PaymentTransaction
	// ✅ Используем 'db' из параметра
	err := db.Preload("Subscription").Preload("Subscription.Plan").
		First(&payment, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *SubscriptionRepositoryImpl) FindPaymentByInvID(db *gorm.DB, invID string) (*models.PaymentTransaction, error) {
	var payment models.PaymentTransaction
	// ✅ Используем 'db' из параметра
	err := db.Preload("Subscription").Preload("Subscription.Plan").
		Where("inv_id = ?", invID).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *SubscriptionRepositoryImpl) FindPaymentsByUser(db *gorm.DB, userID string) ([]models.PaymentTransaction, error) {
	var payments []models.PaymentTransaction
	// ✅ Используем 'db' из параметра
	err := db.Preload("Subscription").Preload("Subscription.Plan").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&payments).Error
	return payments, err
}

func (r *SubscriptionRepositoryImpl) UpdatePaymentStatus(db *gorm.DB, invID string, status models.PaymentStatus, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if paidAt != nil {
		updates["paid_at"] = paidAt
	}

	// ✅ Используем 'db' из параметра
	result := db.Model(&models.PaymentTransaction{}).Where("inv_id = ?", invID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *SubscriptionRepositoryImpl) DeletePayment(db *gorm.DB, id string) error {
	// ✅ Используем 'db' из параметра
	result := db.Where("id = ?", id).Delete(&models.PaymentTransaction{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

// Usage and limits operations

func (r *SubscriptionRepositoryImpl) GetUserUsage(db *gorm.DB, userID string) (map[string]int, error) {
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	err := db.Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	var usage map[string]int
	if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage: %w", err)
	}

	return usage, nil
}

func (r *SubscriptionRepositoryImpl) GetUserLimits(db *gorm.DB, userID string) (map[string]int, error) {
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	err := db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	// ❗️ ИСПРАВЛЕНИЕ: Добавляем проверку, что Plan был загружен
	if subscription.Plan.ID == "" {
		return nil, ErrSubscriptionPlanNotFound
	}

	var limits map[string]int
	if err := json.Unmarshal(subscription.Plan.Limits, &limits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
	}

	return limits, nil
}

func (r *SubscriptionRepositoryImpl) CanUserPublish(db *gorm.DB, userID string) (bool, error) {
	// ✅ 'db' пробрасывается в хелпер
	return r.canUseFeatureByUserID(db, userID, "publications")
}

func (r *SubscriptionRepositoryImpl) CanUserRespond(db *gorm.DB, userID string) (bool, error) {
	// ✅ 'db' пробрасывается в хелпер
	return r.canUseFeatureByUserID(db, userID, "responses")
}

func (r *SubscriptionRepositoryImpl) GetUserSubscriptionStats(db *gorm.DB, userID string) (*UserSubscriptionStats, error) {
	var subscription models.UserSubscription
	// ✅ Используем 'db' из параметра
	err := db.Preload("Plan").Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	// ❗️ ИСПРАВЛЕНИЕ: Добавляем проверку, что Plan был загружен
	// Если план не загружен (например, PlanID = NULL), мы не можем продолжить
	if subscription.Plan.ID == "" {
		return nil, ErrSubscriptionPlanNotFound
	}

	// Получаем usage
	var usage Usage
	if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage: %w", err)
	}

	// Получаем limits
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

func (r *SubscriptionRepositoryImpl) GetPlatformSubscriptionStats(db *gorm.DB) (*PlatformSubscriptionStats, error) {
	var stats PlatformSubscriptionStats

	// Total subscriptions
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.UserSubscription{}).Count(&stats.TotalSubscriptions).Error; err != nil {
		return nil, err
	}

	// Active subscriptions
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.UserSubscription{}).Where("status = ?", models.SubscriptionStatusActive).
		Count(&stats.ActiveSubscriptions).Error; err != nil {
		return nil, err
	}

	// Expired subscriptions
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.UserSubscription{}).Where("status = ?", models.SubscriptionStatusExpired).
		Count(&stats.ExpiredSubscriptions).Error; err != nil {
		return nil, err
	}

	// Canceled subscriptions
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.UserSubscription{}).Where("status = ?", models.SubscriptionStatusCancelled).
		Count(&stats.CanceledSubscriptions).Error; err != nil {
		return nil, err
	}

	// Total revenue
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).Where("status = ?", models.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue).Error; err != nil {
		return nil, err
	}

	// Monthly recurring revenue (last 30 days)
	monthAgo := time.Now().AddDate(0, -1, 0)
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).
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

	// ✅ Используем 'db' из параметра
	err := db.Model(&models.UserSubscription{}).
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

func (r *SubscriptionRepositoryImpl) GetRevenueStats(db *gorm.DB, days int) (*RevenueStats, error) {
	var stats RevenueStats
	now := time.Now()

	// Total revenue
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).Where("status = ?", models.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TotalRevenue).Error; err != nil {
		return nil, err
	}

	// Today revenue
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND paid_at >= ?", models.PaymentStatusPaid, todayStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TodayRevenue).Error; err != nil {
		return nil, err
	}

	// This week revenue
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()))
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND paid_at >= ?", models.PaymentStatusPaid, weekStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.ThisWeekRevenue).Error; err != nil {
		return nil, err
	}

	// This month revenue
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).
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

	// ✅ Используем 'db' из параметра
	err := db.Model(&models.PaymentTransaction{}).
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
	// ✅ Этот хелпер не использует db, он безопасен
	if subscription.Status != models.SubscriptionStatusActive {
		return false
	}

	var usage map[string]int
	var limits map[string]int

	if err := json.Unmarshal(subscription.CurrentUsage, &usage); err != nil {
		return false
	}

	// ❗️ ИСПРАВЛЕНИЕ: Проверяем, что Plan был загружен, по ID (т.к. Plan - это struct)
	if subscription.Plan.ID == "" {
		return false // План не был загружен, безопасный выход
	}
	if err := json.Unmarshal(subscription.Plan.Limits, &limits); err != nil {
		return false
	}

	return usage[feature] < limits[feature]
}

// ✅ Хелпер теперь принимает 'db'
func (r *SubscriptionRepositoryImpl) canUseFeatureByUserID(db *gorm.DB, userID string, feature string) (bool, error) {
	// ✅ 'db' пробрасывается
	// ❗️ Мы должны использовать Preload("Plan"), чтобы canUseFeature работал
	subscription, err := r.FindUserSubscription(db, userID)
	if err != nil {
		return false, err
	}

	// ✅ canUseFeature теперь безопасен
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

func (r *SubscriptionRepositoryImpl) GetSubscriptionMetrics(db *gorm.DB, dateFrom, dateTo time.Time) (*SubscriptionMetrics, error) {
	var metrics SubscriptionMetrics

	// Total active subscribers
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.UserSubscription{}).
		Where("status = ? AND created_at BETWEEN ? AND ?",
			models.SubscriptionStatusActive, dateFrom, dateTo).
		Count(&metrics.TotalSubscribers).Error; err != nil {
		return nil, err
	}

	// Calculate total revenue
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.PaymentTransaction{}).
		Where("status = ? AND created_at BETWEEN ? AND ?",
			models.PaymentStatusPaid, dateFrom, dateTo).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&metrics.TotalRevenue).Error; err != nil {
		return nil, err
	}

	// Calculate MRR
	days := dateTo.Sub(dateFrom).Hours() / 24
	if days > 0 {
		metrics.MRR = metrics.TotalRevenue / (days / 30.0)
	}

	// Calculate ARPU
	if metrics.TotalSubscribers > 0 {
		metrics.ARPU = metrics.TotalRevenue / float64(metrics.TotalSubscribers)
	}

	// Calculate churn rate
	var cancelledSubs int64
	// ✅ Используем 'db' из параметра
	if err := db.Model(&models.UserSubscription{}).
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

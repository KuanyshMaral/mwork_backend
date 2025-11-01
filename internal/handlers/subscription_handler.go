package handlers

import (
	"net/http"
	// "strconv" // <-- Больше не нужен
	"mwork_backend/pkg/apperrors" // <-- Добавлен импорт

	"mwork_backend/internal/middleware" // <-- Все еще нужен для RegisterRoutes
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	*BaseHandler        // <-- 1. Встраиваем BaseHandler
	subscriptionService services.SubscriptionService
}

// 2. Обновляем конструктор
func NewSubscriptionHandler(base *BaseHandler, subscriptionService services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		BaseHandler:         base, // <-- 3. Сохраняем его
		subscriptionService: subscriptionService,
	}
}

// RegisterRoutes не требует изменений
func (h *SubscriptionHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes - Plan information
	plans := r.Group("/plans")
	{
		plans.GET("", h.GetPlans)
		plans.GET("/:planId", h.GetPlan)
	}

	// Protected routes - User subscription operations
	subscriptions := r.Group("/subscriptions")
	subscriptions.Use(middleware.AuthMiddleware())
	{
		subscriptions.GET("/my", h.GetUserSubscription)
		subscriptions.GET("/my/stats", h.GetUserSubscriptionStats)
		subscriptions.POST("/subscribe", h.CreateSubscription)
		subscriptions.PUT("/cancel", h.CancelSubscription)
		subscriptions.PUT("/renew", h.RenewSubscription)
		subscriptions.GET("/check-limit", h.CheckSubscriptionLimit)
		subscriptions.POST("/increment-usage", h.IncrementUsage)
		subscriptions.PUT("/reset-usage", h.ResetUsage)
	}

	// Protected routes - Payment operations
	payments := r.Group("/payments")
	payments.Use(middleware.AuthMiddleware())
	{
		payments.POST("/create", h.CreatePayment)
		payments.GET("/history", h.GetPaymentHistory)
		payments.GET("/:paymentId/status", h.GetPaymentStatus)
	}

	// Robokassa integration routes
	robokassa := r.Group("/robokassa")
	{
		robokassa.POST("/init", middleware.AuthMiddleware(), h.InitRobokassaPayment)
		robokassa.POST("/callback", h.ProcessRobokassaCallback) // No auth - external callback
		robokassa.GET("/check/:paymentId", middleware.AuthMiddleware(), h.CheckRobokassaPayment)
	}

	// Admin routes - Plan management
	adminPlans := r.Group("/admin/plans")
	adminPlans.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		adminPlans.POST("", h.CreatePlan)
		adminPlans.PUT("/:planId", h.UpdatePlan)
		adminPlans.DELETE("/:planId", h.DeletePlan)
	}

	// Admin routes - Subscription management
	adminSubscriptions := r.Group("/admin/subscriptions")
	adminSubscriptions.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		adminSubscriptions.GET("/stats/platform", h.GetPlatformSubscriptionStats)
		adminSubscriptions.GET("/stats/revenue", h.GetRevenueStats)
		adminSubscriptions.GET("/expiring", h.GetExpiringSubscriptions)
		adminSubscriptions.GET("/expired", h.GetExpiredSubscriptions)
		adminSubscriptions.POST("/process-expired", h.ProcessExpiredSubscriptions)
	}
}

// --- Plan handlers ---

func (h *SubscriptionHandler) GetPlans(c *gin.Context) {
	roleParam := c.Query("role")
	role := models.UserRole(roleParam)

	if role == "" {
		role = models.UserRoleModel // Default to model plans
	}

	// ✅ DB: Используем h.GetDB(c)
	plans, err := h.subscriptionService.GetPlans(h.GetDB(c), role)
	if err != nil {
		h.HandleServiceError(c, err) // <-- 4. Используем HandleServiceError
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plans": plans,
		"total": len(plans),
	})
}

func (h *SubscriptionHandler) GetPlan(c *gin.Context) {
	planID := c.Param("planId")

	// ✅ DB: Используем h.GetDB(c)
	plan, err := h.subscriptionService.GetPlan(h.GetDB(c), planID)
	if err != nil {
		h.HandleServiceError(c, err) // <-- 4. Используем HandleServiceError
		return
	}

	c.JSON(http.StatusOK, plan)
}

func (h *SubscriptionHandler) CreatePlan(c *gin.Context) {
	// 5. Используем GetAndAuthorizeUserID
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req models.CreatePlanRequest
	// 6. Используем BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.CreatePlan(h.GetDB(c), adminID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Plan created successfully"})
}

func (h *SubscriptionHandler) UpdatePlan(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	planID := c.Param("planId")

	var req models.UpdatePlanRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.UpdatePlan(h.GetDB(c), adminID, planID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plan updated successfully"})
}

func (h *SubscriptionHandler) DeletePlan(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	planID := c.Param("planId")

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.DeletePlan(h.GetDB(c), adminID, planID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plan deleted successfully"})
}

// --- User subscription handlers ---

func (h *SubscriptionHandler) GetUserSubscription(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	subscription, err := h.subscriptionService.GetUserSubscription(h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, subscription)
}

func (h *SubscriptionHandler) GetUserSubscriptionStats(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.subscriptionService.GetUserSubscriptionStats(h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	subscription, err := h.subscriptionService.CreateSubscription(h.GetDB(c), userID, req.PlanID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.CancelSubscription(h.GetDB(c), userID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled successfully"})
}

func (h *SubscriptionHandler) RenewSubscription(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.RenewSubscription(h.GetDB(c), userID, req.PlanID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription renewed successfully"})
}

func (h *SubscriptionHandler) CheckSubscriptionLimit(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	feature := c.Query("feature")

	if feature == "" {
		apperrors.HandleError(c, apperrors.NewBadRequestError("feature parameter is required"))
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	canUse, err := h.subscriptionService.CheckSubscriptionLimit(h.GetDB(c), userID, feature)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"can_use": canUse,
		"feature": feature,
	})
}

func (h *SubscriptionHandler) IncrementUsage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		Feature string `json:"feature" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.IncrementUsage(h.GetDB(c), userID, req.Feature); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usage incremented successfully"})
}

func (h *SubscriptionHandler) ResetUsage(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.ResetUsage(h.GetDB(c), userID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usage reset successfully"})
}

// --- Payment handlers ---

func (h *SubscriptionHandler) CreatePayment(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	payment, err := h.subscriptionService.CreatePayment(h.GetDB(c), userID, req.PlanID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *SubscriptionHandler) GetPaymentHistory(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	payments, err := h.subscriptionService.GetPaymentHistory(h.GetDB(c), userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"total":    len(payments),
	})
}

func (h *SubscriptionHandler) GetPaymentStatus(c *gin.Context) {
	// Проверяем, что пользователь авторизован,
	// даже если userID не передается в сервис
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	paymentID := c.Param("paymentId")

	// ✅ DB: Используем h.GetDB(c)
	payment, err := h.subscriptionService.GetPaymentStatus(h.GetDB(c), paymentID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, payment)
}

// --- Robokassa handlers ---

func (h *SubscriptionHandler) InitRobokassaPayment(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	response, err := h.subscriptionService.InitRobokassaPayment(h.GetDB(c), userID, req.PlanID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SubscriptionHandler) ProcessRobokassaCallback(c *gin.Context) {
	var data models.RobokassaCallbackData
	// BindAndValidate_JSON использует c.ShouldBind(),
	// который автоматически определяет JSON или Form,
	// поэтому двойная проверка не нужна.
	if !h.BindAndValidate_JSON(c, &data) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.ProcessRobokassaCallback(h.GetDB(c), &data); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment processed successfully"})
}

func (h *SubscriptionHandler) CheckRobokassaPayment(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	paymentID := c.Param("paymentId")

	// ✅ DB: Используем h.GetDB(c)
	response, err := h.subscriptionService.CheckRobokassaPayment(h.GetDB(c), paymentID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// --- Admin handlers ---

func (h *SubscriptionHandler) GetPlatformSubscriptionStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.subscriptionService.GetPlatformSubscriptionStats(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) GetRevenueStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// 7. Используем ParseQueryInt
	days := ParseQueryInt(c, "days", 30)
	if days <= 0 || days > 365 {
		days = 30
	}

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.subscriptionService.GetRevenueStats(h.GetDB(c), days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) GetExpiringSubscriptions(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	days := ParseQueryInt(c, "days", 7)
	if days <= 0 || days > 90 {
		days = 7
	}

	// ✅ DB: Используем h.GetDB(c)
	subscriptions, err := h.subscriptionService.GetExpiringSubscriptions(h.GetDB(c), days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
		"total":         len(subscriptions),
		"expiring_in":   days,
	})
}

func (h *SubscriptionHandler) GetExpiredSubscriptions(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	subscriptions, err := h.subscriptionService.GetExpiredSubscriptions(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
		"total":         len(subscriptions),
	})
}

func (h *SubscriptionHandler) ProcessExpiredSubscriptions(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.subscriptionService.ProcessExpiredSubscriptions(h.GetDB(c)); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired subscriptions processed successfully"})
}

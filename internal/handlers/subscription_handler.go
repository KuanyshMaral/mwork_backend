package handlers

import (
	"net/http"
	"strconv"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	subscriptionService services.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

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

// Plan handlers

func (h *SubscriptionHandler) GetPlans(c *gin.Context) {
	roleParam := c.Query("role")
	role := models.UserRole(roleParam)

	if role == "" {
		role = models.UserRoleModel // Default to model plans
	}

	plans, err := h.subscriptionService.GetPlans(role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plans": plans,
		"total": len(plans),
	})
}

func (h *SubscriptionHandler) GetPlan(c *gin.Context) {
	planID := c.Param("planId")

	plan, err := h.subscriptionService.GetPlan(planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

func (h *SubscriptionHandler) CreatePlan(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	var req models.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.subscriptionService.CreatePlan(adminID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Plan created successfully"})
}

func (h *SubscriptionHandler) UpdatePlan(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	planID := c.Param("planId")

	var req models.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.subscriptionService.UpdatePlan(adminID, planID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "plan not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plan updated successfully"})
}

func (h *SubscriptionHandler) DeletePlan(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	planID := c.Param("planId")

	if err := h.subscriptionService.DeletePlan(adminID, planID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "plan not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plan deleted successfully"})
}

// User subscription handlers

func (h *SubscriptionHandler) GetUserSubscription(c *gin.Context) {
	userID := middleware.GetUserID(c)

	subscription, err := h.subscriptionService.GetUserSubscription(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

func (h *SubscriptionHandler) GetUserSubscriptionStats(c *gin.Context) {
	userID := middleware.GetUserID(c)

	stats, err := h.subscriptionService.GetUserSubscriptionStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	subscription, err := h.subscriptionService.CreateSubscription(userID, req.PlanID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "plan not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.subscriptionService.CancelSubscription(userID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "subscription already cancelled" {
			statusCode = http.StatusBadRequest
		} else if err.Error() == "subscription not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled successfully"})
}

func (h *SubscriptionHandler) RenewSubscription(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.subscriptionService.RenewSubscription(userID, req.PlanID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "plan not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription renewed successfully"})
}

func (h *SubscriptionHandler) CheckSubscriptionLimit(c *gin.Context) {
	userID := middleware.GetUserID(c)
	feature := c.Query("feature")

	if feature == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feature parameter is required"})
		return
	}

	canUse, err := h.subscriptionService.CheckSubscriptionLimit(userID, feature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"can_use": canUse,
		"feature": feature,
	})
}

func (h *SubscriptionHandler) IncrementUsage(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Feature string `json:"feature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.subscriptionService.IncrementUsage(userID, req.Feature); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usage incremented successfully"})
}

func (h *SubscriptionHandler) ResetUsage(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.subscriptionService.ResetUsage(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usage reset successfully"})
}

// Payment handlers

func (h *SubscriptionHandler) CreatePayment(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	payment, err := h.subscriptionService.CreatePayment(userID, req.PlanID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "plan not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *SubscriptionHandler) GetPaymentHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	payments, err := h.subscriptionService.GetPaymentHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"total":    len(payments),
	})
}

func (h *SubscriptionHandler) GetPaymentStatus(c *gin.Context) {
	paymentID := c.Param("paymentId")

	payment, err := h.subscriptionService.GetPaymentStatus(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// Robokassa handlers

func (h *SubscriptionHandler) InitRobokassaPayment(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	response, err := h.subscriptionService.InitRobokassaPayment(userID, req.PlanID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "plan not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SubscriptionHandler) ProcessRobokassaCallback(c *gin.Context) {
	var data models.RobokassaCallbackData
	if err := c.ShouldBindJSON(&data); err != nil {
		// Try form binding for URL-encoded data
		if err := c.ShouldBind(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback data"})
			return
		}
	}

	if err := h.subscriptionService.ProcessRobokassaCallback(&data); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "invalid signature" || err.Error() == "invalid payment amount" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment processed successfully"})
}

func (h *SubscriptionHandler) CheckRobokassaPayment(c *gin.Context) {
	paymentID := c.Param("paymentId")

	response, err := h.subscriptionService.CheckRobokassaPayment(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Admin handlers

func (h *SubscriptionHandler) GetPlatformSubscriptionStats(c *gin.Context) {
	stats, err := h.subscriptionService.GetPlatformSubscriptionStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) GetRevenueStats(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsed, err := strconv.Atoi(daysParam); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	stats, err := h.subscriptionService.GetRevenueStats(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) GetExpiringSubscriptions(c *gin.Context) {
	days := 7
	if daysParam := c.Query("days"); daysParam != "" {
		if parsed, err := strconv.Atoi(daysParam); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	subscriptions, err := h.subscriptionService.GetExpiringSubscriptions(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
		"total":         len(subscriptions),
		"expiring_in":   days,
	})
}

func (h *SubscriptionHandler) GetExpiredSubscriptions(c *gin.Context) {
	subscriptions, err := h.subscriptionService.GetExpiredSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
		"total":         len(subscriptions),
	})
}

func (h *SubscriptionHandler) ProcessExpiredSubscriptions(c *gin.Context) {
	if err := h.subscriptionService.ProcessExpiredSubscriptions(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired subscriptions processed successfully"})
}

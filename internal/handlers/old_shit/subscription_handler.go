package old_shit

import (
	"fmt"
	"mwork_backend/internal/utils"
	"net/http"
	"time"

	"mwork_backend/internal/dto"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services/subscription"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	planService      *subscription.PlanService
	userSubService   *subscription.UserSubscriptionService
	robokassaService *subscription.RobokassaService
}

func NewSubscriptionHandler(
	plan *subscription.PlanService,
	userSub *subscription.UserSubscriptionService,
	robokassa *subscription.RobokassaService,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		planService:      plan,
		userSubService:   userSub,
		robokassaService: robokassa,
	}
}

func (h *SubscriptionHandler) GetAllPlans(c *gin.Context) {
	ctx := c.Request.Context()
	plans, err := h.planService.GetAllPlans(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch plans"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *SubscriptionHandler) GetPlansWithStats(c *gin.Context) {
	ctx := c.Request.Context()
	plans, err := h.planService.GetPlansWithStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch plans with stats"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *SubscriptionHandler) GetRevenueByPeriod(c *gin.Context) {
	var query struct {
		Start string `form:"start"`
		End   string `form:"end"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date query"})
		return
	}

	start, err1 := time.Parse(time.RFC3339, query.Start)
	end, err2 := time.Parse(time.RFC3339, query.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	ctx := c.Request.Context()
	revenue, err := h.planService.GetRevenueByPeriod(ctx, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch revenue data"})
		return
	}
	c.JSON(http.StatusOK, revenue)
}

func (h *SubscriptionHandler) CreatePlan(c *gin.Context) {
	var plan dto.PlanBase
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan data"})
		return
	}
	ctx := c.Request.Context()
	if err := h.planService.CreatePlan(ctx, &plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create plan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "plan created"})
}

func (h *SubscriptionHandler) DeletePlan(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	if err := h.planService.DeletePlan(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plan deleted"})
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var sub models.UserSubscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription data"})
		return
	}
	ctx := c.Request.Context()
	if err := h.userSubService.Create(ctx, &sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "subscription created"})
}

func (h *SubscriptionHandler) CancelMySubscription(c *gin.Context) {
	userID := c.GetString("userID") // JWT middleware должен вставлять userID в контекст
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ctx := c.Request.Context()
	if err := h.userSubService.CancelSubscription(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription canceled"})
}

func (h *SubscriptionHandler) ForceCancelSubscription(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	if err := h.userSubService.ForceCancelSubscription(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to force cancel subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription forcibly canceled"})
}

func (h *SubscriptionHandler) ForceExtendSubscription(c *gin.Context) {
	var req struct {
		NewEndDate string `json:"new_end_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
		return
	}
	newEnd, err := time.Parse(time.RFC3339, req.NewEndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}
	id := c.Param("id")
	ctx := c.Request.Context()
	if err := h.userSubService.ForceExtendSubscription(ctx, id, newEnd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to extend subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription extended"})
}

func (h *SubscriptionHandler) GetUserSubscriptions(c *gin.Context) {
	var query struct {
		Status string `form:"status"`
	}
	_ = c.ShouldBindQuery(&query)
	ctx := c.Request.Context()
	subs, err := h.userSubService.GetAllUserSubscriptions(ctx, &query.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch subscriptions"})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *SubscriptionHandler) GetSubscriptionStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.userSubService.GetStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *SubscriptionHandler) InitiatePayment(c *gin.Context) {
	var req struct {
		PlanID string `json:"plan_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Получаем план
	ctx := c.Request.Context()
	plan, err := h.planService.GetByID(ctx, req.PlanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription plan not found"})
		return
	}

	orderID := utils.GenerateOrderID()
	amount := plan.Price
	description := fmt.Sprintf("Оплата подписки: %s", plan.Name)
	email := "" // Можно доработать позже

	paymentURL, err := h.robokassaService.GeneratePaymentURL(orderID, amount, description, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_url": paymentURL})
}

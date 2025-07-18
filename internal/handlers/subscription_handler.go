package handlers

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

// -------------------- Планы --------------------

// GetAllPlans godoc
// @Summary Получить все подписочные планы
// @Tags Subscription
// @Produce json
// @Success 200 {array} dto.PlanBase
// @Router /subscriptions/plans [get]
func (h *SubscriptionHandler) GetAllPlans(c *gin.Context) {
	ctx := c.Request.Context()
	plans, err := h.planService.GetAllPlans(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch plans"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

// GetPlansWithStats godoc
// @Summary Получить планы с числом пользователей на каждом
// @Tags Subscription
// @Produce json
// @Success 200 {array} dto.PlanWithStats
// @Router /subscriptions/plans/stats [get]
func (h *SubscriptionHandler) GetPlansWithStats(c *gin.Context) {
	ctx := c.Request.Context()
	plans, err := h.planService.GetPlansWithStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch plans with stats"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

// GetRevenueByPeriod godoc
// @Summary Получить выручку по каждому плану за период
// @Tags Subscription
// @Produce json
// @Param start query string true "Начало периода (RFC3339)"
// @Param end query string true "Конец периода (RFC3339)"
// @Success 200 {array} dto.PlanRevenue
// @Failure 400 {object} models.ErrorResponse
// @Router /subscriptions/plans/revenue [get]
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

// CreatePlan godoc
// @Summary Создать новый подписочный план
// @Tags Subscription
// @Accept json
// @Produce json
// @Param plan body dto.PlanBase true "Данные плана"
// @Success 201 {object} models.GenericResponse
// @Failure 400,500 {object} models.GenericResponse
// @Router /subscriptions/plans [post]
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

// DeletePlan godoc
// @Summary Удалить подписочный план по ID
// @Tags Subscription
// @Produce json
// @Param id path string true "ID плана"
// @Success 200 {object} models.GenericResponse
// @Failure 400,500 {object} models.GenericResponse
// @Router /subscriptions/plans/{id} [delete]
func (h *SubscriptionHandler) DeletePlan(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	if err := h.planService.DeletePlan(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plan deleted"})
}

// -------------------- Подписки --------------------

// CreateSubscription godoc
// @Summary Создать подписку на план для пользователя
// @Tags Subscription
// @Accept json
// @Produce json
// @Param subscription body dto.CreateSubscriptionRequest true "Запрос на создание подписки"
// @Success 201 {object} models.GenericResponse
// @Failure 400,500 {object} models.GenericResponse
// @Router /subscriptions [post]
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

// CancelMySubscription godoc
// @Summary Отменить свою подписку
// @Tags Subscription
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.GenericResponse
// @Failure 401,500 {object} models.ErrorResponse
// @Router /subscriptions/my/cancel [post]
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

// ForceCancelSubscription godoc
// @Summary Принудительно отменить подписку пользователя (админ)
// @Tags Subscription
// @Produce json
// @Param id path string true "ID подписки"
// @Success 200 {object} models.GenericResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /subscriptions/{id}/force-cancel [post]
func (h *SubscriptionHandler) ForceCancelSubscription(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	if err := h.userSubService.ForceCancelSubscription(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to force cancel subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription forcibly canceled"})
}

// ForceExtendSubscription godoc
// @Summary Принудительно продлить подписку до новой даты
// @Tags Subscription
// @Accept json
// @Produce json
// @Param id path string true "ID подписки"
// @Param request body dto.ForceExtendSubscriptionRequest true "Новая дата окончания"
// @Success 200 {object} models.GenericResponse
// @Failure 400,500 {object} models.ErrorResponse
// @Router /subscriptions/{id}/force-extend [post]
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

// GetUserSubscriptions godoc
// @Summary Получить список подписок пользователей
// @Tags Subscription
// @Produce json
// @Param status query string false "Фильтр по статусу (active, canceled)"
// @Success 200 {array} dto.UserSubscriptionDTO
// @Failure 500 {object} models.ErrorResponse
// @Router /subscriptions/users [get]
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

// GetSubscriptionStats godoc
// @Summary Получить общую статистику подписок
// @Tags Subscription
// @Produce json
// @Success 200 {object} dto.PlanWithStats
// @Failure 500 {object} models.ErrorResponse
// @Router /subscriptions/stats [get]
func (h *SubscriptionHandler) GetSubscriptionStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.userSubService.GetStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// InitiatePayment godoc
// @Summary Инициировать оплату через Robokassa
// @Tags Subscription
// @Accept json
// @Produce json
// @Param request body dto.InitiatePaymentRequest true "ID плана"
// @Success 200 {object} map[string]string
// @Failure 400,401,404,500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /subscriptions/pay [post]
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

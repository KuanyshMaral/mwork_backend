package handlers

import (
	"github.com/gin-gonic/gin"
	"mwork_front_fn/backend/services"
	"net/http"
)

type SubscriptionHandler struct {
	service *services.SubscriptionService
}

func NewSubscriptionHandler(service *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service}
}

// Получить все планы
// GetPlans godoc
// @Summary Получить список подписочных планов
// @Description Возвращает все доступные планы подписки
// @Tags subscriptions
// @Accept json
// @Produce json
// @Success 200 {array} dto.SubscriptionPlanResponse
// @Failure 500 {object} models.ErrorResponse "Ошибка при получении планов"
// @Router /subscriptions/plans [get]
func (h *SubscriptionHandler) GetPlans(c *gin.Context) {
	plans, err := h.service.GetPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get plans"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

// Получить текущую подписку пользователя
// GetUserSubscription godoc
// @Summary Получить активную подписку пользователя
// @Description Возвращает текущую активную подписку по ID пользователя
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param userID path string true "ID пользователя"
// @Success 200 {object} dto.UserSubscriptionResponse
// @Failure 400 {object} models.ErrorResponse "userID не указан"
// @Failure 404 {object} models.ErrorResponse "Подписка не найдена"
// @Router /subscriptions/user/{userID} [get]
func (h *SubscriptionHandler) GetUserSubscription(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID required"})
		return
	}

	sub, err := h.service.GetUserSubscription(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// Оформить подписку
type SubscribeRequest struct {
	UserID    string `json:"user_id"`
	PlanID    string `json:"plan_id"`
	AutoRenew bool   `json:"auto_renew"`
}

// CreateSubscription godoc
// @Summary Оформить подписку
// @Description Создаёт новую подписку на основе переданных данных
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body handlers.SubscribeRequest true "Данные подписки"
// @Success 201 {object} dto.UserSubscriptionResponse
// @Failure 400 {object} models.ErrorResponse "Некорректный ввод"
// @Failure 500 {object} models.ErrorResponse "Ошибка при создании подписки"
// @Router /subscriptions [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	sub, err := h.service.CreateSubscription(c.Request.Context(), req.UserID, req.PlanID, req.AutoRenew)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// Проверить лимит использования
// CheckUsageLimit godoc
// @Summary Проверить лимит использования
// @Description Проверяет, можно ли выполнить указанное действие в рамках текущей подписки
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_id query string true "ID пользователя"
// @Param action query string true "Тип действия (например: upload, apply)"
// @Success 200 {object} map[string]interface{} "allowed: bool, remaining: int"
// @Failure 400 {object} models.ErrorResponse "Отсутствует user_id или action"
// @Failure 500 {object} models.ErrorResponse "Ошибка при проверке лимита"
// @Router /subscriptions/usage-limit [get]
func (h *SubscriptionHandler) CheckUsageLimit(c *gin.Context) {
	userID := c.Query("user_id")
	action := c.Query("action")
	if userID == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and action are required"})
		return
	}

	ok, remaining, err := h.service.CheckUsageLimit(c.Request.Context(), userID, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": ok, "remaining": remaining})
}

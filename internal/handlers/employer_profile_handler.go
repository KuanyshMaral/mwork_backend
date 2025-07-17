package handlers

import (
	"github.com/gin-gonic/gin"
	"mwork_front_fn/internal/models"
	"mwork_front_fn/internal/services"
	"net/http"
)

type EmployerProfileHandler struct {
	service *services.EmployerProfileService
}

func NewEmployerProfileHandler(service *services.EmployerProfileService) *EmployerProfileHandler {
	return &EmployerProfileHandler{service: service}
}

// POST /employer-profiles
// CreateProfile godoc
// @Summary Создать профиль работодателя
// @Description Создаёт новый профиль работодателя на основе переданных данных
// @Tags employer-profiles
// @Accept json
// @Produce json
// @Param profile body models.EmployerProfile true "Данные профиля работодателя"
// @Success 201 {object} models.EmployerProfile
// @Failure 400 {object} models.ErrorResponse "Некорректный JSON"
// @Failure 500 {object} models.ErrorResponse "Ошибка при создании профиля"
// @Router /employer-profiles [post]
func (h *EmployerProfileHandler) CreateProfile(c *gin.Context) {
	var profile models.EmployerProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if err := h.service.CreateProfile(c.Request.Context(), &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create employer profile"})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// GET /employer-profiles/:user_id
// GetProfile godoc
// @Summary Получить профиль работодателя по ID пользователя
// @Description Возвращает профиль работодателя, связанный с указанным user_id
// @Tags employer-profiles
// @Accept json
// @Produce json
// @Param user_id path string true "ID пользователя"
// @Success 200 {object} models.EmployerProfile
// @Failure 404 {object} models.ErrorResponse "Профиль не найден"
// @Router /employer-profiles/{user_id} [get]
func (h *EmployerProfileHandler) GetProfile(c *gin.Context) {
	userID := c.Param("user_id")
	profile, err := h.service.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

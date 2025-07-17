package handlers

import (
	"github.com/gin-gonic/gin"
	"mwork_front_fn/internal/models"
	"mwork_front_fn/internal/services"
	"net/http"
)

type CastingHandler struct {
	service *services.CastingService
}

func NewCastingHandler(service *services.CastingService) *CastingHandler {
	return &CastingHandler{service: service}
}

// Создать кастинг
// Create godoc
// @Summary Создать кастинг
// @Description Создаёт новый кастинг по переданным данным
// @Tags castings
// @Accept json
// @Produce json
// @Param casting body models.Casting true "Данные кастинга"
// @Success 201 {object} models.Casting
// @Failure 400 {object} models.ErrorResponse "Неверный JSON"
// @Failure 500 {object} models.ErrorResponse "Ошибка при создании кастинга"
// @Router /castings [post]

func (h *CastingHandler) Create(c *gin.Context) {
	var casting models.Casting
	if err := c.ShouldBindJSON(&casting); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if err := h.service.Create(c.Request.Context(), &casting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create casting"})
		return
	}

	c.JSON(http.StatusCreated, casting)
}

// Получить кастинг по ID
// GetByID godoc
// @Summary Получить кастинг по ID
// @Description Возвращает кастинг по его уникальному идентификатору
// @Tags castings
// @Accept json
// @Produce json
// @Param id path string true "ID кастинга"
// @Success 200 {object} models.Casting
// @Failure 404 {object} models.ErrorResponse "Кастинг не найден"
// @Router /castings/{id} [get]

func (h *CastingHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	casting, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "casting not found"})
		return
	}
	c.JSON(http.StatusOK, casting)
}

// Обновить кастинг
// Update godoc
// @Summary Обновить кастинг
// @Description Обновляет существующий кастинг по ID
// @Tags castings
// @Accept json
// @Produce json
// @Param id path string true "ID кастинга"
// @Param casting body models.Casting true "Обновлённые данные кастинга"
// @Success 200 {object} models.Casting
// @Failure 400 {object} models.ErrorResponse "Некорректный JSON"
// @Failure 500 {object} models.ErrorResponse "Ошибка при обновлении кастинга"
// @Router /castings/{id} [put]

func (h *CastingHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var update models.Casting
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	update.ID = id

	if err := h.service.Update(c.Request.Context(), &update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update casting"})
		return
	}
	c.JSON(http.StatusOK, update)
}

// Удалить кастинг
// Delete godoc
// @Summary Удалить кастинг
// @Description Удаляет кастинг по ID
// @Tags castings
// @Accept json
// @Produce json
// @Param id path string true "ID кастинга"
// @Success 204 "Кастинг удалён"
// @Failure 500 {object} models.ErrorResponse "Ошибка при удалении кастинга"
// @Router /castings/{id} [delete]

func (h *CastingHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete casting"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Список кастингов по работодателю
// ListByEmployer godoc
// @Summary Получить кастинги по работодателю
// @Description Возвращает список кастингов, созданных работодателем
// @Tags castings
// @Accept json
// @Produce json
// @Param employer_id query string true "ID работодателя"
// @Success 200 {array} models.Casting
// @Failure 500 {object} models.ErrorResponse "Ошибка при получении кастингов"
// @Router /castings [get]

func (h *CastingHandler) ListByEmployer(c *gin.Context) {
	employerID := c.Query("employer_id")
	list, err := h.service.ListByEmployer(c.Request.Context(), employerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch castings"})
		return
	}
	c.JSON(http.StatusOK, list)
}

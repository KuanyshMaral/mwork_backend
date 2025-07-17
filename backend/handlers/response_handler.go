package handlers

import (
	"github.com/gin-gonic/gin"
	"mwork_front_fn/backend/models"
	"mwork_front_fn/backend/services"
	"net/http"
)

type ResponseHandler struct {
	service *services.ResponseService
}

func NewResponseHandler(service *services.ResponseService) *ResponseHandler {
	return &ResponseHandler{service: service}
}

// POST /responses
// Create godoc
// @Summary Отклик на кастинг
// @Description Создаёт отклик на кастинг со статусом "pending"
// @Tags responses
// @Accept json
// @Produce json
// @Param response body models.CastingResponse true "Отклик на кастинг"
// @Success 201 {object} models.CastingResponse
// @Failure 400 {object} models.ErrorResponse "Некорректный JSON"
// @Failure 500 {object} models.ErrorResponse "Ошибка при создании отклика"
// @Router /responses [post]
func (h *ResponseHandler) Create(c *gin.Context) {
	var res models.CastingResponse
	if err := c.ShouldBindJSON(&res); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	res.Status = "pending"

	if err := h.service.Create(c.Request.Context(), &res); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create response"})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GET /responses/:id
// GetByID godoc
// @Summary Получить отклик по ID
// @Description Возвращает отклик по ID
// @Tags responses
// @Accept json
// @Produce json
// @Param id path string true "ID отклика"
// @Success 200 {object} models.CastingResponse
// @Failure 404 {object} models.ErrorResponse "Отклик не найден"
// @Router /responses/{id} [get]
func (h *ResponseHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "response not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// GET /responses?casting_id=...
// ListByCasting godoc
// @Summary Получить отклики на кастинг
// @Description Возвращает список откликов по ID кастинга
// @Tags responses
// @Accept json
// @Produce json
// @Param casting_id query string true "ID кастинга"
// @Success 200 {array} models.CastingResponse
// @Failure 500 {object} models.ErrorResponse "Ошибка при получении откликов"
// @Router /responses [get]
func (h *ResponseHandler) ListByCasting(c *gin.Context) {
	castingID := c.Query("casting_id")
	responses, err := h.service.ListByCasting(c.Request.Context(), castingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list responses"})
		return
	}
	c.JSON(http.StatusOK, responses)
}

package handlers

import (
	"mwork_front_fn/backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// GetUser godoc
// @Summary Получить пользователя по ID
// @Description Возвращает пользователя по его ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "ID пользователя"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse "missing user ID"
// @Failure 400 {object} models.ErrorResponse "user not found"
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	user, err := h.service.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// PUT /users/:id
// UpdateUser godoc
// @Summary Обновить пользователя
// @Description Обновляет поля пользователя по его ID (например, статус)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "ID пользователя"
// @Param update body object true "Данные для обновления"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse "invalid input"
// @Failure 400 {object} models.ErrorResponse "user not found"
// @Failure 400 {object} models.ErrorResponse "failed to update user"
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	user, err := h.service.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var update map[string]interface{}
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if status, ok := update["status"].(string); ok {
		user.Status = status
	}

	if err := h.service.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DELETE /users/:id
// DeleteUser godoc
// @Summary Удалить пользователя
// @Description Удаляет пользователя по его ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "ID пользователя"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse "missing user ID"
// @Failure 400 {object} models.ErrorResponse "failed to delete user"
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.Status(http.StatusNoContent)
}

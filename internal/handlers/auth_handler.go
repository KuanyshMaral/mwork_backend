package handlers

import (
	"mwork_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register godoc
// @Summary Регистрация нового пользователя
// @Description Регистрирует нового пользователя по email, паролю и роли
// @Tags auth
// @Accept json
// @Produce json
// @Param input body RegisterInput true "Данные регистрации"
// @Success 201 {object} models.User
// @Failure 400 {object} models.ErrorResponse "Неверный формат ввода или ошибка регистрации"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	user, err := h.auth.Register(c.Request.Context(), input.Email, input.Password, input.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// Login godoc
// @Summary Вход пользователя
// @Description Возвращает access и refresh токены при успешной аутентификации
// @Tags auth
// @Accept json
// @Produce json
// @Param input body LoginInput true "Email и пароль"
// @Success 200 {object} map[string]interface{} "access_token, refresh_token, user"
// @Failure 400 {object} models.ErrorResponse "Неверный формат ввода"
// @Failure 401 {object} models.ErrorResponse "Неверные учётные данные"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	accessToken, refreshToken, user, err := h.auth.Login(c.Request.Context(), input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// RefreshToken godoc
// @Summary Обновление access-токена
// @Description Получить новый access-токен по refresh-токену
// @Tags auth
// @Accept json
// @Produce json
// @Param input body RefreshInput true "Refresh токен"
// @Success 200 {object} map[string]string "access_token"
// @Failure 400 {object} models.ErrorResponse "Неверный формат ввода"
// @Failure 401 {object} models.ErrorResponse "Недействительный токен"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var input RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	newToken, err := h.auth.RefreshService().ValidateAndRefresh(c.Request.Context(), input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": newToken})
}

// Logout godoc
// @Summary Выход из системы
// @Description Удаляет refresh токен (разлогинивает пользователя)
// @Tags auth
// @Accept json
// @Produce json
// @Param input body RefreshInput true "Refresh токен для удаления"
// @Success 200 {object} map[string]string "message"
// @Failure 400 {object} models.ErrorResponse "Неверный формат ввода"
// @Failure 401 {object} models.ErrorResponse "Ошибка при удалении токена"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var input RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	err := h.auth.RefreshService().DeleteByToken(c.Request.Context(), input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logout successful"})
}

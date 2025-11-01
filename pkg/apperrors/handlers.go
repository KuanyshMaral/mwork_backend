package apperrors

import (
	"log"

	"github.com/gin-gonic/gin"
)

// ErrorResponse - стандартный ответ об ошибке
type ErrorResponse struct {
	Error *AppError `json:"error"`
}

// GinErrorHandler - обработчик ошибок для Gin
type GinErrorHandler struct {
	Debug bool
}

// HandleGinError - основная логика обработки ошибок для Gin
func (h *GinErrorHandler) HandleGinError(c *gin.Context, err error) {
	appErr, ok := AsAppError(err)
	if !ok {
		// Если это не AppError, оборачиваем в InternalError
		appErr = InternalError(err)
		if !h.Debug {
			// В продакшене скрываем детали
			appErr.Message = "Internal server error"
			appErr.Details = nil
		}
	}

	// Логирование
	if appErr.HTTPCode >= 500 {
		log.Printf("Server error: %v", appErr.Unwrap())
	}

	// Отправка ответа
	c.JSON(appErr.HTTPCode, ErrorResponse{Error: appErr})
}

// HandleError - быстрая функция-помощник для Gin
// (Вы можете настроить Debug из конфига)
func HandleError(c *gin.Context, err error) {
	handler := &GinErrorHandler{Debug: true} // TODO: В проде установить false
	handler.HandleGinError(c, err)
}

// AsAppError - пытается преобразовать error в *AppError
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// HandleValidationError - специальный обработчик для ошибок валидации Gin
func HandleValidationError(c *gin.Context, err error) {
	// Преобразование ошибок валидации Gin в наш формат
	// (Здесь можно добавить логику парсинга validator.ValidationErrors)
	validationErr := ValidationError(gin.H{"details": err.Error()})
	HandleError(c, validationErr)
}

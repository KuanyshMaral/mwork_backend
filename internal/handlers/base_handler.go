package handlers

import (
	"strconv"
	"time"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/logger"
	"mwork_backend/internal/validator"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 1. Базовая структура обработчика
// ============================================================================

type BaseHandler struct {
	validator *validator.Validator
}

func NewBaseHandler(v *validator.Validator) *BaseHandler {
	return &BaseHandler{
		validator: v,
	}
}

// ============================================================================
// 2. Методы привязки и валидации (с контекстным логгированием)
// ============================================================================

func (h *BaseHandler) BindAndValidate_JSON(c *gin.Context, obj interface{}) bool {
	ctx := c.Request.Context()

	if err := c.ShouldBind(obj); err != nil {
		logger.CtxWithError(ctx, "Failed to bind JSON body", err, "path", c.Request.URL.Path)
		appErrors.HandleError(c, appErrors.NewBadRequestError("Invalid request body: "+err.Error()))
		return false
	}

	if err := h.validator.Validate(obj); err != nil {
		if vErr, ok := err.(*validator.ValidationError); ok {
			logger.CtxWarn(ctx, "Validation failed", "errors", vErr.Errors, "path", c.Request.URL.Path)

			// --- ИСПРАВЛЕНИЕ ЗДЕСЬ ---
			// Было: appErrors.HandleError(c, appErrors.NewValidationError(vErr.Errors))
			// Стало:
			appErrors.HandleError(c, appErrors.ValidationError(vErr.Errors))
			// -------------------------

		} else {
			logger.CtxWithError(ctx, "Internal validator error", err, "path", c.Request.URL.Path)
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return false
	}
	return true
}

func (h *BaseHandler) BindAndValidate_Query(c *gin.Context, obj interface{}) bool {
	ctx := c.Request.Context()

	if err := c.ShouldBindQuery(obj); err != nil {
		logger.CtxWithError(ctx, "Failed to bind query params", err, "path", c.Request.URL.Path)
		appErrors.HandleError(c, appErrors.NewBadRequestError("Invalid query parameters: "+err.Error()))
		return false
	}

	if err := h.validator.Validate(obj); err != nil {
		if vErr, ok := err.(*validator.ValidationError); ok {
			logger.CtxWarn(ctx, "Validation failed (query)", "errors", vErr.Errors, "path", c.Request.URL.Path)

			// --- ИСПРАВЛЕНИЕ ЗДЕСЬ ---
			// Было: appErrors.HandleError(c, appErrors.NewValidationError(vErr.Errors))
			// Стало:
			appErrors.HandleError(c, appErrors.ValidationError(vErr.Errors))
			// -------------------------

		} else {
			logger.CtxWithError(ctx, "Internal validator error (query)", err, "path", c.Request.URL.Path)
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return false
	}
	return true
}

// ============================================================================
// 3. Обработчики ошибок (с контекстным логгированием)
// ============================================================================

func (h *BaseHandler) HandleServiceError(c *gin.Context, err error) {
	ctx := c.Request.Context()

	var appErr *appErrors.AppError
	if appErrors.As(err, &appErr) {
		logger.CtxWarn(ctx, "Service error",
			"error", appErr.Message,
			"details", appErr.Details,
			"path", c.Request.URL.Path,
		)
		appErrors.HandleError(c, appErr)
	} else {
		logger.CtxWithError(ctx, "Internal server error", err, "path", c.Request.URL.Path)
		appErrors.HandleError(c, appErrors.InternalError(err))
	}
}

// ============================================================================
// 4. Вспомогательные функции (с контекстным логгированием)
// ============================================================================

func (h *BaseHandler) GetAndAuthorizeUserID(c *gin.Context) (string, bool) {
	ctx := c.Request.Context()

	userIDVal, exists := c.Get("userID")
	if !exists {
		logger.CtxWarn(ctx, "Unauthorized access: userID not found in context",
			"path", c.Request.URL.Path,
			"ip", c.ClientIP(),
		)
		appErrors.HandleError(c, appErrors.NewUnauthorizedError("User not authenticated"))
		return "", false
	}

	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		logger.CtxWarn(ctx, "Unauthorized access: invalid userID in context",
			"path", c.Request.URL.Path,
			"ip", c.ClientIP(),
		)
		appErrors.HandleError(c, appErrors.NewUnauthorizedError("Invalid user ID in context"))
		return "", false
	}

	return userIDStr, true
}

// ============================================================================
// 5. Функции парсинга (остаются как есть)
// ============================================================================

func ParseQueryInt(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func ParseParamInt(c *gin.Context, key string) (int, error) {
	valueStr := c.Param(key)
	if valueStr == "" {
		return 0, appErrors.NewBadRequestError("Missing required path parameter: " + key)
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, appErrors.NewBadRequestError("Invalid path parameter: " + key + " is not an integer")
	}
	return value, nil
}

func ParsePagination(c *gin.Context) (page int, pageSize int) {
	const defaultPage = 1
	const defaultPageSize = 20
	const maxPageSize = 100

	page = ParseQueryInt(c, "page", defaultPage)
	if page <= 0 {
		page = defaultPage
	}

	pageSize = ParseQueryInt(c, "page_size", defaultPageSize)
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

func ParseQueryDateRange(c *gin.Context, defaultDaysAgo int) (time.Time, time.Time, error) {
	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")

	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -defaultDaysAgo)

	var err error
	if dateFromStr != "" {
		dateFrom, err = time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			return time.Time{}, time.Time{}, appErrors.NewBadRequestError("Invalid date_from format. Use RFC3339 (YYYY-M-DDTHH:MM:SSZ)")
		}
	}

	if dateToStr != "" {
		dateTo, err = time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			return time.Time{}, time.Time{}, appErrors.NewBadRequestError("Invalid date_to format. Use RFC3339 (YYYY-M-DDTHH:MM:SSZ)")
		}
	}

	if dateFrom.After(dateTo) {
		return time.Time{}, time.Time{}, appErrors.NewBadRequestError("date_from cannot be after date_to")
	}

	return dateFrom, dateTo, nil
}

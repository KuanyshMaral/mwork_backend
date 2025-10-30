package handlers

import (
	"net/http"

	"mwork_backend/internal/middleware" // Still needed for RegisterRoutes
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type ResponseHandler struct {
	*BaseHandler    // <-- 1. Embed BaseHandler
	responseService services.ResponseService
}

// 2. Update the constructor
func NewResponseHandler(base *BaseHandler, responseService services.ResponseService) *ResponseHandler {
	return &ResponseHandler{
		BaseHandler:     base, // <-- 3. Assign it
		responseService: responseService,
	}
}

// RegisterRoutes remains unchanged
func (h *ResponseHandler) RegisterRoutes(r *gin.RouterGroup) {
	responses := r.Group("/responses")
	responses.Use(middleware.AuthMiddleware())
	{
		// Model routes
		responses.POST("/castings/:castingId", middleware.RoleMiddleware(models.UserRoleModel), h.CreateResponse)
		responses.GET("/my", middleware.RoleMiddleware(models.UserRoleModel), h.GetMyResponses)
		responses.DELETE("/:responseId", middleware.RoleMiddleware(models.UserRoleModel), h.DeleteResponse)

		// Employer routes
		responses.GET("/castings/:castingId/list", middleware.RoleMiddleware(models.UserRoleEmployer), h.GetCastingResponses)
		responses.PUT("/:responseId/status", middleware.RoleMiddleware(models.UserRoleEmployer), h.UpdateResponseStatus)
		responses.PUT("/:responseId/viewed", middleware.RoleMiddleware(models.UserRoleEmployer), h.MarkResponseAsViewed)
		responses.GET("/castings/:castingId/stats", middleware.RoleMiddleware(models.UserRoleEmployer), h.GetResponseStats)

		// Common routes
		responses.GET("/:responseId", h.GetResponse)
	}
}

// --- Model handlers ---

func (h *ResponseHandler) CreateResponse(c *gin.Context) {
	// 4. Use GetAndAuthorizeUserID
	modelID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	castingID := c.Param("castingId")

	var req dto.CreateResponseRequest
	// 5. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	response, err := h.responseService.CreateResponse(modelID, castingID, &req)
	if err != nil {
		// 6. Use HandleServiceError
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *ResponseHandler) GetMyResponses(c *gin.Context) {
	modelID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	responses, err := h.responseService.GetModelResponses(modelID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"responses": responses,
		"total":     len(responses),
	})
}

func (h *ResponseHandler) DeleteResponse(c *gin.Context) {
	modelID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	responseID := c.Param("responseId")

	if err := h.responseService.DeleteResponse(modelID, responseID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response deleted successfully"})
}

// --- Employer handlers ---

func (h *ResponseHandler) GetCastingResponses(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	castingID := c.Param("castingId")

	responses, err := h.responseService.GetCastingResponses(castingID, employerID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"responses": responses,
		"total":     len(responses),
	})
}

func (h *ResponseHandler) UpdateResponseStatus(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	responseID := c.Param("responseId")

	var req struct {
		Status models.ResponseStatus `json:"status" binding:"required"`
	}
	// Use BindAndValidate_JSON for the anonymous struct
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.responseService.UpdateResponseStatus(employerID, responseID, req.Status); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response status updated successfully"})
}

func (h *ResponseHandler) MarkResponseAsViewed(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	responseID := c.Param("responseId")

	if err := h.responseService.MarkResponseAsViewed(employerID, responseID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response marked as viewed"})
}

func (h *ResponseHandler) GetResponseStats(c *gin.Context) {
	// This route is protected, so we must check authorization
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	castingID := c.Param("castingId")

	stats, err := h.responseService.GetResponseStats(castingID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// --- Common handlers ---

func (h *ResponseHandler) GetResponse(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	responseID := c.Param("responseId")

	response, err := h.responseService.GetResponse(responseID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

package handlers

import (
	"net/http"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type ResponseHandler struct {
	responseService services.ResponseService
}

func NewResponseHandler(responseService services.ResponseService) *ResponseHandler {
	return &ResponseHandler{
		responseService: responseService,
	}
}

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

// Model handlers

func (h *ResponseHandler) CreateResponse(c *gin.Context) {
	modelID := middleware.GetUserID(c)
	castingID := c.Param("castingId")

	var req dto.CreateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	response, err := h.responseService.CreateResponse(modelID, castingID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "model profile not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "casting not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "casting is not active" {
			statusCode = http.StatusBadRequest
		} else if err.Error() == "You have already responded to this casting" {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *ResponseHandler) GetMyResponses(c *gin.Context) {
	modelID := middleware.GetUserID(c)

	responses, err := h.responseService.GetModelResponses(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"responses": responses,
		"total":     len(responses),
	})
}

func (h *ResponseHandler) DeleteResponse(c *gin.Context) {
	modelID := middleware.GetUserID(c)
	responseID := c.Param("responseId")

	if err := h.responseService.DeleteResponse(modelID, responseID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "cannot delete response that has been reviewed" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response deleted successfully"})
}

// Employer handlers

func (h *ResponseHandler) GetCastingResponses(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	castingID := c.Param("castingId")

	responses, err := h.responseService.GetCastingResponses(castingID, employerID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"responses": responses,
		"total":     len(responses),
	})
}

func (h *ResponseHandler) UpdateResponseStatus(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	responseID := c.Param("responseId")

	var req struct {
		Status models.ResponseStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.responseService.UpdateResponseStatus(employerID, responseID, req.Status); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response status updated successfully"})
}

func (h *ResponseHandler) MarkResponseAsViewed(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	responseID := c.Param("responseId")

	if err := h.responseService.MarkResponseAsViewed(employerID, responseID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response marked as viewed"})
}

func (h *ResponseHandler) GetResponseStats(c *gin.Context) {
	castingID := c.Param("castingId")

	stats, err := h.responseService.GetResponseStats(castingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Common handlers

func (h *ResponseHandler) GetResponse(c *gin.Context) {
	userID := middleware.GetUserID(c)
	responseID := c.Param("responseId")

	response, err := h.responseService.GetResponse(responseID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "response not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

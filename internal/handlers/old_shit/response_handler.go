package old_shit

import (
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseHandler struct {
	service *services.ResponseService
}

func NewResponseHandler(service *services.ResponseService) *ResponseHandler {
	return &ResponseHandler{service: service}
}

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

func (h *ResponseHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "response not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *ResponseHandler) ListByCasting(c *gin.Context) {
	castingID := c.Query("casting_id")
	responses, err := h.service.ListByCasting(c.Request.Context(), castingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list responses"})
		return
	}
	c.JSON(http.StatusOK, responses)
}

func (h *ResponseHandler) AcceptResponse(c *gin.Context) {
	responseID := c.Param("id")
	employerID := c.GetString("userID")

	if employerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.service.AcceptResponse(c.Request.Context(), responseID, employerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response accepted successfully"})
}

func (h *ResponseHandler) RejectResponse(c *gin.Context) {
	responseID := c.Param("id")
	employerID := c.GetString("userID")

	if employerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.service.RejectResponse(c.Request.Context(), responseID, employerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response rejected successfully"})
}

func (h *ResponseHandler) ListByModel(c *gin.Context) {
	modelID := c.GetString("userID")

	if modelID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	responses, err := h.service.ListByModel(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list responses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"responses": responses})
}

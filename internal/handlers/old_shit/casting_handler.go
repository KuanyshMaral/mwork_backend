package old_shit

import (
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CastingHandler struct {
	service *services.CastingService
}

func NewCastingHandler(service *services.CastingService) *CastingHandler {
	return &CastingHandler{service: service}
}

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

func (h *CastingHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	casting, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "casting not found"})
		return
	}
	c.JSON(http.StatusOK, casting)
}

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

func (h *CastingHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete casting"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *CastingHandler) ListByEmployer(c *gin.Context) {
	employerID := c.Query("employer_id")
	list, err := h.service.ListByEmployer(c.Request.Context(), employerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch castings"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *CastingHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
}

func (h *CastingHandler) GetResponses(c *gin.Context) {
	castingID := c.Param("id")

	responses, err := h.service.GetResponses(c.Request.Context(), castingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch responses"})
		return
	}

	c.JSON(http.StatusOK, responses)
}

func (h *CastingHandler) AcceptResponse(c *gin.Context) {
	responseID := c.Param("id")

	if err := h.service.AcceptResponse(c.Request.Context(), responseID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "response accepted, chat created"})
}

func (h *CastingHandler) RejectResponse(c *gin.Context) {
	responseID := c.Param("id")

	if err := h.service.RejectResponse(c.Request.Context(), responseID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "response rejected"})
}

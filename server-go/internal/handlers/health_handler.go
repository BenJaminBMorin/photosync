package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/photosync/server/internal/models"
)

// HealthHandler handles health check endpoints
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck returns the server health status
// @Summary Health check
// @Description Returns the current health status of the server
// @Tags health
// @Produce json
// @Success 200 {object} models.HealthResponse "Server is healthy"
// @Router /api/health [get]
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

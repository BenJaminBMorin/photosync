package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	setupRepo repository.SetupConfigRepo
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(setupRepo repository.SetupConfigRepo) *HealthHandler {
	return &HealthHandler{
		setupRepo: setupRepo,
	}
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

// GetAppInfo returns public app information
// @Summary Get app info
// @Description Returns public app information like app name
// @Tags public
// @Produce json
// @Success 200 {object} models.AppInfoResponse
// @Router /api/info [get]
func (h *HealthHandler) GetAppInfo(w http.ResponseWriter, r *http.Request) {
	appName, _ := h.setupRepo.Get(r.Context(), repository.SetupKeyAppName)
	if appName == "" {
		appName = "PhotoSync"
	}

	response := models.AppInfoResponse{
		AppName: appName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

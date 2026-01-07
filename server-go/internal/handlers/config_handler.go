package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// ConfigHandler handles configuration management endpoints
type ConfigHandler struct {
	configService *services.ConfigService
	smtpService   *services.SMTPService
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(configService *services.ConfigService, smtpService *services.SMTPService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
		smtpService:   smtpService,
	}
}

// GetConfig returns all editable configuration
// @Summary Get all configuration
// @Description Get all editable configuration items with categories
// @Tags config
// @Produce json
// @Success 200 {object} models.ConfigResponse
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/config [get]
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.configService.GetAllConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateConfig updates configuration values
// @Summary Update configuration
// @Description Update one or more configuration items
// @Tags config
// @Accept json
// @Produce json
// @Param request body []models.ConfigUpdate true "Configuration updates"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/config [put]
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var updates []models.ConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(updates) == 0 {
		http.Error(w, "No updates provided", http.StatusBadRequest)
		return
	}

	if err := h.configService.UpdateConfig(r.Context(), updates, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetSMTPConfig returns SMTP configuration (password masked)
// @Summary Get SMTP configuration
// @Description Get SMTP configuration with password masked
// @Tags config
// @Produce json
// @Success 200 {object} models.SMTPConfig
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/config/smtp [get]
func (h *ConfigHandler) GetSMTPConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.configService.GetSMTPConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateSMTPConfig updates SMTP configuration
// @Summary Update SMTP configuration
// @Description Update SMTP settings for email sending
// @Tags config
// @Accept json
// @Produce json
// @Param request body models.SMTPConfig true "SMTP configuration"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/config/smtp [put]
func (h *ConfigHandler) UpdateSMTPConfig(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var config models.SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.configService.UpdateSMTPConfig(r.Context(), &config, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// TestSMTP sends a test email
// @Summary Test SMTP configuration
// @Description Send a test email to verify SMTP configuration
// @Tags config
// @Accept json
// @Produce json
// @Param request body TestSMTPRequest true "Test email address"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/config/smtp/test [post]
func (h *ConfigHandler) TestSMTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	if err := h.configService.TestSMTPConfig(r.Context(), req.Email, h.smtpService); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetRestartStatus checks if server restart is required
// @Summary Get restart status
// @Description Check if any configuration changes require a server restart
// @Tags config
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/config/restart-status [get]
func (h *ConfigHandler) GetRestartStatus(w http.ResponseWriter, r *http.Request) {
	config, err := h.configService.GetAllConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"restartRequired": config.RestartRequired,
	})
}

// TestSMTPRequest for swagger docs
type TestSMTPRequest struct {
	Email string `json:"email"`
}

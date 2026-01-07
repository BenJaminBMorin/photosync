package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// SetupHandler handles setup wizard endpoints
type SetupHandler struct {
	setupService  *services.SetupService
	configService *services.ConfigService
	smtpService   *services.SMTPService
}

// NewSetupHandler creates a new SetupHandler
func NewSetupHandler(
	setupService *services.SetupService,
	configService *services.ConfigService,
	smtpService *services.SMTPService,
) *SetupHandler {
	return &SetupHandler{
		setupService:  setupService,
		configService: configService,
		smtpService:   smtpService,
	}
}

// GetStatus returns the current setup status
// @Summary Get setup status
// @Description Check which setup steps have been completed
// @Tags setup
// @Produce json
// @Success 200 {object} services.SetupStatus
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/status [get]
func (h *SetupHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.setupService.GetStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// UploadFirebaseCredentials handles Firebase service account upload
// @Summary Upload Firebase credentials
// @Description Upload a Firebase service account JSON file
// @Tags setup
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Firebase service account JSON file"
// @Success 200 {object} services.FirebaseConfig
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/firebase [post]
func (h *SetupHandler) UploadFirebaseCredentials(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	config, err := h.setupService.SaveFirebaseCredentials(r.Context(), file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// CreateAdmin creates the first admin user
// @Summary Create admin user
// @Description Create the first admin user during setup
// @Tags setup
// @Accept json
// @Produce json
// @Param request body services.CreateAdminRequest true "Admin user details"
// @Success 200 {object} services.CreateAdminResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/admin [post]
func (h *SetupHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req services.CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.setupService.CreateAdminUser(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CompleteSetup marks setup as complete
// @Summary Complete setup
// @Description Mark the setup wizard as complete
// @Tags setup
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/complete [post]
func (h *SetupHandler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	if err := h.setupService.CompleteSetup(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ConfigureEmail configures SMTP settings during setup
// @Summary Configure email/SMTP
// @Description Configure SMTP settings for email sending during setup
// @Tags setup
// @Accept json
// @Produce json
// @Param request body models.SMTPConfig true "SMTP configuration"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/email [post]
func (h *SetupHandler) ConfigureEmail(w http.ResponseWriter, r *http.Request) {
	var config models.SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Use system user ID for setup operations
	if err := h.configService.UpdateSMTPConfig(r.Context(), &config, "system"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// TestEmail sends a test email during setup
// @Summary Test email configuration
// @Description Send a test email to verify SMTP configuration during setup
// @Tags setup
// @Accept json
// @Produce json
// @Param request body TestEmailRequest true "Test email address"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/email/test [post]
func (h *SetupHandler) TestEmail(w http.ResponseWriter, r *http.Request) {
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

// GetValidationStatus returns critical config validation status
// @Summary Get validation status
// @Description Check critical configuration validation status
// @Tags setup
// @Produce json
// @Success 200 {object} models.ValidationResult
// @Failure 500 {object} models.ErrorResponse
// @Router /api/setup/validation [get]
func (h *SetupHandler) GetValidationStatus(w http.ResponseWriter, r *http.Request) {
	result, err := h.configService.ValidateConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// TestEmailRequest for swagger docs
type TestEmailRequest struct {
	Email string `json:"email"`
}

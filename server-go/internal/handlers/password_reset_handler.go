package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// PasswordResetHandler handles password reset endpoints
type PasswordResetHandler struct {
	resetService *services.PasswordResetService
	authService  *services.AuthService
}

// NewPasswordResetHandler creates a new PasswordResetHandler
func NewPasswordResetHandler(
	resetService *services.PasswordResetService,
	authService *services.AuthService,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		resetService: resetService,
		authService:  authService,
	}
}

// InitiateEmailResetRequest is the request body for initiating email reset
type InitiateEmailResetRequest struct {
	Email string `json:"email"`
}

// SuccessResponse is a generic success response
type SuccessResponse struct {
	Success bool `json:"success"`
}

// InitiateEmailReset starts a password reset flow via email
// @Summary Initiate email-based password reset
// @Description Send password reset code to email address
// @Tags password-reset
// @Accept json
// @Produce json
// @Param request body InitiateEmailResetRequest true "Email address"
// @Success 200 {object} SuccessResponse
// @Router /api/auth/password-reset/initiate-email [post]
func (h *PasswordResetHandler) InitiateEmailReset(w http.ResponseWriter, r *http.Request) {
	var req InitiateEmailResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email is present
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Get IP address
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
		if ipAddress == "" {
			ipAddress = "127.0.0.1"
		}
	}

	// Initiate email reset (always returns success to prevent email enumeration)
	if err := h.resetService.InitiateEmailReset(r.Context(), req.Email, ipAddress); err != nil {
		// Log error but don't return to user (prevent enumeration)
	}

	// Always return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{Success: true})
}

// VerifyCodeAndResetRequest is the request body for verifying reset code
type VerifyCodeAndResetRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

// VerifyCodeAndResetResponse is the response body for password reset verification
type VerifyCodeAndResetResponse struct {
	Success bool `json:"success"`
}

// VerifyCodeAndReset verifies a reset code and updates the password
// @Summary Verify reset code and reset password
// @Description Verify email reset code and set new password
// @Tags password-reset
// @Accept json
// @Produce json
// @Param request body VerifyCodeAndResetRequest true "Reset code and new password"
// @Success 200 {object} VerifyCodeAndResetResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/auth/password-reset/verify-code [post]
func (h *PasswordResetHandler) VerifyCodeAndReset(w http.ResponseWriter, r *http.Request) {
	var req VerifyCodeAndResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate all required fields
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.Code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}
	if req.NewPassword == "" {
		http.Error(w, "New password is required", http.StatusBadRequest)
		return
	}

	// Get IP address
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
		if ipAddress == "" {
			ipAddress = "127.0.0.1"
		}
	}

	// Verify code and reset password
	if err := h.resetService.VerifyCodeAndResetPassword(r.Context(), req.Email, req.Code, req.NewPassword, ipAddress); err != nil {
		switch err {
		case models.ErrResetTokenNotFound, models.ErrInvalidResetCode, models.ErrTooManyAttempts:
			http.Error(w, "Invalid or expired reset code", http.StatusUnauthorized)
			return
		case models.ErrPasswordTooShort:
			http.Error(w, "Password does not meet minimum requirements", http.StatusBadRequest)
			return
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VerifyCodeAndResetResponse{Success: true})
}

// InitiatePhoneResetRequest is the request body for phone-based reset
type InitiatePhoneResetRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"newPassword"`
}

// InitiatePhoneResetResponse is the response body for initiating phone reset
type InitiatePhoneResetResponse struct {
	RequestID string `json:"requestId"`
	ExpiresAt string `json:"expiresAt"`
}

// InitiatePhoneReset starts a password reset flow via phone 2FA
// @Summary Initiate phone-based password reset
// @Description Start password reset with phone 2FA approval from registered device
// @Tags password-reset
// @Accept json
// @Produce json
// @Param request body InitiatePhoneResetRequest true "Email and new password"
// @Success 200 {object} InitiatePhoneResetResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/auth/password-reset/initiate-phone [post]
func (h *PasswordResetHandler) InitiatePhoneReset(w http.ResponseWriter, r *http.Request) {
	var req InitiatePhoneResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.NewPassword == "" {
		http.Error(w, "New password is required", http.StatusBadRequest)
		return
	}

	// Get IP address
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
		if ipAddress == "" {
			ipAddress = "127.0.0.1"
		}
	}

	// Get User-Agent
	userAgent := r.Header.Get("User-Agent")

	// Initiate phone reset
	requestID, err := h.resetService.InitiatePhoneReset(r.Context(), req.Email, req.NewPassword, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(InitiatePhoneResetResponse{
		RequestID: requestID,
		ExpiresAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})
}

// CheckPhoneResetStatusResponse is the response body for checking phone reset status
type CheckPhoneResetStatusResponse struct {
	Status       models.AuthRequestStatus `json:"status"`
	ExpiresAt    string                   `json:"expiresAt"`
	SessionToken string                   `json:"sessionToken,omitempty"`
}

// CheckPhoneResetStatus polls for phone reset request approval status
// @Summary Check phone reset status
// @Description Poll the status of a phone-based password reset request
// @Tags password-reset
// @Produce json
// @Param id path string true "Reset request ID"
// @Success 200 {object} CheckPhoneResetStatusResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/auth/password-reset/status/{id} [get]
func (h *PasswordResetHandler) CheckPhoneResetStatus(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	// Check auth request status
	status, err := h.authService.CheckAuthStatus(r.Context(), requestID)
	if err != nil {
		if err == models.ErrAuthRequestNotFound {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return status response
	response := CheckPhoneResetStatusResponse{
		Status:       status.Status,
		ExpiresAt:    status.ExpiresAt.Format(time.RFC3339),
		SessionToken: status.SessionToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CompletePhoneResetResponse is the response body for completing phone reset
type CompletePhoneResetResponse struct {
	Success bool `json:"success"`
}

// CompletePhoneReset completes a phone-based password reset after approval
// @Summary Complete phone reset
// @Description Finalize password reset after device approval
// @Tags password-reset
// @Produce json
// @Param id path string true "Reset request ID"
// @Success 200 {object} CompletePhoneResetResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/auth/password-reset/complete/{id} [post]
func (h *PasswordResetHandler) CompletePhoneReset(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	// Complete phone reset
	if err := h.resetService.CompletePhoneReset(r.Context(), requestID); err != nil {
		if err == models.ErrAuthRequestNotFound {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CompletePhoneResetResponse{Success: true})
}

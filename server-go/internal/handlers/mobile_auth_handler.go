package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

// MobileAuthHandler handles mobile app authentication endpoints
type MobileAuthHandler struct {
	mobileAuthService *services.MobileAuthService
	deviceRepo        repository.DeviceRepo
	userRepo          repository.UserRepo
}

// NewMobileAuthHandler creates a new MobileAuthHandler
func NewMobileAuthHandler(
	mobileAuthService *services.MobileAuthService,
	deviceRepo repository.DeviceRepo,
	userRepo repository.UserRepo,
) *MobileAuthHandler {
	return &MobileAuthHandler{
		mobileAuthService: mobileAuthService,
		deviceRepo:        deviceRepo,
		userRepo:          userRepo,
	}
}

// LoginRequest is the request body for mobile login
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceName string `json:"deviceName"`
	Platform   string `json:"platform"`
	FCMToken   string `json:"fcmToken"`
}

// LoginResponse is the response body for successful login
type LoginResponse struct {
	Success bool                  `json:"success"`
	User    models.UserResponse   `json:"user"`
	Device  models.DeviceResponse `json:"device"`
	APIKey  string                `json:"apiKey"`
}

// Login authenticates a mobile app user and returns API key
// @Summary Mobile login with password
// @Description Authenticate mobile app user with email and password
// @Tags mobile-auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/mobile/auth/login [post]
func (h *MobileAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate all required fields
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}
	if req.DeviceName == "" {
		http.Error(w, "Device name is required", http.StatusBadRequest)
		return
	}
	if req.Platform == "" {
		http.Error(w, "Platform is required", http.StatusBadRequest)
		return
	}
	if req.FCMToken == "" {
		http.Error(w, "FCM token is required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := h.mobileAuthService.LoginWithPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case models.ErrUserNotFound, models.ErrInvalidPassword:
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		case models.ErrPasswordNotSet:
			http.Error(w, "Password not set for this user", http.StatusBadRequest)
			return
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Check if device already exists with this FCM token
	device, err := h.deviceRepo.GetByFCMToken(r.Context(), req.FCMToken)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update or create device
	if device != nil {
		// Device exists, update it
		if err := h.deviceRepo.UpdateToken(r.Context(), device.ID, req.FCMToken); err != nil {
			http.Error(w, "Failed to update device", http.StatusInternalServerError)
			return
		}
	} else {
		// Create new device
		device, err = models.NewDevice(user.ID, req.DeviceName, req.Platform, req.FCMToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.deviceRepo.Add(r.Context(), device); err != nil {
			http.Error(w, "Failed to register device", http.StatusInternalServerError)
			return
		}
	}

	// Generate new API key
	newAPIKey, err := models.GenerateAPIKey()
	if err != nil {
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Hash the API key
	apiKeyHash := models.HashAPIKey(newAPIKey)

	// Update user with new API key hash
	if err := h.userRepo.UpdateAPIKeyHash(r.Context(), user.ID, apiKeyHash); err != nil {
		http.Error(w, "Failed to save API key", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Success: true,
		User:    user.ToResponse(),
		Device:  device.ToResponse(),
		APIKey:  newAPIKey,
	})
}

// RefreshAPIKeyRequest is the request body for API key refresh
type RefreshAPIKeyRequest struct {
	Password string `json:"password"`
}

// RefreshAPIKeyResponse is the response body for API key refresh
type RefreshAPIKeyResponse struct {
	Success bool   `json:"success"`
	APIKey  string `json:"apiKey"`
}

// RefreshAPIKey generates a new API key after password verification
// @Summary Refresh API key
// @Description Generate a new API key for the current user after password verification
// @Tags mobile-auth
// @Accept json
// @Produce json
// @Param request body RefreshAPIKeyRequest true "Password for verification"
// @Success 200 {object} RefreshAPIKeyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/mobile/auth/refresh-key [post]
func (h *MobileAuthHandler) RefreshAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var req RefreshAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate password is provided
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Refresh API key
	newAPIKey, err := h.mobileAuthService.RefreshAPIKey(r.Context(), user.ID, req.Password)
	if err != nil {
		switch err {
		case models.ErrInvalidPassword:
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RefreshAPIKeyResponse{
		Success: true,
		APIKey:  newAPIKey,
	})
}

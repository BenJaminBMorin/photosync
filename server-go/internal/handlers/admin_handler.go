package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// AdminHandler handles admin API endpoints
type AdminHandler struct {
	adminService *services.AdminService
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// ListUsers returns all users with statistics
// @Summary List all users
// @Description Get a list of all users with their statistics
// @Tags admin
// @Produce json
// @Success 200 {object} models.UserListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users [get]
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.adminService.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetUser returns a single user with statistics
// @Summary Get user details
// @Description Get detailed information about a specific user
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.AdminUserResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users/{id} [get]
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	user, err := h.adminService.GetUser(r.Context(), userID)
	if err != nil {
		if err == models.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user and return their API key (shown only once)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "User details"
// @Success 201 {object} models.CreateUserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users [post]
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.adminService.CreateUser(r.Context(), req)
	if err != nil {
		if err == models.ErrEmailExists {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		if err == models.ErrEmptyEmail || err == models.ErrEmptyDisplayName {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.CreateUserResponse{
		User:   user.ToResponse(),
		APIKey: user.APIKey,
	})
}

// UpdateUser updates a user's details
// @Summary Update user
// @Description Update a user's details
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body models.UpdateUserRequest true "User details"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.adminService.UpdateUser(r.Context(), userID, req); err != nil {
		if err == models.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err == models.ErrEmailExists {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteUser deletes a user
// @Summary Delete user
// @Description Delete a user (cannot delete yourself)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	adminUser := middleware.GetUserFromContext(r.Context())

	if err := h.adminService.DeleteUser(r.Context(), userID, adminUser.ID); err != nil {
		if err == models.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ResetAPIKey generates a new API key for a user
// @Summary Reset API key
// @Description Generate a new API key for a user (shown only once)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.ResetAPIKeyResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users/{id}/reset-api-key [post]
func (h *AdminHandler) ResetAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	apiKey, err := h.adminService.ResetAPIKey(r.Context(), userID)
	if err != nil {
		if err == models.ErrUserNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ResetAPIKeyResponse{APIKey: apiKey})
}

// GetUserDevices returns all devices for a user
// @Summary List user devices
// @Description Get all devices registered for a user
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {array} models.AdminDeviceResponse
// @Security SessionAuth
// @Router /api/admin/users/{id}/devices [get]
func (h *AdminHandler) GetUserDevices(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	devices, err := h.adminService.GetUserDevices(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]models.AdminDeviceResponse, 0, len(devices))
	for _, d := range devices {
		responses = append(responses, models.AdminDeviceResponse{
			ID:           d.ID,
			UserID:       d.UserID,
			DeviceName:   d.DeviceName,
			Platform:     d.Platform,
			RegisteredAt: d.RegisteredAt,
			LastSeenAt:   d.LastSeenAt,
			IsActive:     d.IsActive,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// DeleteUserDevice removes a device
// @Summary Delete device
// @Description Remove a device registration
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Param deviceId path string true "Device ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/users/{id}/devices/{deviceId} [delete]
func (h *AdminHandler) DeleteUserDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	if err := h.adminService.DeleteDevice(r.Context(), deviceID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetUserSessions returns all active sessions for a user
// @Summary List user sessions
// @Description Get all active web sessions for a user
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {array} models.AdminSessionResponse
// @Security SessionAuth
// @Router /api/admin/users/{id}/sessions [get]
func (h *AdminHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	sessions, err := h.adminService.GetUserSessions(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]models.AdminSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		responses = append(responses, models.AdminSessionResponse{
			ID:             s.ID,
			UserID:         s.UserID,
			CreatedAt:      s.CreatedAt,
			ExpiresAt:      s.ExpiresAt,
			LastActivityAt: s.LastActivityAt,
			IPAddress:      s.IPAddress,
			UserAgent:      s.UserAgent,
			IsActive:       s.IsActive,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// InvalidateUserSession ends a specific session
// @Summary Invalidate session
// @Description Force logout a specific session
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Param sessionId path string true "Session ID"
// @Success 200 {object} map[string]bool
// @Security SessionAuth
// @Router /api/admin/users/{id}/sessions/{sessionId} [delete]
func (h *AdminHandler) InvalidateUserSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if err := h.adminService.InvalidateSession(r.Context(), sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetSystemStatus returns system health and statistics
// @Summary Get system status
// @Description Get system health, statistics, and configuration status
// @Tags admin
// @Produce json
// @Success 200 {object} models.SystemStatusResponse
// @Security SessionAuth
// @Router /api/admin/system/status [get]
func (h *AdminHandler) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.adminService.GetSystemStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetSystemConfig returns current system configuration
// @Summary Get system configuration
// @Description Get current system configuration settings
// @Tags admin
// @Produce json
// @Success 200 {object} models.SystemConfigResponse
// @Security SessionAuth
// @Router /api/admin/system/config [get]
func (h *AdminHandler) GetSystemConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.adminService.GetSystemConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

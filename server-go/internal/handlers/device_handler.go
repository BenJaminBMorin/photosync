package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// DeviceHandler handles device registration endpoints
type DeviceHandler struct {
	deviceRepo repository.DeviceRepo
}

// NewDeviceHandler creates a new DeviceHandler
func NewDeviceHandler(deviceRepo repository.DeviceRepo) *DeviceHandler {
	return &DeviceHandler{
		deviceRepo: deviceRepo,
	}
}

// RegisterDevice registers a new device for push notifications
// @Summary Register device
// @Description Register a device for push notifications
// @Tags devices
// @Accept json
// @Produce json
// @Param request body models.RegisterDeviceRequest true "Device info"
// @Success 200 {object} models.DeviceResponse
// @Failure 400 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/devices/register [post]
func (h *DeviceHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FCMToken == "" {
		http.Error(w, "FCM token is required", http.StatusBadRequest)
		return
	}

	// Check if device with this token already exists
	existing, err := h.deviceRepo.GetByFCMToken(r.Context(), req.FCMToken)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var device *models.Device
	if existing != nil {
		// Update existing device
		if err := h.deviceRepo.UpdateToken(r.Context(), existing.ID, req.FCMToken); err != nil {
			http.Error(w, "Failed to update device", http.StatusInternalServerError)
			return
		}
		device = existing
	} else {
		// Create new device
		var err error
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device.ToResponse())
}

// ListDevices returns all devices for the current user
// @Summary List devices
// @Description List all registered devices for the current user
// @Tags devices
// @Produce json
// @Success 200 {array} models.DeviceResponse
// @Security ApiKeyAuth
// @Router /api/devices [get]
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	devices, err := h.deviceRepo.GetAllForUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to list devices", http.StatusInternalServerError)
		return
	}

	responses := make([]models.DeviceResponse, 0, len(devices))
	for _, d := range devices {
		responses = append(responses, d.ToResponse())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// DeleteDevice removes a device
// @Summary Delete device
// @Description Remove a registered device
// @Tags devices
// @Param id path string true "Device ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/devices/{id} [delete]
func (h *DeviceHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		http.Error(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	// Verify device belongs to user
	device, err := h.deviceRepo.GetByID(r.Context(), deviceID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if device == nil || device.UserID != user.ID {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	deleted, err := h.deviceRepo.Delete(r.Context(), deviceID)
	if err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

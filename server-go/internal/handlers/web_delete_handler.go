package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// WebDeleteHandler handles web photo deletion endpoints
type WebDeleteHandler struct {
	deleteService *services.DeleteService
}

// NewWebDeleteHandler creates a new WebDeleteHandler
func NewWebDeleteHandler(deleteService *services.DeleteService) *WebDeleteHandler {
	return &WebDeleteHandler{
		deleteService: deleteService,
	}
}

// InitiateDelete starts the push notification delete approval flow
// @Summary Initiate photo deletion
// @Description Start the push notification photo deletion approval flow
// @Tags web-delete
// @Accept json
// @Produce json
// @Param request body models.InitiateDeleteRequest true \"Photo IDs to delete\"
// @Success 200 {object} services.InitiateDeleteResult
// @Failure 400 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/web/delete/initiate [post]
func (h *WebDeleteHandler) InitiateDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var req models.InitiateDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.PhotoIDs) == 0 {
		http.Error(w, "Photo IDs are required", http.StatusBadRequest)
		return
	}

	// Get client IP and user agent
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	userAgent := r.Header.Get("User-Agent")

	result, err := h.deleteService.InitiateDelete(r.Context(), user.ID, req.PhotoIDs, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CheckStatus polls for delete request status
// @Summary Check delete status
// @Description Check the status of a pending delete request
// @Tags web-delete
// @Produce json
// @Param id path string true \"Delete request ID\"
// @Success 200 {object} models.DeleteStatusResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/web/delete/status/{id} [get]
func (h *WebDeleteHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	status, err := h.deleteService.CheckDeleteStatus(r.Context(), requestID)
	if err != nil {
		if err == models.ErrDeleteRequestNotFound {
			http.Error(w, "Delete request not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// RespondDelete handles approve/deny from mobile
// @Summary Respond to delete request
// @Description Approve or deny a delete request from mobile app
// @Tags web-delete
// @Accept json
// @Produce json
// @Param request body models.RespondDeleteRequest true \"Response\"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/web/delete/respond [post]
func (h *WebDeleteHandler) RespondDelete(w http.ResponseWriter, r *http.Request) {
	var req models.RespondDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get device ID from request body (sent by mobile app)
	deviceID := ""
	if req.DeviceID != nil {
		deviceID = *req.DeviceID
	}

	if err := h.deleteService.RespondToDelete(r.Context(), req.RequestID, req.Approved, deviceID); err != nil {
		if err == models.ErrDeleteRequestNotFound {
			http.Error(w, "Delete request not found", http.StatusNotFound)
			return
		}
		if err == models.ErrDeleteAlreadyResolved {
			http.Error(w, "Delete request already resolved", http.StatusConflict)
			return
		}
		if err == models.ErrDeleteRequestExpired {
			http.Error(w, "Delete request expired", http.StatusGone)
			return
		}
		log.Printf("ERROR: RespondToDelete failed for request %s: %v", req.RequestID, err)
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

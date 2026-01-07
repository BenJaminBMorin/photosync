package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// Ensure models is used
var _ = models.HashAPIKey

// WebAuthHandler handles web authentication endpoints
type WebAuthHandler struct {
	authService *services.AuthService
}

// NewWebAuthHandler creates a new WebAuthHandler
func NewWebAuthHandler(authService *services.AuthService) *WebAuthHandler {
	return &WebAuthHandler{
		authService: authService,
	}
}

// InitiateAuth starts the push notification auth flow
// @Summary Initiate authentication
// @Description Start the push notification authentication flow
// @Tags web-auth
// @Accept json
// @Produce json
// @Param request body models.InitiateAuthRequest true "Email to authenticate"
// @Success 200 {object} services.InitiateAuthResult
// @Failure 400 {object} models.ErrorResponse
// @Router /api/web/auth/initiate [post]
func (h *WebAuthHandler) InitiateAuth(w http.ResponseWriter, r *http.Request) {
	var req models.InitiateAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Get client IP and user agent
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	userAgent := r.Header.Get("User-Agent")

	result, err := h.authService.InitiateAuth(r.Context(), req.Email, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CheckStatus polls for auth request status
// @Summary Check auth status
// @Description Check the status of a pending auth request
// @Tags web-auth
// @Produce json
// @Param id path string true "Auth request ID"
// @Success 200 {object} models.AuthStatusResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/web/auth/status/{id} [get]
func (h *WebAuthHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	status, err := h.authService.CheckAuthStatus(r.Context(), requestID)
	if err != nil {
		if err == models.ErrAuthRequestNotFound {
			http.Error(w, "Auth request not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If approved with session token, set cookie
	if status.Status == models.AuthStatusApproved && status.SessionToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    status.SessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   24 * 60 * 60, // 24 hours
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// RespondAuth handles approve/deny from mobile
// @Summary Respond to auth request
// @Description Approve or deny an auth request from mobile app
// @Tags web-auth
// @Accept json
// @Produce json
// @Param request body models.RespondAuthRequest true "Response"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/web/auth/respond [post]
func (h *WebAuthHandler) RespondAuth(w http.ResponseWriter, r *http.Request) {
	var req models.RespondAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get device ID from context (if we track which device responded)
	deviceID := ""
	// TODO: Get from device registration context

	if err := h.authService.RespondToAuth(r.Context(), req.RequestID, req.Approved, deviceID); err != nil {
		if err == models.ErrAuthRequestNotFound {
			http.Error(w, "Auth request not found", http.StatusNotFound)
			return
		}
		if err == models.ErrAuthAlreadyResolved {
			http.Error(w, "Auth request already resolved", http.StatusConflict)
			return
		}
		if err == models.ErrAuthRequestExpired {
			http.Error(w, "Auth request expired", http.StatusGone)
			return
		}
		log.Printf("ERROR: RespondToAuth failed for request %s: %v", req.RequestID, err)
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetSession returns current session info
// @Summary Get current session
// @Description Get information about the current web session
// @Tags web-auth
// @Produce json
// @Success 200 {object} models.SessionResponse
// @Failure 401 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/web/session [get]
func (h *WebAuthHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSessionFromContext(r.Context())
	user := middleware.GetUserFromContext(r.Context())

	if session == nil || user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	response := models.SessionResponse{
		ExpiresAt:      session.ExpiresAt,
		LastActivityAt: session.LastActivityAt,
		User:           user.ToResponse(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AdminLogin allows login with API key directly (for testing/admin access)
// @Summary Admin login with API key
// @Description Login directly using API key (bypasses push notification)
// @Tags web-auth
// @Accept json
// @Produce json
// @Param request body AdminLoginRequest true "API key"
// @Success 200 {object} map[string]string
// @Failure 401 {object} models.ErrorResponse
// @Router /api/web/auth/admin-login [post]
func (h *WebAuthHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Look up user by API key hash
	keyHash := models.HashAPIKey(req.APIKey)
	user, err := h.authService.GetUserByAPIKeyHash(r.Context(), keyHash)
	if err != nil || user == nil {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Create session directly
	session, err := h.authService.CreateSessionForUser(r.Context(), user.ID, r.RemoteAddr, r.Header.Get("User-Agent"))
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"sessionId": session.ID,
	})
}

// AdminLoginRequest for swagger docs
type AdminLoginRequest struct {
	APIKey string `json:"apiKey"`
}

// Logout ends the current session
// @Summary Logout
// @Description End the current web session
// @Tags web-auth
// @Produce json
// @Success 200 {object} map[string]bool
// @Security SessionAuth
// @Router /api/web/auth/logout [post]
func (h *WebAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSessionFromContext(r.Context())
	if session == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	if err := h.authService.Logout(r.Context(), session.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

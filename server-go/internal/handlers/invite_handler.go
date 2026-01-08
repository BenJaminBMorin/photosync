package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

type InviteHandler struct {
	inviteRepo    *repository.InviteTokenRepository
	userRepo      *repository.UserRepository
	smtpService   *services.SMTPService
	serverBaseURL string
}

func NewInviteHandler(
	inviteRepo *repository.InviteTokenRepository,
	userRepo *repository.UserRepository,
	smtpService *services.SMTPService,
	serverBaseURL string,
) *InviteHandler {
	return &InviteHandler{
		inviteRepo:    inviteRepo,
		userRepo:      userRepo,
		smtpService:   smtpService,
		serverBaseURL: serverBaseURL,
	}
}

// GenerateInviteResponse contains the generated invite information
type GenerateInviteResponse struct {
	Token     string `json:"token"`
	InviteURL string `json:"inviteUrl"`
	ExpiresAt string `json:"expiresAt"`
}

// RedeemInviteRequest represents the request to redeem an invite
type RedeemInviteRequest struct {
	Token      string `json:"token"`
	DeviceInfo string `json:"deviceInfo,omitempty"`
}

// RedeemInviteResponse contains the user's credentials after redeeming invite
type RedeemInviteResponse struct {
	ServerURL string `json:"serverUrl"`
	APIKey    string `json:"apiKey"`
	Email     string `json:"email"`
	UserID    string `json:"userId"`
}

// HandleGenerateInvite creates a new invite token for a user (admin only)
func (h *InviteHandler) HandleGenerateInvite(w http.ResponseWriter, r *http.Request) {
	// Get admin user from context (set by auth middleware)
	adminUser, ok := r.Context().Value("user").(*models.User)
	if !ok || !adminUser.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from URL path
	userID := chi.URLParam(r, "id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get the user to invite
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if user already has a pending invite
	hasPending, err := h.inviteRepo.HasPendingInvite(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to check existing invites", http.StatusInternalServerError)
		return
	}
	if hasPending {
		http.Error(w, "User already has a pending invite", http.StatusConflict)
		return
	}

	// Create invite token
	invite, err := models.NewInviteToken(user.ID, user.Email, adminUser.ID)
	if err != nil {
		http.Error(w, "Failed to generate invite token", http.StatusInternalServerError)
		return
	}

	// Save to database
	if err := h.inviteRepo.Add(r.Context(), invite); err != nil {
		http.Error(w, "Failed to save invite token", http.StatusInternalServerError)
		return
	}

	// Generate deep link URL with both token and server
	inviteURL := fmt.Sprintf("photosync://invite?token=%s&server=%s", invite.Token, h.serverBaseURL)

	// Send email with invite link (using SMTP service if configured)
	if h.smtpService != nil {
		if err := h.smtpService.SendInviteEmail(r.Context(), user.Email, user.DisplayName, invite.Token, inviteURL); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to send invite email: %v\n", err)
		}
	}

	// Return invite information
	response := GenerateInviteResponse{
		Token:     invite.Token,
		InviteURL: inviteURL,
		ExpiresAt: invite.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRedeemInvite redeems an invite token and returns user credentials
func (h *InviteHandler) HandleRedeemInvite(w http.ResponseWriter, r *http.Request) {
	var req RedeemInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	// Get invite token
	invite, err := h.inviteRepo.GetByToken(r.Context(), req.Token)
	if err != nil {
		http.Error(w, "Failed to retrieve invite", http.StatusInternalServerError)
		return
	}
	if invite == nil {
		http.Error(w, "Invalid invite token", http.StatusNotFound)
		return
	}

	// Validate invite
	if !invite.IsValid() {
		if invite.Used {
			http.Error(w, "Invite token has already been used", http.StatusGone)
		} else {
			http.Error(w, "Invite token has expired", http.StatusGone)
		}
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(r.Context(), invite.UserID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get client IP
	ipAddress := getClientIP(r)

	// Mark invite as used
	deviceInfo := req.DeviceInfo
	if deviceInfo == "" {
		deviceInfo = r.UserAgent()
	}

	if err := h.inviteRepo.MarkUsed(r.Context(), invite.ID, ipAddress, deviceInfo); err != nil {
		http.Error(w, "Failed to mark invite as used", http.StatusInternalServerError)
		return
	}

	// Return user credentials
	response := RedeemInviteResponse{
		ServerURL: h.serverBaseURL,
		APIKey:    user.APIKey,
		Email:     user.Email,
		UserID:    user.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

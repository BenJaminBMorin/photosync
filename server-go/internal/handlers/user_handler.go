package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// UserHandler handles user-related API endpoints
type UserHandler struct {
	userPrefsRepo repository.UserPreferencesRepository
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userPrefsRepo repository.UserPreferencesRepository) *UserHandler {
	return &UserHandler{
		userPrefsRepo: userPrefsRepo,
	}
}

// GetCurrentUser returns the current authenticated user's information
// GET /api/users/me
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.ToResponse())
}

// GetPreferences returns the current user's preferences
// GET /api/users/me/preferences
func (h *UserHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	prefs, err := h.userPrefsRepo.Get(r.Context(), user.ID)
	if err != nil {
		// If preferences don't exist, return defaults
		if err == models.ErrPreferencesNotFound {
			defaultPrefs := models.NewUserPreferences(user.ID)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(defaultPrefs)
			return
		}
		http.Error(w, "Failed to retrieve preferences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

// UpdatePreferences updates the current user's preferences
// PUT /api/users/me/preferences
func (h *UserHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.UserPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing preferences or create new
	prefs, err := h.userPrefsRepo.Get(r.Context(), user.ID)
	if err != nil && err != models.ErrPreferencesNotFound {
		http.Error(w, "Failed to retrieve preferences", http.StatusInternalServerError)
		return
	}

	if prefs == nil {
		// Create new preferences
		prefs = &models.UserPreferences{
			UserID:    user.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Update fields from request
	if req.GlobalThemeID != nil {
		prefs.GlobalThemeID = req.GlobalThemeID
	}

	// Save preferences
	if err := h.userPrefsRepo.CreateOrUpdate(r.Context(), prefs); err != nil {
		http.Error(w, "Failed to update preferences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

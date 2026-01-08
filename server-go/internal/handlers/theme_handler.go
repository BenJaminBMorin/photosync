package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// ThemeHandler handles theme API endpoints
type ThemeHandler struct {
	themeService *services.ThemeService
}

// NewThemeHandler creates a new ThemeHandler
func NewThemeHandler(themeService *services.ThemeService) *ThemeHandler {
	return &ThemeHandler{
		themeService: themeService,
	}
}

// ListThemes returns all available themes (public endpoint)
// GET /api/themes
func (h *ThemeHandler) ListThemes(w http.ResponseWriter, r *http.Request) {
	themes, err := h.themeService.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve themes", http.StatusInternalServerError)
		return
	}

	// Convert to lightweight ThemeInfo for public listing
	themeInfos := make([]models.ThemeInfo, len(themes))
	for i, theme := range themes {
		themeInfos[i] = theme.ToThemeInfo()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(themeInfos)
}

// GetTheme returns a specific theme by ID (public endpoint)
// GET /api/themes/:id
func (h *ThemeHandler) GetTheme(w http.ResponseWriter, r *http.Request) {
	themeID := chi.URLParam(r, "id")
	if themeID == "" {
		http.Error(w, "Theme ID is required", http.StatusBadRequest)
		return
	}

	theme, err := h.themeService.GetByID(r.Context(), themeID)
	if err != nil {
		if err == models.ErrThemeNotFound {
			http.Error(w, "Theme not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve theme", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(theme)
}

// GetThemeCSS returns the generated CSS for a theme (public endpoint)
// GET /api/themes/:id/css
func (h *ThemeHandler) GetThemeCSS(w http.ResponseWriter, r *http.Request) {
	themeID := chi.URLParam(r, "id")
	if themeID == "" {
		http.Error(w, "Theme ID is required", http.StatusBadRequest)
		return
	}

	css, err := h.themeService.GenerateCSS(r.Context(), themeID)
	if err != nil {
		if err == models.ErrThemeNotFound {
			http.Error(w, "Theme not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to generate CSS", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css")
	w.Write([]byte(css))
}

// --- Admin Endpoints ---

// ListAllThemes returns all themes including full properties (admin only)
// GET /api/admin/themes
func (h *ThemeHandler) ListAllThemes(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	themes, err := h.themeService.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve themes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(themes)
}

// GetThemeAdmin returns a specific theme with full properties (admin only)
// GET /api/admin/themes/:id
func (h *ThemeHandler) GetThemeAdmin(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	themeID := chi.URLParam(r, "id")
	if themeID == "" {
		http.Error(w, "Theme ID is required", http.StatusBadRequest)
		return
	}

	theme, err := h.themeService.GetByID(r.Context(), themeID)
	if err != nil {
		if err == models.ErrThemeNotFound {
			http.Error(w, "Theme not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve theme", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(theme)
}

// CreateTheme creates a new custom theme (admin only)
// POST /api/admin/themes
func (h *ThemeHandler) CreateTheme(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var theme models.Theme
	if err := json.NewDecoder(r.Body).Decode(&theme); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set created_by to current user
	theme.CreatedBy = &user.ID

	if err := h.themeService.Create(r.Context(), &theme); err != nil {
		if err == models.ErrInvalidThemeID || err == models.ErrInvalidThemeName {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to create theme", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&theme)
}

// UpdateTheme updates an existing theme (admin only, not system themes)
// PUT /api/admin/themes/:id
func (h *ThemeHandler) UpdateTheme(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	themeID := chi.URLParam(r, "id")
	if themeID == "" {
		http.Error(w, "Theme ID is required", http.StatusBadRequest)
		return
	}

	var theme models.Theme
	if err := json.NewDecoder(r.Body).Decode(&theme); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the ID matches the URL parameter
	theme.ID = themeID

	if err := h.themeService.Update(r.Context(), &theme); err != nil {
		if err == models.ErrSystemThemeEdit {
			http.Error(w, "System themes cannot be modified", http.StatusForbidden)
			return
		}
		if err == models.ErrThemeNotFound {
			http.Error(w, "Theme not found", http.StatusNotFound)
			return
		}
		if err == models.ErrInvalidThemeID || err == models.ErrInvalidThemeName {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to update theme", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&theme)
}

// DeleteTheme deletes a theme (admin only, not system themes)
// DELETE /api/admin/themes/:id
func (h *ThemeHandler) DeleteTheme(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	themeID := chi.URLParam(r, "id")
	if themeID == "" {
		http.Error(w, "Theme ID is required", http.StatusBadRequest)
		return
	}

	if err := h.themeService.Delete(r.Context(), themeID); err != nil {
		if err == models.ErrSystemThemeEdit {
			http.Error(w, "System themes cannot be deleted", http.StatusForbidden)
			return
		}
		if err == models.ErrThemeNotFound {
			http.Error(w, "Theme not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete theme", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetThemePreview returns CSS preview for a theme (admin only)
// GET /api/admin/themes/:id/preview
func (h *ThemeHandler) GetThemePreview(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	themeID := chi.URLParam(r, "id")
	if themeID == "" {
		http.Error(w, "Theme ID is required", http.StatusBadRequest)
		return
	}

	css, err := h.themeService.GenerateCSS(r.Context(), themeID)
	if err != nil {
		if err == models.ErrThemeNotFound {
			http.Error(w, "Theme not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to generate CSS", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css")
	w.Write([]byte(css))
}

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/services"
)

// CollectionHandler handles collection API endpoints
type CollectionHandler struct {
	collectionService *services.CollectionService
}

// NewCollectionHandler creates a new CollectionHandler
func NewCollectionHandler(collectionService *services.CollectionService) *CollectionHandler {
	return &CollectionHandler{
		collectionService: collectionService,
	}
}

// ListCollections returns collections owned by and shared with the user
func (h *CollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collections, err := h.collectionService.ListCollections(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to list collections", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

// CreateCollection creates a new collection
func (h *CollectionHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection, err := h.collectionService.CreateCollection(r.Context(), user.ID, &req)
	if err != nil {
		if err == models.ErrCollectionSlugExists {
			http.Error(w, "Slug already exists", http.StatusConflict)
			return
		}
		if err == models.ErrCollectionInvalidTheme {
			http.Error(w, "Invalid theme", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to create collection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(collection)
}

// GetCollection returns a collection by ID
func (h *CollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	collection, err := h.collectionService.GetCollection(r.Context(), collectionID, user.ID)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to get collection", http.StatusInternalServerError)
		return
	}

	// Get photos for this collection
	photos, err := h.collectionService.GetPhotos(r.Context(), collectionID, user.ID)
	if err != nil {
		http.Error(w, "Failed to get photos", http.StatusInternalServerError)
		return
	}

	// Get shares if owner
	var shares []*models.CollectionShareWithUser
	if collection.IsOwner {
		shares, err = h.collectionService.GetShares(r.Context(), collectionID, user.ID)
		if err != nil {
			// Non-fatal, just log
			shares = nil
		}
	}

	response := models.CollectionResponse{
		Collection: collection,
		Photos:     toCollectionPhotos(photos),
		Shares:     shares,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateCollection updates a collection
func (h *CollectionHandler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var req models.UpdateCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection, err := h.collectionService.UpdateCollection(r.Context(), collectionID, user.ID, &req)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		if err == models.ErrCollectionSlugExists {
			http.Error(w, "Slug already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to update collection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

// DeleteCollection deletes a collection
func (h *CollectionHandler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	err := h.collectionService.DeleteCollection(r.Context(), collectionID, user.ID)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to delete collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateVisibility changes collection visibility
func (h *CollectionHandler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var req models.UpdateVisibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection, err := h.collectionService.UpdateVisibility(r.Context(), collectionID, user.ID, req.Visibility)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		if err == models.ErrCollectionInvalidVisibility {
			http.Error(w, "Invalid visibility", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to update visibility", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

// AddPhotos adds photos to a collection
func (h *CollectionHandler) AddPhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var req models.AddPhotosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.collectionService.AddPhotos(r.Context(), collectionID, user.ID, req.PhotoIDs)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		if err == models.ErrCollectionPhotoNotOwned {
			http.Error(w, "You can only add your own photos", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to add photos", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemovePhotos removes photos from a collection
func (h *CollectionHandler) RemovePhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var req models.RemovePhotosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.collectionService.RemovePhotos(r.Context(), collectionID, user.ID, req.PhotoIDs)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to remove photos", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReorderPhotos reorders photos in a collection
func (h *CollectionHandler) ReorderPhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var req models.ReorderPhotosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.collectionService.ReorderPhotos(r.Context(), collectionID, user.ID, req.PhotoIDs)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to reorder photos", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ShareWithUsers shares a collection with users by email
func (h *CollectionHandler) ShareWithUsers(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	if collectionID == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var req models.ShareCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	failedEmails, err := h.collectionService.ShareWithUsers(r.Context(), collectionID, user.ID, req.Emails)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to share collection", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"failedEmails": failedEmails,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RemoveShare removes a share from a collection
func (h *CollectionHandler) RemoveShare(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collectionID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if collectionID == "" || userID == "" {
		http.Error(w, "Collection ID and User ID required", http.StatusBadRequest)
		return
	}

	err := h.collectionService.RemoveShare(r.Context(), collectionID, user.ID, userID)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		if err == models.ErrCollectionAccessDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to remove share", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetThemes returns available themes
func (h *CollectionHandler) GetThemes(w http.ResponseWriter, r *http.Request) {
	response := models.ThemesResponse{
		Themes: models.GetAvailableThemes(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper to convert photos to collection photo format
func toCollectionPhotos(photos []*models.Photo) []*models.CollectionPhotoWithDetails {
	result := make([]*models.CollectionPhotoWithDetails, len(photos))
	for i, p := range photos {
		result[i] = &models.CollectionPhotoWithDetails{
			Photo: p,
		}
	}
	return result
}

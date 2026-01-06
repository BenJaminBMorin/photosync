package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// WebGalleryHandler handles web gallery endpoints
type WebGalleryHandler struct {
	photoRepo   *repository.PhotoRepositoryPostgres
	storagePath string
}

// NewWebGalleryHandler creates a new WebGalleryHandler
func NewWebGalleryHandler(photoRepo *repository.PhotoRepositoryPostgres, storagePath string) *WebGalleryHandler {
	return &WebGalleryHandler{
		photoRepo:   photoRepo,
		storagePath: storagePath,
	}
}

// PhotoListResponse is the response for listing photos
type PhotoListResponse struct {
	Photos     interface{} `json:"photos"`
	TotalCount int         `json:"totalCount"`
	Skip       int         `json:"skip"`
	Take       int         `json:"take"`
}

// ListPhotos returns paginated photos for the current user
// @Summary List photos
// @Description Get paginated list of user's photos
// @Tags web-gallery
// @Produce json
// @Param skip query int false "Number of photos to skip" default(0)
// @Param take query int false "Number of photos to return" default(50)
// @Success 200 {object} PhotoListResponse
// @Security SessionAuth
// @Router /api/web/photos [get]
func (h *WebGalleryHandler) ListPhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if skip < 0 {
		skip = 0
	}
	if take <= 0 || take > 100 {
		take = 50
	}

	var photos []*models.Photo
	var count int
	var err error

	// Admin users see all photos, regular users only see their own
	if user.IsAdmin {
		photos, err = h.photoRepo.GetAll(r.Context(), skip, take)
		if err != nil {
			http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
			return
		}
		count, err = h.photoRepo.GetCount(r.Context())
	} else {
		photos, err = h.photoRepo.GetAllForUser(r.Context(), user.ID, skip, take)
		if err != nil {
			http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
			return
		}
		count, err = h.photoRepo.GetCountForUser(r.Context(), user.ID)
	}

	if err != nil {
		http.Error(w, "Failed to get photo count", http.StatusInternalServerError)
		return
	}

	response := PhotoListResponse{
		Photos:     photos,
		TotalCount: count,
		Skip:       skip,
		Take:       take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPhotosWithLocation returns photos that have GPS coordinates for map view
// @Summary List photos with location
// @Description Get paginated list of photos with GPS coordinates
// @Tags web-gallery
// @Produce json
// @Param skip query int false "Number of photos to skip" default(0)
// @Param take query int false "Number of photos to return" default(500)
// @Success 200 {object} PhotoListResponse
// @Security SessionAuth
// @Router /api/web/photos/locations [get]
func (h *WebGalleryHandler) ListPhotosWithLocation(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if skip < 0 {
		skip = 0
	}
	if take <= 0 || take > 1000 {
		take = 500 // Higher limit for map view
	}

	var photos []*models.Photo
	var count int
	var err error

	// Admin users see all photos, regular users only see their own
	if user.IsAdmin {
		photos, err = h.photoRepo.GetPhotosWithLocation(r.Context(), skip, take)
		if err != nil {
			http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
			return
		}
		count, err = h.photoRepo.GetLocationCount(r.Context())
	} else {
		photos, err = h.photoRepo.GetPhotosWithLocationForUser(r.Context(), user.ID, skip, take)
		if err != nil {
			http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
			return
		}
		count, err = h.photoRepo.GetLocationCountForUser(r.Context(), user.ID)
	}

	if err != nil {
		http.Error(w, "Failed to get photo count", http.StatusInternalServerError)
		return
	}

	response := PhotoListResponse{
		Photos:     photos,
		TotalCount: count,
		Skip:       skip,
		Take:       take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ServeImage serves the full image file
// @Summary Get photo image
// @Description Serve the full resolution photo
// @Tags web-gallery
// @Produce image/jpeg
// @Param id path string true "Photo ID"
// @Success 200 {file} binary
// @Failure 404 {string} string "Photo not found"
// @Security SessionAuth
// @Router /api/web/photos/{id}/image [get]
func (h *WebGalleryHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := chi.URLParam(r, "id")
	if photoID == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}

	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Verify ownership (if photo has user_id set)
	if photo.UserID != nil && *photo.UserID != user.ID && !user.IsAdmin {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Serve the file
	imagePath := filepath.Join(h.storagePath, photo.StoredPath)
	h.serveFile(w, imagePath)
}

// ServeThumbnail serves a thumbnail version of the image
// @Summary Get photo thumbnail
// @Description Serve a thumbnail version of the photo. Use size query param: small (200px), medium (500px), large (1000px)
// @Tags web-gallery
// @Produce image/jpeg
// @Param id path string true "Photo ID"
// @Param size query string false "Thumbnail size: small, medium, large" default(small)
// @Success 200 {file} binary
// @Failure 404 {string} string "Photo not found"
// @Security SessionAuth
// @Router /api/web/photos/{id}/thumbnail [get]
func (h *WebGalleryHandler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := chi.URLParam(r, "id")
	if photoID == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}

	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Verify ownership (if photo has user_id set)
	if photo.UserID != nil && *photo.UserID != user.ID && !user.IsAdmin {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Determine which thumbnail size to serve
	size := r.URL.Query().Get("size")
	var thumbPath *string

	switch size {
	case "large":
		thumbPath = photo.ThumbLarge
	case "medium":
		thumbPath = photo.ThumbMedium
	case "small", "":
		thumbPath = photo.ThumbSmall
	default:
		thumbPath = photo.ThumbSmall
	}

	// If thumbnail exists, serve it
	if thumbPath != nil && *thumbPath != "" {
		fullPath := filepath.Join(h.storagePath, *thumbPath)
		if _, err := os.Stat(fullPath); err == nil {
			h.serveFile(w, fullPath)
			return
		}
	}

	// Fall back to original image if no thumbnail
	imagePath := filepath.Join(h.storagePath, photo.StoredPath)
	h.serveFile(w, imagePath)
}

// DeletePhoto deletes a photo (admin only or own photos)
// @Summary Delete a photo
// @Description Delete a photo by ID. Admins can delete any photo, users can only delete their own.
// @Tags web-gallery
// @Param id path string true "Photo ID"
// @Success 204 "Photo deleted successfully"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Photo not found"
// @Security SessionAuth
// @Router /api/web/photos/{id} [delete]
func (h *WebGalleryHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoID := chi.URLParam(r, "id")
	if photoID == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}

	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Check permission: admin can delete any, users can only delete their own
	if !user.IsAdmin {
		if photo.UserID == nil || *photo.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	// Delete the files
	imagePath := filepath.Join(h.storagePath, photo.StoredPath)
	os.Remove(imagePath)

	// Delete thumbnails
	if photo.ThumbSmall != nil {
		os.Remove(filepath.Join(h.storagePath, *photo.ThumbSmall))
	}
	if photo.ThumbMedium != nil {
		os.Remove(filepath.Join(h.storagePath, *photo.ThumbMedium))
	}
	if photo.ThumbLarge != nil {
		os.Remove(filepath.Join(h.storagePath, *photo.ThumbLarge))
	}

	// Delete from database
	_, err = h.photoRepo.Delete(r.Context(), photoID)
	if err != nil {
		http.Error(w, "Failed to delete photo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// serveFile serves a file with proper content type detection
func (h *WebGalleryHandler) serveFile(w http.ResponseWriter, path string) {
	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Detect content type
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])

	// Seek back to beginning
	file.Seek(0, 0)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	io.Copy(w, file)
}

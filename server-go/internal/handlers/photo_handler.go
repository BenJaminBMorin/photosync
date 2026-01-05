package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

// PhotoHandler handles photo-related endpoints
type PhotoHandler struct {
	repo           repository.PhotoRepo
	storageService *services.PhotoStorageService
	hashService    *services.HashService
}

// NewPhotoHandler creates a new PhotoHandler
func NewPhotoHandler(
	repo repository.PhotoRepo,
	storageService *services.PhotoStorageService,
	hashService *services.HashService,
) *PhotoHandler {
	return &PhotoHandler{
		repo:           repo,
		storageService: storageService,
		hashService:    hashService,
	}
}

// Upload handles photo upload
// @Summary Upload a photo
// @Description Upload a new photo to the server. Automatically detects duplicates via SHA256 hash.
// @Tags photos
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Photo file to upload"
// @Param originalFilename formData string false "Original filename (uses uploaded filename if not provided)"
// @Param dateTaken formData string false "Date photo was taken (RFC3339 format)"
// @Success 200 {object} models.UploadResult "Photo uploaded successfully (or duplicate found)"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid API key"
// @Failure 500 {object} models.ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /api/photos/upload [post]
func (h *PhotoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.respondError(w, http.StatusBadRequest, "Request must be multipart/form-data.")
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "No file provided or file is empty.")
		return
	}
	defer file.Close()

	// Get metadata
	originalFilename := r.FormValue("originalFilename")
	if originalFilename == "" {
		originalFilename = header.Filename
	}

	dateTakenStr := r.FormValue("dateTaken")
	dateTaken := time.Now().UTC()
	if dateTakenStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateTakenStr); err == nil {
			dateTaken = parsed
		}
	}

	// Read file content for hashing
	content, err := io.ReadAll(file)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to read file.")
		return
	}

	// Compute hash
	fileHash := h.hashService.ComputeHashBytes(content)

	// Check for duplicate
	existing, err := h.repo.GetByHash(r.Context(), fileHash)
	if err != nil {
		log.Printf("Error checking hash: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	if existing != nil {
		log.Printf("Duplicate photo detected: %s", fileHash)
		h.respondJSON(w, http.StatusOK, models.DuplicateUploadResult(
			existing.ID,
			existing.StoredPath,
			existing.UploadedAt,
		))
		return
	}

	// Store the file
	storedPath, err := h.storageService.Store(
		bytes.NewReader(content),
		originalFilename,
		dateTaken,
		int64(len(content)),
	)
	if err != nil {
		log.Printf("Error storing file: %v", err)
		switch err {
		case models.ErrFileTooLarge:
			h.respondError(w, http.StatusBadRequest, err.Error())
		case models.ErrInvalidExtension:
			h.respondError(w, http.StatusBadRequest, err.Error())
		default:
			h.respondError(w, http.StatusInternalServerError, "Failed to store file.")
		}
		return
	}

	// Create database record
	photo, err := models.NewPhoto(originalFilename, storedPath, fileHash, int64(len(content)), dateTaken)
	if err != nil {
		h.storageService.Delete(storedPath) // Clean up
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.repo.Add(r.Context(), photo); err != nil {
		h.storageService.Delete(storedPath) // Clean up
		log.Printf("Error saving to database: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to save photo record.")
		return
	}

	log.Printf("Photo uploaded: %s -> %s", photo.ID, storedPath)

	h.respondJSON(w, http.StatusOK, models.NewUploadResult(photo.ID, storedPath, photo.UploadedAt))
}

// CheckHashes checks which hashes already exist
// @Summary Check if photos exist by hash
// @Description Check which SHA256 hashes already exist on the server. Useful for avoiding duplicate uploads.
// @Tags photos
// @Accept json
// @Produce json
// @Param request body models.CheckHashesRequest true "Hashes to check (max 1000)"
// @Success 200 {object} models.CheckHashesResult "Hash check results"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid API key"
// @Failure 500 {object} models.ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /api/photos/check [post]
func (h *PhotoHandler) CheckHashes(w http.ResponseWriter, r *http.Request) {
	var req models.CheckHashesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	if len(req.Hashes) == 0 {
		h.respondError(w, http.StatusBadRequest, "At least one hash is required.")
		return
	}

	const maxHashes = 1000
	if len(req.Hashes) > maxHashes {
		h.respondError(w, http.StatusBadRequest, "Maximum 1000 hashes can be checked at once.")
		return
	}

	// Normalize hashes
	normalized := make([]string, 0, len(req.Hashes))
	seen := make(map[string]bool)
	for _, hash := range req.Hashes {
		n := strings.ToLower(strings.TrimSpace(hash))
		if !seen[n] {
			normalized = append(normalized, n)
			seen[n] = true
		}
	}

	existing, err := h.repo.GetExistingHashes(r.Context(), normalized)
	if err != nil {
		log.Printf("Error checking hashes: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	existingSet := make(map[string]bool)
	for _, e := range existing {
		existingSet[e] = true
	}

	missing := make([]string, 0)
	for _, n := range normalized {
		if !existingSet[n] {
			missing = append(missing, n)
		}
	}

	h.respondJSON(w, http.StatusOK, models.CheckHashesResult{
		Existing: existing,
		Missing:  missing,
	})
}

// List returns paginated photos
// @Summary List all photos
// @Description Get a paginated list of all photos stored on the server
// @Tags photos
// @Produce json
// @Param skip query int false "Number of photos to skip" default(0)
// @Param take query int false "Number of photos to return (max 100)" default(50)
// @Success 200 {object} models.PhotoListResponse "List of photos"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid API key"
// @Failure 500 {object} models.ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /api/photos [get]
func (h *PhotoHandler) List(w http.ResponseWriter, r *http.Request) {
	skip := 0
	take := 50

	if s := r.URL.Query().Get("skip"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			skip = v
		}
	}

	if t := r.URL.Query().Get("take"); t != "" {
		if v, err := strconv.Atoi(t); err == nil && v >= 1 && v <= 100 {
			take = v
		}
	}

	photos, err := h.repo.GetAll(r.Context(), skip, take)
	if err != nil {
		log.Printf("Error getting photos: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	totalCount, err := h.repo.GetCount(r.Context())
	if err != nil {
		log.Printf("Error getting count: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	responses := make([]models.PhotoResponse, len(photos))
	for i, p := range photos {
		responses[i] = models.PhotoToResponse(p)
	}

	h.respondJSON(w, http.StatusOK, models.PhotoListResponse{
		Photos:     responses,
		TotalCount: totalCount,
		Skip:       skip,
		Take:       take,
	})
}

// GetByID returns a single photo by ID
// @Summary Get photo by ID
// @Description Get metadata for a single photo by its ID
// @Tags photos
// @Produce json
// @Param id path string true "Photo ID (UUID)"
// @Success 200 {object} models.PhotoResponse "Photo details"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid API key"
// @Failure 404 {object} models.ErrorResponse "Photo not found"
// @Failure 500 {object} models.ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /api/photos/{id} [get]
func (h *PhotoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Photo ID is required.")
		return
	}

	photo, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("Error getting photo: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	if photo == nil {
		h.respondError(w, http.StatusNotFound, "Photo not found.")
		return
	}

	h.respondJSON(w, http.StatusOK, models.PhotoToResponse(photo))
}

// Delete removes a photo by ID
// @Summary Delete a photo
// @Description Delete a photo by its ID. This removes both the database record and the file.
// @Tags photos
// @Param id path string true "Photo ID (UUID)"
// @Success 204 "Photo deleted successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized - invalid API key"
// @Failure 404 {object} models.ErrorResponse "Photo not found"
// @Failure 500 {object} models.ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /api/photos/{id} [delete]
func (h *PhotoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Photo ID is required.")
		return
	}

	photo, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("Error getting photo: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	if photo == nil {
		h.respondError(w, http.StatusNotFound, "Photo not found.")
		return
	}

	// Delete file
	h.storageService.Delete(photo.StoredPath)

	// Delete from database
	deleted, err := h.repo.Delete(r.Context(), id)
	if err != nil {
		log.Printf("Error deleting photo: %v", err)
		h.respondError(w, http.StatusInternalServerError, "Database error.")
		return
	}

	if !deleted {
		h.respondError(w, http.StatusNotFound, "Photo not found.")
		return
	}

	log.Printf("Photo deleted: %s", id)

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *PhotoHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *PhotoHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, models.ErrorResponse{Error: message})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

// ConflictHandler handles file conflict API endpoints (admin only)
type ConflictHandler struct {
	fileConflictRepo repository.FileConflictRepo
	photoRepo        repository.PhotoRepo
	metadataService  *services.MetadataService
}

// NewConflictHandler creates a new ConflictHandler
func NewConflictHandler(
	fileConflictRepo repository.FileConflictRepo,
	photoRepo repository.PhotoRepo,
	metadataService *services.MetadataService,
) *ConflictHandler {
	return &ConflictHandler{
		fileConflictRepo: fileConflictRepo,
		photoRepo:        photoRepo,
		metadataService:  metadataService,
	}
}

// ListConflicts returns all file conflicts
// @Summary List all file conflicts
// @Description Get all file conflicts with optional status filter
// @Tags admin,conflicts
// @Produce json
// @Param status query string false "Filter by status (pending, resolved_db, resolved_file, ignored)"
// @Param skip query int false "Number of records to skip" default(0)
// @Param take query int false "Number of records to return" default(20)
// @Success 200 {object} models.FileConflictListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts [get]
func (h *ConflictHandler) ListConflicts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if take <= 0 {
		take = 20
	}
	if take > 100 {
		take = 100
	}

	conflicts, total, err := h.fileConflictRepo.GetAll(r.Context(), status, skip, take)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.FileConflictListResponse{
		Conflicts:  conflicts,
		TotalCount: total,
		Skip:       skip,
		Take:       take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPendingConflicts returns only pending file conflicts
// @Summary List pending file conflicts
// @Description Get file conflicts that need resolution
// @Tags admin,conflicts
// @Produce json
// @Param skip query int false "Number of records to skip" default(0)
// @Param take query int false "Number of records to return" default(20)
// @Success 200 {object} models.FileConflictListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts/pending [get]
func (h *ConflictHandler) ListPendingConflicts(w http.ResponseWriter, r *http.Request) {
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if take <= 0 {
		take = 20
	}
	if take > 100 {
		take = 100
	}

	conflicts, total, err := h.fileConflictRepo.GetPending(r.Context(), skip, take)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.FileConflictListResponse{
		Conflicts:  conflicts,
		TotalCount: total,
		Skip:       skip,
		Take:       take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetConflict returns details of a specific conflict
// @Summary Get conflict details
// @Description Get detailed information about a specific file conflict
// @Tags admin,conflicts
// @Produce json
// @Param id path string true "Conflict ID"
// @Success 200 {object} models.FileConflict
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts/{id} [get]
func (h *ConflictHandler) GetConflict(w http.ResponseWriter, r *http.Request) {
	conflictID := chi.URLParam(r, "id")

	conflict, err := h.fileConflictRepo.GetByID(r.Context(), conflictID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conflict == nil {
		http.Error(w, "Conflict not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflict)
}

// GetConflictStats returns conflict statistics
// @Summary Get conflict statistics
// @Description Get counts of file conflicts by status
// @Tags admin,conflicts
// @Produce json
// @Success 200 {object} models.FileConflictStats
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts/stats [get]
func (h *ConflictHandler) GetConflictStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.fileConflictRepo.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ResolveConflictDB resolves a conflict by updating the file metadata to match the database
// @Summary Resolve conflict (use DB values)
// @Description Update the file's embedded metadata to match the database values
// @Tags admin,conflicts
// @Accept json
// @Produce json
// @Param id path string true "Conflict ID"
// @Param request body models.ResolveConflictRequest false "Optional notes"
// @Success 200 {object} models.FileConflict
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts/{id}/resolve-db [post]
func (h *ConflictHandler) ResolveConflictDB(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conflictID := chi.URLParam(r, "id")

	var req models.ResolveConflictRequest
	json.NewDecoder(r.Body).Decode(&req) // Ignore error, notes are optional

	// Get conflict details
	conflict, err := h.fileConflictRepo.GetByID(r.Context(), conflictID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conflict == nil {
		http.Error(w, "Conflict not found", http.StatusNotFound)
		return
	}

	// Get photo from database
	photo, err := h.photoRepo.GetByID(r.Context(), conflict.PhotoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Update file metadata to match database
	if h.metadataService != nil {
		updates := map[string]string{
			"PhotoID": photo.ID,
		}
		if photo.UserID != nil {
			updates["UserID"] = *photo.UserID
		}
		if photo.OriginDeviceID != nil {
			updates["DeviceID"] = *photo.OriginDeviceID
		}

		if err := h.metadataService.UpdateEmbeddedMetadata(conflict.FilePath, updates); err != nil {
			http.Error(w, "Failed to update file metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Mark conflict as resolved
	err = h.fileConflictRepo.Resolve(r.Context(), conflictID, models.ConflictStatusResolvedDB, admin.ID, req.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated conflict
	conflict, _ = h.fileConflictRepo.GetByID(r.Context(), conflictID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflict)
}

// ResolveConflictFile resolves a conflict by updating the database to match the file metadata
// @Summary Resolve conflict (use file values)
// @Description Update the database to match the file's embedded metadata values
// @Tags admin,conflicts
// @Accept json
// @Produce json
// @Param id path string true "Conflict ID"
// @Param request body models.ResolveConflictRequest false "Optional notes"
// @Success 200 {object} models.FileConflict
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 501 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts/{id}/resolve-file [post]
func (h *ConflictHandler) ResolveConflictFile(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conflictID := chi.URLParam(r, "id")

	var req models.ResolveConflictRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Get conflict details
	conflict, err := h.fileConflictRepo.GetByID(r.Context(), conflictID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conflict == nil {
		http.Error(w, "Conflict not found", http.StatusNotFound)
		return
	}

	// Get the photo from the database
	photo, err := h.photoRepo.GetByID(r.Context(), conflict.PhotoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if photo == nil {
		http.Error(w, "Associated photo not found", http.StatusNotFound)
		return
	}

	// Update the database to match file metadata based on conflict type
	var updateNotes string
	switch conflict.ConflictType {
	case models.ConflictTypeUserIDMismatch:
		if conflict.FileUserID != nil {
			photo.UserID = conflict.FileUserID
			updateNotes = "Updated photo user ID to match file metadata"
		}
	case models.ConflictTypeDeviceIDMismatch:
		if conflict.FileDeviceID != nil {
			photo.OriginDeviceID = conflict.FileDeviceID
			updateNotes = "Updated photo device ID to match file metadata"
		}
	case models.ConflictTypePhotoIDMismatch:
		// Photo ID mismatch is complex - the file claims to be a different photo
		// We'll update the ownership info but note that manual review may be needed
		if conflict.FileUserID != nil {
			photo.UserID = conflict.FileUserID
		}
		if conflict.FileDeviceID != nil {
			photo.OriginDeviceID = conflict.FileDeviceID
		}
		updateNotes = "Updated photo ownership from file metadata. Photo ID mismatch may require manual review."
	}

	// Save updated photo to database
	if err := h.photoRepo.Update(r.Context(), photo); err != nil {
		http.Error(w, "Failed to update photo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build resolution notes
	notes := req.Notes
	if notes == nil {
		notes = &updateNotes
	} else {
		combined := updateNotes + " " + *notes
		notes = &combined
	}

	// Mark conflict as resolved
	err = h.fileConflictRepo.Resolve(r.Context(), conflictID, models.ConflictStatusResolvedFile, admin.ID, notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated conflict
	conflict, _ = h.fileConflictRepo.GetByID(r.Context(), conflictID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflict)
}

// IgnoreConflict marks a conflict as ignored
// @Summary Ignore conflict
// @Description Mark a file conflict as ignored (won't appear in pending list)
// @Tags admin,conflicts
// @Accept json
// @Produce json
// @Param id path string true "Conflict ID"
// @Param request body models.ResolveConflictRequest false "Optional notes"
// @Success 200 {object} models.FileConflict
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/conflicts/{id}/ignore [post]
func (h *ConflictHandler) IgnoreConflict(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conflictID := chi.URLParam(r, "id")

	var req models.ResolveConflictRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Verify conflict exists
	conflict, err := h.fileConflictRepo.GetByID(r.Context(), conflictID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conflict == nil {
		http.Error(w, "Conflict not found", http.StatusNotFound)
		return
	}

	// Mark as ignored
	err = h.fileConflictRepo.Resolve(r.Context(), conflictID, models.ConflictStatusIgnored, admin.ID, req.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated conflict
	conflict, _ = h.fileConflictRepo.GetByID(r.Context(), conflictID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflict)
}

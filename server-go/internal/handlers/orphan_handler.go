package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

// OrphanHandler handles orphan file API endpoints
type OrphanHandler struct {
	orphanFileRepo   repository.OrphanFileRepo
	photoRepo        repository.PhotoRepo
	deviceRepo       repository.DeviceRepo
	storagePath      string
	storageService   *services.PhotoStorageService
	hashService      *services.HashService
	exifService      *services.EXIFService
	thumbnailService *services.ThumbnailService
	metadataService  *services.MetadataService
}

// NewOrphanHandler creates a new OrphanHandler
func NewOrphanHandler(
	orphanFileRepo repository.OrphanFileRepo,
	photoRepo repository.PhotoRepo,
	deviceRepo repository.DeviceRepo,
	storagePath string,
	storageService *services.PhotoStorageService,
	hashService *services.HashService,
	exifService *services.EXIFService,
	thumbnailService *services.ThumbnailService,
	metadataService *services.MetadataService,
) *OrphanHandler {
	return &OrphanHandler{
		orphanFileRepo:   orphanFileRepo,
		photoRepo:        photoRepo,
		deviceRepo:       deviceRepo,
		storagePath:      storagePath,
		storageService:   storageService,
		hashService:      hashService,
		exifService:      exifService,
		thumbnailService: thumbnailService,
		metadataService:  metadataService,
	}
}

// ====================
// User Endpoints
// ====================

// ListMyOrphans returns orphan files belonging to the authenticated user
// @Summary List my orphan files
// @Description Get orphan files that have embedded metadata matching the authenticated user
// @Tags orphans
// @Produce json
// @Param status query string false "Filter by status (pending, ignored, claimed)"
// @Param skip query int false "Number of records to skip" default(0)
// @Param take query int false "Number of records to return" default(20)
// @Success 200 {object} models.OrphanFileListResponse
// @Failure 401 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/web/orphans [get]
func (h *OrphanHandler) ListMyOrphans(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	status := r.URL.Query().Get("status")
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if take <= 0 {
		take = 20
	}
	if take > 100 {
		take = 100
	}

	orphans, total, err := h.orphanFileRepo.GetForUser(r.Context(), user.ID, status, skip, take)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.OrphanFileListResponse{
		OrphanFiles: orphans,
		TotalCount:  total,
		Skip:        skip,
		Take:        take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ClaimOrphan claims an orphan file and adds it to the user's photo library
// @Summary Claim an orphan file
// @Description Claim an orphan file and add it to the user's photo library (only for files belonging to the user)
// @Tags orphans
// @Produce json
// @Param id path string true "Orphan file ID"
// @Param request body models.ClaimOrphanRequest false "Optional device ID"
// @Success 200 {object} models.ClaimOrphanResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/web/orphans/{id}/claim [post]
func (h *OrphanHandler) ClaimOrphan(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orphanID := chi.URLParam(r, "id")

	// Get the orphan to verify ownership
	orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orphan == nil {
		http.Error(w, "Orphan file not found", http.StatusNotFound)
		return
	}

	// Users can only claim their own orphans
	if orphan.EmbeddedUserID == nil || *orphan.EmbeddedUserID != user.ID {
		http.Error(w, "You can only claim your own orphan files", http.StatusForbidden)
		return
	}

	// Parse optional device ID
	var req models.ClaimOrphanRequest
	json.NewDecoder(r.Body).Decode(&req) // Ignore error, device ID is optional

	deviceID := req.DeviceID
	if deviceID == "" && orphan.EmbeddedDeviceID != nil {
		deviceID = *orphan.EmbeddedDeviceID
	}

	// Create photo record from orphan
	photo, err := h.createPhotoFromOrphan(r.Context(), orphan, user.ID, deviceID)
	if err != nil {
		http.Error(w, "Failed to create photo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Mark orphan as claimed
	err = h.orphanFileRepo.UpdateStatus(r.Context(), orphanID, models.OrphanStatusClaimed, user.ID)
	if err != nil {
		log.Printf("Warning: failed to update orphan status after claiming: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ClaimOrphanResponse{
		Photo:   photo,
		Orphan:  orphan,
		Message: "Orphan file successfully claimed and added to your library",
	})
}

// IgnoreOrphan marks an orphan file as ignored (user can ignore their own orphans)
// @Summary Ignore an orphan file
// @Description Mark an orphan file as ignored (only for files belonging to the user)
// @Tags orphans
// @Produce json
// @Param id path string true "Orphan file ID"
// @Success 200 {object} models.OrphanFile
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/web/orphans/{id}/ignore [post]
func (h *OrphanHandler) IgnoreOrphan(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orphanID := chi.URLParam(r, "id")

	// Get the orphan to verify ownership
	orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orphan == nil {
		http.Error(w, "Orphan file not found", http.StatusNotFound)
		return
	}

	// Users can only ignore their own orphans
	if orphan.EmbeddedUserID == nil || *orphan.EmbeddedUserID != user.ID {
		http.Error(w, "You can only manage your own orphan files", http.StatusForbidden)
		return
	}

	// Update status
	err = h.orphanFileRepo.UpdateStatus(r.Context(), orphanID, models.OrphanStatusIgnored, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated record
	orphan, _ = h.orphanFileRepo.GetByID(r.Context(), orphanID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orphan)
}

// ====================
// Admin Endpoints
// ====================

// AdminListOrphans returns all orphan files (admin only)
// @Summary List all orphan files
// @Description Get all orphan files with optional filters
// @Tags admin,orphans
// @Produce json
// @Param status query string false "Filter by status"
// @Param skip query int false "Number of records to skip" default(0)
// @Param take query int false "Number of records to return" default(20)
// @Success 200 {object} models.OrphanFileListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans [get]
func (h *OrphanHandler) AdminListOrphans(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if take <= 0 {
		take = 20
	}
	if take > 100 {
		take = 100
	}

	orphans, total, err := h.orphanFileRepo.GetAll(r.Context(), status, skip, take)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.OrphanFileListResponse{
		OrphanFiles: orphans,
		TotalCount:  total,
		Skip:        skip,
		Take:        take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AdminListUnassignedOrphans returns orphan files with no embedded user
// @Summary List unassigned orphan files
// @Description Get orphan files that have no embedded user ID
// @Tags admin,orphans
// @Produce json
// @Param skip query int false "Number of records to skip" default(0)
// @Param take query int false "Number of records to return" default(20)
// @Success 200 {object} models.OrphanFileListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/unassigned [get]
func (h *OrphanHandler) AdminListUnassignedOrphans(w http.ResponseWriter, r *http.Request) {
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	take, _ := strconv.Atoi(r.URL.Query().Get("take"))

	if take <= 0 {
		take = 20
	}
	if take > 100 {
		take = 100
	}

	orphans, total, err := h.orphanFileRepo.GetUnassigned(r.Context(), skip, take)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.OrphanFileListResponse{
		OrphanFiles: orphans,
		TotalCount:  total,
		Skip:        skip,
		Take:        take,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AdminGetOrphanStats returns orphan file statistics
// @Summary Get orphan file statistics
// @Description Get counts of orphan files by status
// @Tags admin,orphans
// @Produce json
// @Success 200 {object} models.OrphanFileStats
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/stats [get]
func (h *OrphanHandler) AdminGetOrphanStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.orphanFileRepo.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// AdminAssignOrphan assigns an orphan file to a user
// @Summary Assign orphan to user
// @Description Assign an orphan file to a specific user and optionally device
// @Tags admin,orphans
// @Accept json
// @Produce json
// @Param id path string true "Orphan file ID"
// @Param request body models.AssignOrphanRequest true "Assignment details"
// @Success 200 {object} models.OrphanFile
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/{id}/assign [post]
func (h *OrphanHandler) AdminAssignOrphan(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orphanID := chi.URLParam(r, "id")

	var req models.AssignOrphanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Verify orphan exists
	orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orphan == nil {
		http.Error(w, "Orphan file not found", http.StatusNotFound)
		return
	}

	// Assign to user
	err = h.orphanFileRepo.AssignToUser(r.Context(), orphanID, req.UserID, req.DeviceID, admin.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated record
	orphan, _ = h.orphanFileRepo.GetByID(r.Context(), orphanID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orphan)
}

// AdminDeleteOrphan deletes an orphan file from disk and database
// @Summary Delete orphan file
// @Description Delete an orphan file from both disk and database
// @Tags admin,orphans
// @Produce json
// @Param id path string true "Orphan file ID"
// @Success 204 "No Content"
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/{id} [delete]
func (h *OrphanHandler) AdminDeleteOrphan(w http.ResponseWriter, r *http.Request) {
	orphanID := chi.URLParam(r, "id")

	// Get orphan to find file path
	orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orphan == nil {
		http.Error(w, "Orphan file not found", http.StatusNotFound)
		return
	}

	// Delete file from disk
	fullPath := filepath.Join(h.storagePath, orphan.FilePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		http.Error(w, "Failed to delete file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete from database
	if err := h.orphanFileRepo.Delete(r.Context(), orphanID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AdminBulkAssignOrphans assigns multiple orphan files to a user
// @Summary Bulk assign orphans to user
// @Description Assign multiple orphan files to a specific user
// @Tags admin,orphans
// @Accept json
// @Produce json
// @Param request body models.BulkAssignOrphanRequest true "Bulk assignment details"
// @Success 200 {object} map[string]int
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/bulk-assign [post]
func (h *OrphanHandler) AdminBulkAssignOrphans(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.BulkAssignOrphanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if len(req.OrphanIDs) == 0 {
		http.Error(w, "At least one orphan ID is required", http.StatusBadRequest)
		return
	}

	count, err := h.orphanFileRepo.BulkAssign(r.Context(), req.OrphanIDs, req.UserID, req.DeviceID, admin.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"assigned": count})
}

// AdminBulkDeleteOrphans deletes multiple orphan files
// @Summary Bulk delete orphan files
// @Description Delete multiple orphan files from disk and database
// @Tags admin,orphans
// @Accept json
// @Produce json
// @Param request body models.BulkDeleteOrphanRequest true "IDs to delete"
// @Success 200 {object} map[string]int
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/bulk-delete [post]
func (h *OrphanHandler) AdminBulkDeleteOrphans(w http.ResponseWriter, r *http.Request) {
	var req models.BulkDeleteOrphanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.OrphanIDs) == 0 {
		http.Error(w, "At least one orphan ID is required", http.StatusBadRequest)
		return
	}

	// Delete files from disk
	deletedFiles := 0
	for _, id := range req.OrphanIDs {
		orphan, err := h.orphanFileRepo.GetByID(r.Context(), id)
		if err != nil || orphan == nil {
			continue
		}

		fullPath := filepath.Join(h.storagePath, orphan.FilePath)
		if err := os.Remove(fullPath); err == nil || os.IsNotExist(err) {
			deletedFiles++
		}
	}

	// Delete from database
	count, err := h.orphanFileRepo.BulkDelete(r.Context(), req.OrphanIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"deleted": count, "filesRemoved": deletedFiles})
}

// AdminClaimOrphan claims an orphan file and creates a photo record (admin can claim any orphan)
// @Summary Claim orphan and create photo
// @Description Claim an orphan file and add it to a user's photo library
// @Tags admin,orphans
// @Accept json
// @Produce json
// @Param id path string true "Orphan file ID"
// @Param request body models.AssignOrphanRequest true "Assignment details with user ID"
// @Success 200 {object} models.ClaimOrphanResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/{id}/claim [post]
func (h *OrphanHandler) AdminClaimOrphan(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orphanID := chi.URLParam(r, "id")

	var req models.AssignOrphanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Verify orphan exists
	orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orphan == nil {
		http.Error(w, "Orphan file not found", http.StatusNotFound)
		return
	}

	// Create photo record from orphan
	photo, err := h.createPhotoFromOrphan(r.Context(), orphan, req.UserID, req.DeviceID)
	if err != nil {
		http.Error(w, "Failed to create photo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Mark orphan as claimed
	err = h.orphanFileRepo.UpdateStatus(r.Context(), orphanID, models.OrphanStatusClaimed, admin.ID)
	if err != nil {
		log.Printf("Warning: failed to update orphan status after claiming: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ClaimOrphanResponse{
		Photo:   photo,
		Orphan:  orphan,
		Message: "Orphan file successfully claimed and added to user's library",
	})
}

// AdminBulkClaimOrphans claims multiple orphan files and creates photo records
// @Summary Bulk claim orphans and create photos
// @Description Claim multiple orphan files and add them to a user's photo library with file moves to device folders
// @Tags admin,orphans
// @Accept json
// @Produce json
// @Param request body models.BulkClaimOrphanRequest true "Bulk claim details with user ID and orphan IDs"
// @Success 200 {object} models.BulkClaimOrphanResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/bulk-claim [post]
func (h *OrphanHandler) AdminBulkClaimOrphans(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetUserFromContext(r.Context())
	if admin == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.BulkClaimOrphanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if len(req.OrphanIDs) == 0 {
		http.Error(w, "At least one orphan ID is required", http.StatusBadRequest)
		return
	}

	response := models.BulkClaimOrphanResponse{
		Photos: make([]*models.Photo, 0),
		Errors: make([]models.BulkClaimOrphanError, 0),
	}

	for _, orphanID := range req.OrphanIDs {
		// Get orphan
		orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkClaimOrphanError{
				OrphanID: orphanID,
				Error:    "Failed to get orphan: " + err.Error(),
			})
			continue
		}
		if orphan == nil {
			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkClaimOrphanError{
				OrphanID: orphanID,
				Error:    "Orphan not found",
			})
			continue
		}

		// Skip if already claimed
		if orphan.Status == models.OrphanStatusClaimed {
			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkClaimOrphanError{
				OrphanID: orphanID,
				FilePath: orphan.FilePath,
				Error:    "Orphan already claimed",
			})
			continue
		}

		// Create photo record from orphan
		photo, err := h.createPhotoFromOrphan(r.Context(), orphan, req.UserID, req.DeviceID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkClaimOrphanError{
				OrphanID: orphanID,
				FilePath: orphan.FilePath,
				Error:    err.Error(),
			})
			continue
		}

		// Mark orphan as claimed
		if err := h.orphanFileRepo.UpdateStatus(r.Context(), orphanID, models.OrphanStatusClaimed, admin.ID); err != nil {
			log.Printf("Warning: failed to update orphan status after bulk claiming: %v", err)
		}

		response.ClaimedCount++
		response.Photos = append(response.Photos, photo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetOrphanThumbnail generates and returns a thumbnail for an orphan file
// @Summary Get orphan thumbnail
// @Description Generate and return a thumbnail for an orphan file preview
// @Tags admin,orphans
// @Produce image/jpeg
// @Param id path string true "Orphan file ID"
// @Success 200 {file} binary
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/orphans/{id}/thumbnail [get]
func (h *OrphanHandler) GetOrphanThumbnail(w http.ResponseWriter, r *http.Request) {
	orphanID := chi.URLParam(r, "id")

	orphan, err := h.orphanFileRepo.GetByID(r.Context(), orphanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orphan == nil {
		http.Error(w, "Orphan file not found", http.StatusNotFound)
		return
	}

	// Check if we can generate thumbnails
	if h.thumbnailService == nil {
		http.Error(w, "Thumbnail service not available", http.StatusServiceUnavailable)
		return
	}

	fullPath := filepath.Join(h.storagePath, orphan.FilePath)

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get orientation from EXIF if available
	orientation := 1
	if h.exifService != nil {
		exifData, err := h.exifService.ExtractFromBytes(content)
		if err == nil && exifData != nil {
			orientation = exifData.Orientation
		}
	}

	// Generate a temporary thumbnail
	thumbData, err := h.thumbnailService.GenerateSingleThumbnail(content, 300, orientation)
	if err != nil {
		http.Error(w, "Failed to generate thumbnail: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Write(thumbData)
}

// getDeviceFolderPath returns the folder path for organizing photos by device
// Format: devices/{device_name}/YYYY/MM
func (h *OrphanHandler) getDeviceFolderPath(ctx context.Context, deviceID string, dateTaken time.Time) (string, string, error) {
	deviceName := "unknown"

	if deviceID != "" && h.deviceRepo != nil {
		device, err := h.deviceRepo.GetByID(ctx, deviceID)
		if err == nil && device != nil {
			// Sanitize device name for folder
			deviceName = sanitizeDeviceName(device.DeviceName)
		}
	}

	year := dateTaken.Format("2006")
	month := dateTaken.Format("01")

	folderPath := filepath.Join("devices", deviceName, year, month)
	return folderPath, deviceName, nil
}

// sanitizeDeviceName makes a device name safe for use as a folder name
func sanitizeDeviceName(name string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	name = replacer.Replace(strings.TrimSpace(name))

	// Ensure not empty
	if name == "" {
		name = "unknown"
	}

	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}

	return strings.ToLower(name)
}

// createPhotoFromOrphan creates a photo record from an orphan file
// It moves the file to a device-organized folder structure
func (h *OrphanHandler) createPhotoFromOrphan(ctx context.Context, orphan *models.OrphanFile, userID, deviceID string) (*models.Photo, error) {
	fullPath := filepath.Join(h.storagePath, orphan.FilePath)

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	// Compute hash (use existing if available)
	var fileHash string
	if orphan.FileHash != nil {
		fileHash = *orphan.FileHash
	} else if h.hashService != nil {
		fileHash = h.hashService.ComputeHashBytes(content)
	}

	// Check for duplicate by hash
	existing, err := h.photoRepo.GetByHash(ctx, fileHash)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, models.ErrDuplicatePhoto
	}

	// Extract EXIF metadata
	var exifData *services.EXIFData
	if h.exifService != nil {
		exifData, _ = h.exifService.ExtractFromBytes(content)
	}
	if exifData == nil {
		exifData = &services.EXIFData{Orientation: 1}
	}

	// Determine date taken
	dateTaken := time.Now().UTC()
	if orphan.EmbeddedUploadedAt != nil {
		dateTaken = *orphan.EmbeddedUploadedAt
	} else if exifData.DateTaken != nil {
		dateTaken = *exifData.DateTaken
	}

	// Get original filename from path
	originalFilename := filepath.Base(orphan.FilePath)

	// Move file to device-organized folder if storage service available
	storedPath := orphan.FilePath
	if h.storageService != nil && deviceID != "" {
		deviceFolder, deviceName, err := h.getDeviceFolderPath(ctx, deviceID, dateTaken)
		if err == nil {
			newPath, moveErr := h.storageService.MoveFile(orphan.FilePath, deviceFolder, originalFilename)
			if moveErr != nil {
				log.Printf("Warning: failed to move orphan file to device folder: %v", moveErr)
				// Continue with original path
			} else {
				storedPath = newPath
				log.Printf("Moved orphan file to device folder: %s -> %s (device: %s)", orphan.FilePath, newPath, deviceName)
			}
		}
	}

	// Create photo record
	photo, err := models.NewPhoto(originalFilename, storedPath, fileHash, orphan.FileSize, dateTaken)
	if err != nil {
		return nil, err
	}

	// Set user and device
	photo.UserID = &userID
	if deviceID != "" {
		photo.OriginDeviceID = &deviceID
	}

	// Copy EXIF metadata
	photo.CameraMake = exifData.CameraMake
	photo.CameraModel = exifData.CameraModel
	photo.LensModel = exifData.LensModel
	photo.FocalLength = exifData.FocalLength
	photo.Aperture = exifData.Aperture
	photo.ShutterSpeed = exifData.ShutterSpeed
	photo.ISO = exifData.ISO
	photo.Orientation = exifData.Orientation
	photo.Latitude = exifData.Latitude
	photo.Longitude = exifData.Longitude
	photo.Altitude = exifData.Altitude

	// Generate thumbnails (using new stored path)
	if h.thumbnailService != nil && services.IsSupportedFormat(originalFilename) {
		thumbResult, err := h.thumbnailService.GenerateThumbnails(content, photo.ID, storedPath, exifData.Orientation)
		if err != nil {
			log.Printf("Warning: failed to generate thumbnails for claimed orphan: %v", err)
		} else {
			photo.ThumbSmall = &thumbResult.SmallPath
			photo.ThumbMedium = &thumbResult.MediumPath
			photo.ThumbLarge = &thumbResult.LargePath
			photo.Width = &thumbResult.Width
			photo.Height = &thumbResult.Height
		}
	}

	// Save to database
	if err := h.photoRepo.Add(ctx, photo); err != nil {
		return nil, err
	}

	// Update embedded metadata to reflect new ownership
	if h.metadataService != nil {
		go func() {
			metadata := services.PhotoMetadata{
				PhotoID:    photo.ID,
				UserID:     userID,
				DeviceID:   deviceID,
				FileHash:   fileHash,
				UploadedAt: photo.UploadedAt,
			}
			if err := h.metadataService.EmbedFullMetadata(storedPath, metadata); err != nil {
				log.Printf("Warning: failed to update metadata for claimed orphan %s: %v", photo.ID, err)
			}
		}()
	}

	return photo, nil
}

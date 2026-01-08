package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

// SyncHandler handles photo sync endpoints
type SyncHandler struct {
	photoRepo       repository.PhotoRepo
	deviceRepo      repository.DeviceRepo
	syncStateRepo   repository.DeviceSyncStateRepo
	storageService  *services.PhotoStorageService
}

// NewSyncHandler creates a new SyncHandler
func NewSyncHandler(
	photoRepo repository.PhotoRepo,
	deviceRepo repository.DeviceRepo,
	syncStateRepo repository.DeviceSyncStateRepo,
	storageService *services.PhotoStorageService,
) *SyncHandler {
	return &SyncHandler{
		photoRepo:      photoRepo,
		deviceRepo:     deviceRepo,
		syncStateRepo:  syncStateRepo,
		storageService: storageService,
	}
}

// GetSyncStatus returns sync status for a device
// @Summary Get sync status
// @Description Get sync status including photo counts and whether legacy claiming is needed
// @Tags sync
// @Produce json
// @Param X-Device-ID header string false "Device ID"
// @Success 200 {object} models.SyncStatusResponse
// @Failure 401 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/sync/status [get]
func (h *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	deviceID := r.Header.Get("X-Device-ID")

	// Get total photo count
	totalPhotos, err := h.photoRepo.GetCountForUser(r.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting photo count: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get device photo count
	var devicePhotos int
	if deviceID != "" {
		devicePhotos, err = h.photoRepo.GetCountByOriginDevice(r.Context(), user.ID, deviceID)
		if err != nil {
			log.Printf("Error getting device photo count: %v", err)
		}
	}

	// Get legacy photo count
	legacyPhotos, err := h.photoRepo.GetLegacyPhotoCount(r.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting legacy photo count: %v", err)
	}

	// Get sync version
	syncVersion, err := h.syncStateRepo.GetSyncVersion(r.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting sync version: %v", err)
	}

	// Get last sync timestamp for this device
	var lastSyncAt *time.Time
	if deviceID != "" {
		syncState, err := h.syncStateRepo.Get(r.Context(), deviceID)
		if err == nil && syncState != nil && syncState.LastSyncAt != nil {
			lastSyncAt = syncState.LastSyncAt
		}
	}

	response := models.SyncStatusResponse{
		TotalPhotos:       totalPhotos,
		DevicePhotos:      devicePhotos,
		OtherDevicePhotos: totalPhotos - devicePhotos - legacyPhotos,
		LegacyPhotos:      legacyPhotos,
		LastSyncAt:        lastSyncAt,
		ServerVersion:     syncVersion,
		NeedsLegacyClaim:  legacyPhotos > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncPhotos returns photo metadata in batches
// @Summary Sync photos
// @Description Get photo metadata in batches for sync
// @Tags sync
// @Accept json
// @Produce json
// @Param request body models.SyncPhotosRequest true "Sync request"
// @Success 200 {object} models.SyncPhotosResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/sync/photos [post]
func (h *SyncHandler) SyncPhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.SyncPhotosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	// Get photos with cursor-based pagination
	photos, nextCursor, err := h.photoRepo.GetAllForUserWithCursor(
		r.Context(),
		user.ID,
		req.Cursor,
		limit,
		req.SinceTimestamp,
	)
	if err != nil {
		log.Printf("Error getting photos: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get total count for progress tracking
	totalCount, err := h.photoRepo.GetCountForUser(r.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting photo count: %v", err)
	}

	// Get sync version
	syncVersion, _ := h.syncStateRepo.GetSyncVersion(r.Context(), user.ID)

	// Build device lookup for origin device info
	deviceCache := make(map[string]*models.Device)

	// Convert photos to sync items
	syncItems := make([]models.SyncPhotoItem, len(photos))
	for i, photo := range photos {
		item := models.SyncPhotoItem{
			ID:               photo.ID,
			FileHash:         photo.FileHash,
			OriginalFilename: photo.OriginalFilename,
			FileSize:         photo.FileSize,
			DateTaken:        photo.DateTaken,
			UploadedAt:       photo.UploadedAt,
			Width:            photo.Width,
			Height:           photo.Height,
		}

		// Add thumbnail URL if requested
		if req.IncludeThumbnailURLs && photo.ThumbMedium != nil {
			item.ThumbnailURL = "/api/web/photos/" + photo.ID + "/thumbnail"
		}

		// Add origin device info
		if photo.OriginDeviceID != nil {
			// Check cache first
			device, ok := deviceCache[*photo.OriginDeviceID]
			if !ok {
				// Load device from DB
				device, err = h.deviceRepo.GetByID(r.Context(), *photo.OriginDeviceID)
				if err == nil && device != nil {
					deviceCache[*photo.OriginDeviceID] = device
				}
			}

			if device != nil {
				item.OriginDevice = &models.OriginDeviceInfo{
					ID:              device.ID,
					Name:            device.DeviceName,
					Platform:        device.Platform,
					IsCurrentDevice: req.DeviceID == device.ID,
				}
			}
		}

		syncItems[i] = item
	}

	// Update last sync state if we have a device ID and returned photos
	if req.DeviceID != "" && len(photos) > 0 {
		lastPhotoID := photos[len(photos)-1].ID
		if err := h.syncStateRepo.UpdateLastSync(r.Context(), req.DeviceID, lastPhotoID); err != nil {
			log.Printf("Error updating sync state: %v", err)
		}
	}

	response := models.SyncPhotosResponse{
		Photos: syncItems,
		Pagination: models.PaginationInfo{
			Cursor:  nextCursor,
			HasMore: nextCursor != "",
		},
		Sync: models.SyncInfo{
			TotalCount:    totalCount,
			ReturnedCount: len(syncItems),
			ServerVersion: syncVersion,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetLegacyPhotos returns photos without origin device
// @Summary Get legacy photos
// @Description Get photos uploaded before device tracking was enabled
// @Tags sync
// @Produce json
// @Param limit query int false "Maximum photos to return" default(100)
// @Success 200 {object} models.LegacyPhotosResponse
// @Failure 401 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/sync/legacy-photos [get]
func (h *SyncHandler) GetLegacyPhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	photos, err := h.photoRepo.GetLegacyPhotosForUser(r.Context(), user.ID, limit)
	if err != nil {
		log.Printf("Error getting legacy photos: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	totalCount, err := h.photoRepo.GetLegacyPhotoCount(r.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting legacy photo count: %v", err)
	}

	// Convert to sync items
	syncItems := make([]models.SyncPhotoItem, len(photos))
	for i, photo := range photos {
		syncItems[i] = models.SyncPhotoItem{
			ID:               photo.ID,
			FileHash:         photo.FileHash,
			OriginalFilename: photo.OriginalFilename,
			FileSize:         photo.FileSize,
			DateTaken:        photo.DateTaken,
			UploadedAt:       photo.UploadedAt,
			ThumbnailURL:     "/api/web/photos/" + photo.ID + "/thumbnail",
			Width:            photo.Width,
			Height:           photo.Height,
		}
	}

	response := models.LegacyPhotosResponse{
		Photos:     syncItems,
		TotalCount: totalCount,
		Message:    "These photos were uploaded before device tracking was enabled.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ClaimLegacy claims ownership of legacy photos
// @Summary Claim legacy photos
// @Description Claim ownership of photos without origin device
// @Tags sync
// @Accept json
// @Produce json
// @Param request body models.ClaimLegacyRequest true "Claim request"
// @Success 200 {object} models.ClaimLegacyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/sync/claim-legacy [post]
func (h *SyncHandler) ClaimLegacy(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.ClaimLegacyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	// Verify device belongs to user
	device, err := h.deviceRepo.GetByID(r.Context(), req.DeviceID)
	if err != nil || device == nil || device.UserID != user.ID {
		http.Error(w, "Invalid device", http.StatusBadRequest)
		return
	}

	var claimed int
	if req.ClaimAll {
		claimed, err = h.photoRepo.ClaimAllLegacyPhotos(r.Context(), user.ID, req.DeviceID)
	} else if len(req.PhotoIDs) > 0 {
		claimed, err = h.photoRepo.ClaimLegacyPhotos(r.Context(), req.PhotoIDs, req.DeviceID)
	} else {
		http.Error(w, "No photos specified to claim", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error claiming legacy photos: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	response := models.ClaimLegacyResponse{
		Claimed:        claimed,
		AlreadyClaimed: 0,
		Failed:         0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DownloadPhoto serves full-resolution photo for restoring to camera roll
// @Summary Download photo
// @Description Download full-resolution photo for restoring to device
// @Tags sync
// @Produce octet-stream
// @Param id path string true "Photo ID"
// @Success 200 {file} binary "Photo file"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/photos/{id}/download [get]
func (h *SyncHandler) DownloadPhoto(w http.ResponseWriter, r *http.Request) {
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

	// Get photo
	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil {
		log.Printf("Error getting photo: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if photo.UserID == nil || *photo.UserID != user.ID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get full path
	fullPath, err := h.storageService.GetFullPath(photo.StoredPath)
	if err != nil {
		log.Printf("Error getting photo path: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", "attachment; filename=\""+photo.OriginalFilename+"\"")
	w.Header().Set("X-PhotoSync-Hash", photo.FileHash)
	w.Header().Set("X-PhotoSync-DateTaken", photo.DateTaken.Format("2006-01-02T15:04:05Z"))

	// Serve the file (supports range requests)
	http.ServeFile(w, r, fullPath)
}

// DownloadPhotoByHash serves photo by hash (for mobile sync)
// @Summary Download photo by hash
// @Description Download photo by its file hash
// @Tags sync
// @Produce octet-stream
// @Param hash path string true "File hash"
// @Success 200 {file} binary "Photo file"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/sync/download/{hash} [get]
func (h *SyncHandler) DownloadPhotoByHash(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	hash := chi.URLParam(r, "hash")
	if hash == "" {
		http.Error(w, "Hash required", http.StatusBadRequest)
		return
	}

	// Get photo by hash
	photo, err := h.photoRepo.GetByHashAndUser(r.Context(), hash, user.ID)
	if err != nil {
		log.Printf("Error getting photo by hash: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Get full path
	fullPath, err := h.storageService.GetFullPath(photo.StoredPath)
	if err != nil {
		log.Printf("Error getting photo path: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", "attachment; filename=\""+photo.OriginalFilename+"\"")
	w.Header().Set("X-PhotoSync-ID", photo.ID)
	w.Header().Set("X-PhotoSync-Hash", photo.FileHash)
	w.Header().Set("X-PhotoSync-DateTaken", photo.DateTaken.Format("2006-01-02T15:04:05Z"))

	// Serve the file (supports range requests)
	http.ServeFile(w, r, fullPath)
}

// GetThumbnail serves thumbnail for a photo
// @Summary Get thumbnail
// @Description Get thumbnail for a photo during sync
// @Tags sync
// @Produce image/jpeg
// @Param id path string true "Photo ID"
// @Success 200 {file} binary "Thumbnail image"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/sync/thumbnail/{id} [get]
func (h *SyncHandler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
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

	// Get photo
	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil || photo == nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if photo.UserID == nil || *photo.UserID != user.ID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get thumbnail path (prefer medium, fall back to small)
	var thumbPath string
	if photo.ThumbMedium != nil {
		thumbPath = *photo.ThumbMedium
	} else if photo.ThumbSmall != nil {
		thumbPath = *photo.ThumbSmall
	} else {
		http.Error(w, "Thumbnail not available", http.StatusNotFound)
		return
	}

	fullPath, err := h.storageService.GetFullPath(thumbPath)
	if err != nil {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.ServeFile(w, r, fullPath)
}

// helper to suppress unused import warning
var _ = io.EOF

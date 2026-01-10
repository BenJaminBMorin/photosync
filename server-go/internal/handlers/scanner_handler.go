package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/photosync/server/internal/services"
)

// ScannerHandler handles file scanner API endpoints (admin only)
type ScannerHandler struct {
	scannerService *services.FileScannerService
}

// NewScannerHandler creates a new ScannerHandler
func NewScannerHandler(scannerService *services.FileScannerService) *ScannerHandler {
	return &ScannerHandler{
		scannerService: scannerService,
	}
}

// GetStatus returns the current scanner status
// @Summary Get scanner status
// @Description Get the current status of the file scanner service
// @Tags admin,scanner
// @Produce json
// @Success 200 {object} services.ScanStatus
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/scanner/status [get]
func (h *ScannerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scannerService.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartScanner enables the file scanner service
// @Summary Start scanner
// @Description Enable the background file scanner service
// @Tags admin,scanner
// @Produce json
// @Success 200 {object} services.ScanStatus
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/scanner/start [post]
func (h *ScannerHandler) StartScanner(w http.ResponseWriter, r *http.Request) {
	h.scannerService.Start()
	status := h.scannerService.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StopScanner disables the file scanner service
// @Summary Stop scanner
// @Description Disable the background file scanner service
// @Tags admin,scanner
// @Produce json
// @Success 200 {object} services.ScanStatus
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/scanner/stop [post]
func (h *ScannerHandler) StopScanner(w http.ResponseWriter, r *http.Request) {
	h.scannerService.Stop()
	status := h.scannerService.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// RunNow triggers an immediate scan
// @Summary Run scan now
// @Description Trigger an immediate file scan (runs in background)
// @Tags admin,scanner
// @Produce json
// @Success 200 {object} services.ScanStatus
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Scan already running"
// @Security SessionAuth
// @Router /api/admin/scanner/run [post]
func (h *ScannerHandler) RunNow(w http.ResponseWriter, r *http.Request) {
	if h.scannerService.IsRunning() {
		http.Error(w, "A scan is already in progress", http.StatusConflict)
		return
	}

	h.scannerService.RunNow()
	status := h.scannerService.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ScanFile scans a single file by path
// @Summary Scan a single file
// @Description Scan a specific file and return its status (orphan, conflict, or ok)
// @Tags admin,scanner
// @Accept json
// @Produce json
// @Param request body ScanFileRequest true "File path to scan"
// @Success 200 {object} services.SingleFileScanResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/scanner/scan-file [post]
func (h *ScannerHandler) ScanFile(w http.ResponseWriter, r *http.Request) {
	var req ScanFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	result, err := h.scannerService.ScanSingleFile(req.FilePath)
	if err != nil {
		http.Error(w, "Failed to scan file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// VerifyIntegrity verifies a photo's file integrity by comparing hashes
// @Summary Verify photo integrity
// @Description Verify that a photo's file hash matches the database hash
// @Tags admin,scanner
// @Produce json
// @Param photoId query string true "Photo ID to verify"
// @Success 200 {object} services.IntegrityResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security SessionAuth
// @Router /api/admin/scanner/verify [get]
func (h *ScannerHandler) VerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	photoID := r.URL.Query().Get("photoId")
	if photoID == "" {
		http.Error(w, "Photo ID is required", http.StatusBadRequest)
		return
	}

	result, err := h.scannerService.VerifyFileIntegrity(photoID)
	if err != nil {
		http.Error(w, "Failed to verify integrity: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ScanFileRequest is the request body for scanning a single file
type ScanFileRequest struct {
	FilePath string `json:"filePath"`
}

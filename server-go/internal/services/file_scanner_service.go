package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// ScanStatus represents the current status of the file scanner
type ScanStatus struct {
	Running          bool      `json:"running"`
	Enabled          bool      `json:"enabled"`
	LastRun          time.Time `json:"lastRun,omitempty"`
	LastRunDuration  string    `json:"lastRunDuration,omitempty"`
	FilesScanned     int       `json:"filesScanned"`
	OrphansFound     int       `json:"orphansFound"`
	ConflictsFound   int       `json:"conflictsFound"`
	Errors           []string  `json:"errors,omitempty"`
	Progress         float64   `json:"progress"`
	NextScheduledRun time.Time `json:"nextScheduledRun,omitempty"`
}

// FileScannerService handles background scanning for orphan files and conflicts
type FileScannerService struct {
	photoRepo        repository.PhotoRepo
	orphanFileRepo   repository.OrphanFileRepo
	fileConflictRepo repository.FileConflictRepo
	metadataService  *MetadataService
	hashService      *HashService
	storagePath      string
	intervalHours    int
	wsHub            *WebSocketHub

	mu       sync.RWMutex
	enabled  bool
	running  bool
	stopChan chan struct{}
	status   ScanStatus
	ticker   *time.Ticker
}

// NewFileScannerService creates a new FileScannerService
func NewFileScannerService(
	photoRepo repository.PhotoRepo,
	orphanFileRepo repository.OrphanFileRepo,
	fileConflictRepo repository.FileConflictRepo,
	metadataService *MetadataService,
	hashService *HashService,
	storagePath string,
	intervalHours int,
) *FileScannerService {
	if intervalHours <= 0 {
		intervalHours = 24 // Default to 24 hours
	}

	return &FileScannerService{
		photoRepo:        photoRepo,
		orphanFileRepo:   orphanFileRepo,
		fileConflictRepo: fileConflictRepo,
		metadataService:  metadataService,
		hashService:      hashService,
		storagePath:      storagePath,
		intervalHours:    intervalHours,
		stopChan:         make(chan struct{}),
		enabled:          true,
		status: ScanStatus{
			Enabled: true,
			Errors:  []string{},
		},
	}
}

// SetWebSocketHub sets the WebSocket hub for real-time notifications
func (s *FileScannerService) SetWebSocketHub(hub *WebSocketHub) {
	s.wsHub = hub
}

// notifyProgress sends scan progress update via WebSocket
func (s *FileScannerService) notifyProgress() {
	if s.wsHub == nil {
		return
	}

	s.mu.RLock()
	payload := ScannerProgressPayload{
		Running:        s.status.Running,
		FilesScanned:   s.status.FilesScanned,
		OrphansFound:   s.status.OrphansFound,
		ConflictsFound: s.status.ConflictsFound,
		Progress:       s.status.Progress,
	}
	s.mu.RUnlock()

	s.wsHub.BroadcastToTopic(TopicScanner, WSMessage{
		Type:    WSTypeScannerProgress,
		Payload: payload,
	})
}

// notifyScanComplete sends scan completion notification via WebSocket
func (s *FileScannerService) notifyScanComplete() {
	if s.wsHub == nil {
		return
	}

	s.mu.RLock()
	payload := ScannerProgressPayload{
		Running:        false,
		FilesScanned:   s.status.FilesScanned,
		OrphansFound:   s.status.OrphansFound,
		ConflictsFound: s.status.ConflictsFound,
		Progress:       100,
	}
	s.mu.RUnlock()

	s.wsHub.BroadcastToTopic(TopicScanner, WSMessage{
		Type:    WSTypeScannerComplete,
		Payload: payload,
	})
}

// Start begins the background scanning loop
func (s *FileScannerService) Start() {
	s.mu.Lock()
	if s.ticker != nil {
		s.mu.Unlock()
		return // Already started
	}
	s.enabled = true
	s.status.Enabled = true
	s.stopChan = make(chan struct{})
	interval := time.Duration(s.intervalHours) * time.Hour
	s.ticker = time.NewTicker(interval)
	s.status.NextScheduledRun = time.Now().Add(interval)
	s.mu.Unlock()

	log.Printf("File scanner service started (runs every %d hours)", s.intervalHours)

	// Run on schedule
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.mu.Lock()
				s.status.NextScheduledRun = time.Now().Add(time.Duration(s.intervalHours) * time.Hour)
				s.mu.Unlock()
				s.runScan()
			case <-s.stopChan:
				s.mu.Lock()
				s.ticker.Stop()
				s.ticker = nil
				s.mu.Unlock()
				log.Println("File scanner service stopped")
				return
			}
		}
	}()
}

// Stop stops the file scanner service
func (s *FileScannerService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker == nil {
		return // Already stopped
	}

	s.enabled = false
	s.status.Enabled = false
	close(s.stopChan)
}

// IsEnabled returns whether the scanner service is enabled
func (s *FileScannerService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// IsRunning returns whether a scan is currently in progress
func (s *FileScannerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus returns the current scanner status
func (s *FileScannerService) GetStatus() ScanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// RunNow triggers an immediate scan
func (s *FileScannerService) RunNow() {
	go s.runScan()
}

// runScan performs the actual file scan
func (s *FileScannerService) runScan() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("File scan already running, skipping")
		return
	}
	s.running = true
	s.status.Running = true
	s.status.FilesScanned = 0
	s.status.OrphansFound = 0
	s.status.ConflictsFound = 0
	s.status.Progress = 0
	s.status.Errors = []string{}
	s.mu.Unlock()

	startTime := time.Now()
	ctx := context.Background()
	log.Println("Starting file integrity scan...")

	// First pass: count total files
	totalFiles := 0
	filepath.Walk(s.storagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip .thumbs and hidden directories
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if s.isImageFile(info.Name()) {
			totalFiles++
		}
		return nil
	})

	// Second pass: scan files
	filesScanned := 0
	orphansFound := 0
	conflictsFound := 0
	var errors []string

	filepath.Walk(s.storagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, "Walk error: "+err.Error())
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// Skip .thumbs and hidden directories
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip non-image files
		if !s.isImageFile(info.Name()) {
			return nil
		}

		// Get relative path from storage base
		relPath, err := filepath.Rel(s.storagePath, path)
		if err != nil {
			errors = append(errors, "Path error for "+path+": "+err.Error())
			return nil
		}

		// Skip thumbnails
		if strings.Contains(relPath, ".thumbs") {
			return nil
		}

		// Process the file
		isOrphan, isConflict, scanErr := s.processFile(ctx, path, relPath, info)
		if scanErr != nil {
			errors = append(errors, "Scan error for "+relPath+": "+scanErr.Error())
		}
		if isOrphan {
			orphansFound++
		}
		if isConflict {
			conflictsFound++
		}

		filesScanned++

		// Update progress
		if totalFiles > 0 {
			s.mu.Lock()
			s.status.FilesScanned = filesScanned
			s.status.OrphansFound = orphansFound
			s.status.ConflictsFound = conflictsFound
			s.status.Progress = float64(filesScanned) / float64(totalFiles) * 100
			s.mu.Unlock()

			// Send progress update every 10 files or on every orphan/conflict
			if filesScanned%10 == 0 || isOrphan || isConflict {
				s.notifyProgress()
			}
		}

		return nil
	})

	duration := time.Since(startTime)

	s.mu.Lock()
	s.running = false
	s.status.Running = false
	s.status.LastRun = startTime
	s.status.LastRunDuration = duration.Round(time.Millisecond).String()
	s.status.FilesScanned = filesScanned
	s.status.OrphansFound = orphansFound
	s.status.ConflictsFound = conflictsFound
	s.status.Progress = 100
	s.status.Errors = errors
	s.mu.Unlock()

	log.Printf("File scan completed: %d files scanned, %d orphans found, %d conflicts found in %s",
		filesScanned, orphansFound, conflictsFound, duration.Round(time.Millisecond))

	if len(errors) > 0 {
		log.Printf("File scan encountered %d errors", len(errors))
	}

	// Send completion notification
	s.notifyScanComplete()
}

// processFile scans a single file and checks for orphans/conflicts
func (s *FileScannerService) processFile(ctx context.Context, fullPath, relPath string, info os.FileInfo) (isOrphan bool, isConflict bool, err error) {
	// Check if this path is already in orphan_files
	existingOrphan, err := s.orphanFileRepo.GetByPath(ctx, relPath)
	if err != nil {
		return false, false, err
	}
	if existingOrphan != nil {
		// Already tracked as orphan
		return false, false, nil
	}

	// Compute file hash
	file, err := os.Open(fullPath)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	fileHash, err := s.hashService.ComputeHash(file)
	if err != nil {
		return false, false, err
	}

	// Check if hash exists in photos table
	photo, err := s.photoRepo.GetByHash(ctx, fileHash)
	if err != nil {
		return false, false, err
	}

	if photo == nil {
		// File not in database - this is an orphan
		return s.createOrphanRecord(ctx, relPath, info.Size(), fileHash)
	}

	// File exists in database - check for conflicts
	return s.checkForConflicts(ctx, photo, relPath, fileHash)
}

// createOrphanRecord creates an orphan file record
func (s *FileScannerService) createOrphanRecord(ctx context.Context, relPath string, fileSize int64, fileHash string) (isOrphan bool, isConflict bool, err error) {
	orphan := models.NewOrphanFile(relPath, fileSize)
	orphan.FileHash = &fileHash

	// Try to read embedded metadata
	if s.metadataService != nil {
		metadata, metaErr := s.metadataService.ReadFullMetadata(relPath)
		if metaErr == nil && metadata != nil {
			if metadata.PhotoID != "" {
				orphan.EmbeddedPhotoID = &metadata.PhotoID
			}
			if metadata.UserID != "" {
				orphan.EmbeddedUserID = &metadata.UserID
			}
			if metadata.DeviceID != "" {
				orphan.EmbeddedDeviceID = &metadata.DeviceID
			}
			if metadata.FileHash != "" {
				orphan.EmbeddedFileHash = &metadata.FileHash
			}
			if !metadata.UploadedAt.IsZero() {
				orphan.EmbeddedUploadedAt = &metadata.UploadedAt
			}
		}
	}

	err = s.orphanFileRepo.Add(ctx, orphan)
	if err != nil {
		return false, false, err
	}

	return true, false, nil
}

// checkForConflicts checks if file metadata conflicts with database
func (s *FileScannerService) checkForConflicts(ctx context.Context, photo *models.Photo, relPath, fileHash string) (isOrphan bool, isConflict bool, err error) {
	if s.metadataService == nil {
		return false, false, nil
	}

	// Read embedded metadata
	metadata, err := s.metadataService.ReadFullMetadata(relPath)
	if err != nil {
		// Can't read metadata, no conflict detection possible
		return false, false, nil
	}
	if metadata == nil {
		// No embedded metadata, no conflicts
		return false, false, nil
	}

	conflictsCreated := false

	// Check for PhotoID mismatch
	if metadata.PhotoID != "" && metadata.PhotoID != photo.ID {
		conflict := models.NewFileConflict(photo.ID, relPath, models.ConflictTypePhotoIDMismatch)
		conflict.DBPhotoID = &photo.ID
		conflict.FilePhotoID = &metadata.PhotoID
		if photo.UserID != nil {
			conflict.DBUserID = photo.UserID
		}
		if photo.OriginDeviceID != nil {
			conflict.DBDeviceID = photo.OriginDeviceID
		}
		if metadata.UserID != "" {
			conflict.FileUserID = &metadata.UserID
		}
		if metadata.DeviceID != "" {
			conflict.FileDeviceID = &metadata.DeviceID
		}

		if addErr := s.fileConflictRepo.Add(ctx, conflict); addErr != nil {
			return false, false, addErr
		}
		conflictsCreated = true
	}

	// Check for UserID mismatch
	if metadata.UserID != "" && photo.UserID != nil && metadata.UserID != *photo.UserID {
		conflict := models.NewFileConflict(photo.ID, relPath, models.ConflictTypeUserIDMismatch)
		conflict.DBPhotoID = &photo.ID
		conflict.DBUserID = photo.UserID
		conflict.FileUserID = &metadata.UserID
		if metadata.PhotoID != "" {
			conflict.FilePhotoID = &metadata.PhotoID
		}
		if photo.OriginDeviceID != nil {
			conflict.DBDeviceID = photo.OriginDeviceID
		}
		if metadata.DeviceID != "" {
			conflict.FileDeviceID = &metadata.DeviceID
		}

		if addErr := s.fileConflictRepo.Add(ctx, conflict); addErr != nil {
			return false, false, addErr
		}
		conflictsCreated = true
	}

	// Check for DeviceID mismatch
	if metadata.DeviceID != "" && photo.OriginDeviceID != nil && metadata.DeviceID != *photo.OriginDeviceID {
		conflict := models.NewFileConflict(photo.ID, relPath, models.ConflictTypeDeviceIDMismatch)
		conflict.DBPhotoID = &photo.ID
		conflict.DBDeviceID = photo.OriginDeviceID
		conflict.FileDeviceID = &metadata.DeviceID
		if metadata.PhotoID != "" {
			conflict.FilePhotoID = &metadata.PhotoID
		}
		if photo.UserID != nil {
			conflict.DBUserID = photo.UserID
		}
		if metadata.UserID != "" {
			conflict.FileUserID = &metadata.UserID
		}

		if addErr := s.fileConflictRepo.Add(ctx, conflict); addErr != nil {
			return false, false, addErr
		}
		conflictsCreated = true
	}

	return false, conflictsCreated, nil
}

// isImageFile checks if a filename has an image extension
func (s *FileScannerService) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".heic": true,
		".heif": true,
		".tiff": true,
		".tif":  true,
		".bmp":  true,
		".raw":  true,
		".cr2":  true,
		".nef":  true,
		".arw":  true,
		".dng":  true,
	}
	return imageExtensions[ext]
}

// ScanSingleFile scans a single file by its relative path and returns the result
func (s *FileScannerService) ScanSingleFile(relPath string) (*SingleFileScanResult, error) {
	ctx := context.Background()
	fullPath := filepath.Join(s.storagePath, relPath)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	result := &SingleFileScanResult{
		FilePath: relPath,
		FileSize: info.Size(),
	}

	// Compute hash
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileHash, err := s.hashService.ComputeHash(file)
	if err != nil {
		return nil, err
	}
	result.FileHash = fileHash

	// Read embedded metadata
	if s.metadataService != nil {
		metadata, err := s.metadataService.ReadFullMetadata(relPath)
		if err == nil && metadata != nil {
			result.EmbeddedMetadata = metadata
		}
	}

	// Check if file exists in database by hash
	photo, err := s.photoRepo.GetByHash(ctx, fileHash)
	if err != nil {
		return nil, err
	}

	if photo == nil {
		result.Status = "orphan"
		result.Message = "File not found in database"
		return result, nil
	}

	result.PhotoID = &photo.ID
	result.DBUserID = photo.UserID
	result.DBDeviceID = photo.OriginDeviceID

	// Check for conflicts
	if result.EmbeddedMetadata != nil {
		conflicts := []string{}

		if result.EmbeddedMetadata.PhotoID != "" && result.EmbeddedMetadata.PhotoID != photo.ID {
			conflicts = append(conflicts, "photo_id_mismatch")
		}
		if result.EmbeddedMetadata.UserID != "" && photo.UserID != nil && result.EmbeddedMetadata.UserID != *photo.UserID {
			conflicts = append(conflicts, "user_id_mismatch")
		}
		if result.EmbeddedMetadata.DeviceID != "" && photo.OriginDeviceID != nil && result.EmbeddedMetadata.DeviceID != *photo.OriginDeviceID {
			conflicts = append(conflicts, "device_id_mismatch")
		}
		if result.EmbeddedMetadata.FileHash != "" && result.EmbeddedMetadata.FileHash != fileHash {
			conflicts = append(conflicts, "hash_mismatch")
		}

		if len(conflicts) > 0 {
			result.Status = "conflict"
			result.Conflicts = conflicts
			result.Message = fmt.Sprintf("Found %d conflict(s)", len(conflicts))
		} else {
			result.Status = "ok"
			result.Message = "File matches database record"
		}
	} else {
		result.Status = "ok"
		result.Message = "File found in database (no embedded metadata to verify)"
	}

	return result, nil
}

// VerifyFileIntegrity checks if a photo's file hash matches the database hash
func (s *FileScannerService) VerifyFileIntegrity(photoID string) (*IntegrityResult, error) {
	ctx := context.Background()

	// Get photo from database
	photo, err := s.photoRepo.GetByID(ctx, photoID)
	if err != nil {
		return nil, err
	}
	if photo == nil {
		return nil, fmt.Errorf("photo not found")
	}

	fullPath := filepath.Join(s.storagePath, photo.StoredPath)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &IntegrityResult{
				PhotoID:      photoID,
				StoredPath:   photo.StoredPath,
				Status:       "missing",
				Message:      "File does not exist on disk",
				DBHash:       photo.FileHash,
				FileExists:   false,
				HashMatches:  false,
			}, nil
		}
		return nil, err
	}

	result := &IntegrityResult{
		PhotoID:     photoID,
		StoredPath:  photo.StoredPath,
		FileSize:    info.Size(),
		DBHash:      photo.FileHash,
		FileExists:  true,
	}

	// Compute current file hash
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	currentHash, err := s.hashService.ComputeHash(file)
	if err != nil {
		return nil, err
	}
	result.CurrentHash = currentHash

	// Compare hashes
	if currentHash == photo.FileHash {
		result.Status = "ok"
		result.Message = "File hash matches database"
		result.HashMatches = true
	} else {
		result.Status = "corrupted"
		result.Message = "File hash does not match database - file may be corrupted or modified"
		result.HashMatches = false
	}

	return result, nil
}

// SingleFileScanResult contains the result of scanning a single file
type SingleFileScanResult struct {
	FilePath         string         `json:"filePath"`
	FileSize         int64          `json:"fileSize"`
	FileHash         string         `json:"fileHash"`
	Status           string         `json:"status"` // ok, orphan, conflict
	Message          string         `json:"message"`
	PhotoID          *string        `json:"photoId,omitempty"`
	DBUserID         *string        `json:"dbUserId,omitempty"`
	DBDeviceID       *string        `json:"dbDeviceId,omitempty"`
	EmbeddedMetadata *PhotoMetadata `json:"embeddedMetadata,omitempty"`
	Conflicts        []string       `json:"conflicts,omitempty"`
}

// IntegrityResult contains the result of verifying a photo's integrity
type IntegrityResult struct {
	PhotoID     string `json:"photoId"`
	StoredPath  string `json:"storedPath"`
	FileSize    int64  `json:"fileSize,omitempty"`
	DBHash      string `json:"dbHash"`
	CurrentHash string `json:"currentHash,omitempty"`
	Status      string `json:"status"` // ok, corrupted, missing
	Message     string `json:"message"`
	FileExists  bool   `json:"fileExists"`
	HashMatches bool   `json:"hashMatches"`
}

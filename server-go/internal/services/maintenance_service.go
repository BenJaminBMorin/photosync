package services

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/photosync/server/internal/repository"
)

// MaintenanceStatus represents the current status of maintenance tasks
type MaintenanceStatus struct {
	Running           bool      `json:"running"`
	Enabled           bool      `json:"enabled"`
	LastRun           time.Time `json:"lastRun,omitempty"`
	LastRunDuration   string    `json:"lastRunDuration,omitempty"`
	OrphansRemoved    int       `json:"orphansRemoved"`
	ThumbsGenerated   int       `json:"thumbsGenerated"`
	Errors            []string  `json:"errors,omitempty"`
	NextScheduledRun  time.Time `json:"nextScheduledRun,omitempty"`
}

// MaintenanceService handles background maintenance tasks
type MaintenanceService struct {
	photoRepo        repository.PhotoRepo
	thumbnailService *ThumbnailService
	storagePath      string

	mu         sync.RWMutex
	enabled    bool
	running    bool
	stopChan   chan struct{}
	status     MaintenanceStatus
	ticker     *time.Ticker
}

// NewMaintenanceService creates a new MaintenanceService
func NewMaintenanceService(
	photoRepo repository.PhotoRepo,
	thumbnailService *ThumbnailService,
	storagePath string,
) *MaintenanceService {
	return &MaintenanceService{
		photoRepo:        photoRepo,
		thumbnailService: thumbnailService,
		storagePath:      storagePath,
		stopChan:         make(chan struct{}),
		enabled:          true,
		status: MaintenanceStatus{
			Enabled: true,
			Errors:  []string{},
		},
	}
}

// Start begins the background maintenance loop
func (s *MaintenanceService) Start() {
	s.mu.Lock()
	if s.ticker != nil {
		s.mu.Unlock()
		return // Already started
	}
	s.enabled = true
	s.status.Enabled = true
	s.stopChan = make(chan struct{})
	s.ticker = time.NewTicker(1 * time.Hour)
	s.status.NextScheduledRun = time.Now().Add(1 * time.Hour)
	s.mu.Unlock()

	log.Println("Maintenance service started (runs every hour)")

	// Run immediately on startup
	go s.runMaintenance()

	// Then run every hour
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.mu.Lock()
				s.status.NextScheduledRun = time.Now().Add(1 * time.Hour)
				s.mu.Unlock()
				s.runMaintenance()
			case <-s.stopChan:
				s.mu.Lock()
				s.ticker.Stop()
				s.ticker = nil
				s.mu.Unlock()
				log.Println("Maintenance service stopped")
				return
			}
		}
	}()
}

// Stop stops the maintenance service
func (s *MaintenanceService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker == nil {
		return // Already stopped
	}

	s.enabled = false
	s.status.Enabled = false
	close(s.stopChan)
}

// IsEnabled returns whether the maintenance service is enabled
func (s *MaintenanceService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// GetStatus returns the current maintenance status
func (s *MaintenanceService) GetStatus() MaintenanceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// RunNow triggers an immediate maintenance run
func (s *MaintenanceService) RunNow() {
	go s.runMaintenance()
}

// runMaintenance performs all maintenance tasks
func (s *MaintenanceService) runMaintenance() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("Maintenance already running, skipping")
		return
	}
	s.running = true
	s.status.Running = true
	s.status.Errors = []string{}
	s.mu.Unlock()

	startTime := time.Now()
	ctx := context.Background()
	log.Println("Running maintenance tasks...")

	// Task 1: Clean up orphaned photos (no owner)
	orphansRemoved, orphanErrors := s.cleanupOrphanedPhotos(ctx)

	// Task 2: Generate missing thumbnails
	thumbsGenerated, thumbErrors := s.generateMissingThumbnails(ctx)

	duration := time.Since(startTime)

	s.mu.Lock()
	s.running = false
	s.status.Running = false
	s.status.LastRun = startTime
	s.status.LastRunDuration = duration.Round(time.Millisecond).String()
	s.status.OrphansRemoved = orphansRemoved
	s.status.ThumbsGenerated = thumbsGenerated
	s.status.Errors = append(orphanErrors, thumbErrors...)
	s.mu.Unlock()

	if orphansRemoved > 0 {
		log.Printf("Maintenance: Removed %d orphaned photos", orphansRemoved)
	}
	if thumbsGenerated > 0 {
		log.Printf("Maintenance: Generated thumbnails for %d photos", thumbsGenerated)
	}
	if len(orphanErrors) > 0 || len(thumbErrors) > 0 {
		log.Printf("Maintenance: Completed with %d errors", len(orphanErrors)+len(thumbErrors))
	}

	log.Printf("Maintenance tasks completed in %s", duration.Round(time.Millisecond))
}

// cleanupOrphanedPhotos removes photos without an owner
func (s *MaintenanceService) cleanupOrphanedPhotos(ctx context.Context) (int, []string) {
	var errors []string

	// Get orphaned photos (user_id IS NULL)
	photos, err := s.photoRepo.GetOrphanedPhotos(ctx, 100)
	if err != nil {
		errMsg := "Failed to get orphaned photos: " + err.Error()
		log.Printf("Maintenance: %s", errMsg)
		return 0, []string{errMsg}
	}

	removed := 0
	for _, photo := range photos {
		// Delete the actual file
		filePath := filepath.Join(s.storagePath, photo.StoredPath)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			errMsg := "Failed to delete file " + photo.StoredPath + ": " + err.Error()
			log.Printf("Maintenance: %s", errMsg)
			errors = append(errors, errMsg)
		}

		// Delete thumbnails if they exist
		if photo.ThumbSmall != nil {
			os.Remove(filepath.Join(s.storagePath, *photo.ThumbSmall))
		}
		if photo.ThumbMedium != nil {
			os.Remove(filepath.Join(s.storagePath, *photo.ThumbMedium))
		}
		if photo.ThumbLarge != nil {
			os.Remove(filepath.Join(s.storagePath, *photo.ThumbLarge))
		}

		// Delete from database
		if _, err := s.photoRepo.Delete(ctx, photo.ID); err != nil {
			errMsg := "Failed to delete photo " + photo.ID + " from DB: " + err.Error()
			log.Printf("Maintenance: %s", errMsg)
			errors = append(errors, errMsg)
			continue
		}

		removed++
	}

	return removed, errors
}

// generateMissingThumbnails generates thumbnails for photos that don't have them
func (s *MaintenanceService) generateMissingThumbnails(ctx context.Context) (int, []string) {
	var errors []string

	// Process in batches
	batchSize := 50
	totalGenerated := 0
	maxIterations := 100 // Safety limit: max 5000 photos per run

	for i := 0; i < maxIterations; i++ {
		photos, err := s.photoRepo.GetPhotosWithoutThumbnails(ctx, batchSize)
		if err != nil {
			errMsg := "Failed to get photos without thumbnails: " + err.Error()
			log.Printf("Maintenance: %s", errMsg)
			errors = append(errors, errMsg)
			break
		}

		if len(photos) == 0 {
			break
		}

		for _, photo := range photos {
			// Skip unsupported formats silently
			if !IsSupportedFormat(photo.StoredPath) {
				continue
			}

			result, err := s.thumbnailService.RegenerateThumbnailsFromFile(photo.ID, photo.StoredPath)
			if err != nil {
				// Log errors but don't add to errors list (too noisy)
				log.Printf("Maintenance: Failed to generate thumbnails for %s: %v", photo.ID, err)
				continue
			}

			// Update database
			if err := s.photoRepo.UpdateThumbnails(ctx, photo.ID, result.SmallPath, result.MediumPath, result.LargePath); err != nil {
				errMsg := "Failed to update thumbnails in DB for " + photo.ID + ": " + err.Error()
				log.Printf("Maintenance: %s", errMsg)
				errors = append(errors, errMsg)
				continue
			}

			totalGenerated++
		}

		// Small delay between batches to avoid overloading
		time.Sleep(100 * time.Millisecond)
	}

	return totalGenerated, errors
}

package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// DeleteService orchestrates photo deletion approval flow
type DeleteService struct {
	userRepo          repository.UserRepo
	deviceRepo        repository.DeviceRepo
	deleteRequestRepo *repository.DeleteRequestRepository
	photoRepo         repository.PhotoRepo
	fcmService        *FCMService
	deleteTimeout     int // seconds
}

// NewDeleteService creates a new DeleteService
func NewDeleteService(
	userRepo repository.UserRepo,
	deviceRepo repository.DeviceRepo,
	deleteRequestRepo *repository.DeleteRequestRepository,
	photoRepo repository.PhotoRepo,
	fcmService *FCMService,
	deleteTimeout int,
) *DeleteService {
	if deleteTimeout <= 0 {
		deleteTimeout = 60
	}
	return &DeleteService{
		userRepo:          userRepo,
		deviceRepo:        deviceRepo,
		deleteRequestRepo: deleteRequestRepo,
		photoRepo:         photoRepo,
		fcmService:        fcmService,
		deleteTimeout:     deleteTimeout,
	}
}

// InitiateDeleteResult contains the result of initiating deletion
type InitiateDeleteResult struct {
	RequestID string `json:"requestId"`
	ExpiresAt string `json:"expiresAt"`
}

// InitiateDelete starts the push notification delete approval flow
func (s *DeleteService) InitiateDelete(ctx context.Context, userID string, photoIDs []string, ipAddress, userAgent string) (*InitiateDeleteResult, error) {
	// Verify user exists and is active
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Get user's active devices
	devices, err := s.deviceRepo.GetActiveForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no registered devices found")
	}

	// Create delete request
	deleteReq := models.NewDeleteRequest(userID, photoIDs, ipAddress, userAgent, s.deleteTimeout)
	if err := s.deleteRequestRepo.Add(ctx, deleteReq); err != nil {
		return nil, fmt.Errorf("failed to create delete request: %w", err)
	}

	// Send push notifications to all devices
	if s.fcmService != nil {
		tokens := make([]string, 0, len(devices))
		for _, d := range devices {
			tokens = append(tokens, d.FCMToken)
		}

		notification := DeleteRequestNotification{
			RequestID: deleteReq.ID,
			PhotoIDs:  photoIDs,
			Email:     user.Email,
			IPAddress: ipAddress,
			UserAgent: userAgent,
		}

		sent, _ := s.SendDeleteRequestToMultiple(ctx, tokens, notification)
		if sent == 0 {
			return nil, fmt.Errorf("failed to send push notification to any device")
		}
	}

	return &InitiateDeleteResult{
		RequestID: deleteReq.ID,
		ExpiresAt: deleteReq.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// CheckDeleteStatus checks the status of a delete request
func (s *DeleteService) CheckDeleteStatus(ctx context.Context, requestID string) (*models.DeleteStatusResponse, error) {
	deleteReq, err := s.deleteRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delete request: %w", err)
	}
	if deleteReq == nil {
		return nil, models.ErrDeleteRequestNotFound
	}

	// Check if expired
	if deleteReq.IsExpired() && deleteReq.Status == models.DeleteStatusPending {
		deleteReq.Status = models.DeleteStatusExpired
		s.deleteRequestRepo.Update(ctx, deleteReq)
	}

	response := &models.DeleteStatusResponse{
		Status:    deleteReq.Status,
		ExpiresAt: deleteReq.ExpiresAt,
	}

	return response, nil
}

// RespondToDelete handles approve/deny from mobile app
func (s *DeleteService) RespondToDelete(ctx context.Context, requestID string, approved bool, deviceID string) error {
	fmt.Printf("DEBUG: RespondToDelete called - requestID: %s, approved: %v, deviceID: %s\n", requestID, approved, deviceID)

	deleteReq, err := s.deleteRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		fmt.Printf("ERROR: Failed to get delete request: %v\n", err)
		return fmt.Errorf("failed to get delete request: %w", err)
	}
	if deleteReq == nil {
		fmt.Printf("ERROR: Delete request not found: %s\n", requestID)
		return models.ErrDeleteRequestNotFound
	}

	fmt.Printf("DEBUG: Delete request found - status: %s, photoCount: %d\n", deleteReq.Status, len(deleteReq.PhotoIDs))

	if deleteReq.Status != models.DeleteStatusPending {
		fmt.Printf("ERROR: Delete request already resolved - status: %s\n", deleteReq.Status)
		return models.ErrDeleteAlreadyResolved
	}

	if deleteReq.IsExpired() {
		fmt.Printf("ERROR: Delete request expired\n")
		deleteReq.Status = models.DeleteStatusExpired
		s.deleteRequestRepo.Update(ctx, deleteReq)
		return models.ErrDeleteRequestExpired
	}

	if approved {
		fmt.Printf("DEBUG: Approving delete request - deleting %d photos\n", len(deleteReq.PhotoIDs))
		deleteReq.Approve(deviceID)

		// Actually delete the photos
		for _, photoID := range deleteReq.PhotoIDs {
			// TODO: Also delete physical files from storage
			if _, err := s.photoRepo.Delete(ctx, photoID); err != nil {
				// Log error but continue with other photos
				fmt.Printf("ERROR: Failed to delete photo %s: %v\n", photoID, err)
			} else {
				fmt.Printf("DEBUG: Deleted photo %s from database\n", photoID)
			}
		}
	} else {
		fmt.Printf("DEBUG: Denying delete request\n")
		deleteReq.Deny(deviceID)
	}

	fmt.Printf("DEBUG: Updating delete request with status: %s\n", deleteReq.Status)
	if err := s.deleteRequestRepo.Update(ctx, deleteReq); err != nil {
		fmt.Printf("ERROR: Failed to update delete request: %v\n", err)
		return fmt.Errorf("failed to update delete request: %w", err)
	}

	fmt.Printf("DEBUG: Delete request updated successfully\n")
	return nil
}

// DeleteRequestNotification contains data for delete request push notification
type DeleteRequestNotification struct {
	RequestID string
	PhotoIDs  []string
	Email     string
	IPAddress string
	UserAgent string
}

// SendDeleteRequestToMultiple sends delete request notification to multiple devices
func (s *DeleteService) SendDeleteRequestToMultiple(ctx context.Context, tokens []string, notification DeleteRequestNotification) (int, error) {
	if s.fcmService == nil {
		return 0, fmt.Errorf("FCM service not initialized")
	}

	if len(tokens) == 0 {
		return 0, nil
	}

	successCount := 0
	for _, token := range tokens {
		if err := s.fcmService.SendDataNotification(ctx, token, "Photo Deletion Request",
			fmt.Sprintf("Request to delete %d photo(s) from web interface", len(notification.PhotoIDs)),
			map[string]string{
				"type":      "delete_request",
				"requestId": notification.RequestID,
				"photoIds":  strings.Join(notification.PhotoIDs, ","),
				"email":     notification.Email,
				"ipAddress": notification.IPAddress,
				"userAgent": notification.UserAgent,
			}); err != nil {
			fmt.Printf("FCM send failed for token %s...: %v\n", token[:min(20, len(token))], err)
			continue
		}
		successCount++
	}

	return successCount, nil
}


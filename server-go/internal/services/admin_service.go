package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// AdminService provides admin functionality
type AdminService struct {
	userRepo           repository.UserRepo
	deviceRepo         repository.DeviceRepo
	sessionRepo        repository.WebSessionRepo
	photoRepo          repository.PhotoRepo
	setupRepo          repository.SetupConfigRepo
	storageBasePath    string
	startTime          time.Time
	buildVersion       string
	buildDate          string
	containerBuildDate string
}

// NewAdminService creates a new AdminService
func NewAdminService(
	userRepo repository.UserRepo,
	deviceRepo repository.DeviceRepo,
	sessionRepo repository.WebSessionRepo,
	photoRepo repository.PhotoRepo,
	setupRepo repository.SetupConfigRepo,
	storageBasePath string,
	buildVersion string,
	buildDate string,
	containerBuildDate string,
) *AdminService {
	return &AdminService{
		userRepo:           userRepo,
		deviceRepo:         deviceRepo,
		sessionRepo:        sessionRepo,
		photoRepo:          photoRepo,
		setupRepo:          setupRepo,
		storageBasePath:    storageBasePath,
		startTime:          time.Now(),
		buildVersion:       buildVersion,
		buildDate:          buildDate,
		containerBuildDate: containerBuildDate,
	}
}

// ListUsers returns all users with statistics
func (s *AdminService) ListUsers(ctx context.Context) (*models.UserListResponse, error) {
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]models.AdminUserResponse, 0, len(users))
	for _, u := range users {
		devices, _ := s.deviceRepo.GetAllForUser(ctx, u.ID)
		sessions, _ := s.sessionRepo.GetActiveForUser(ctx, u.ID)
		photoCount, _ := s.photoRepo.GetCountForUser(ctx, u.ID)

		responses = append(responses, models.AdminUserResponse{
			ID:           u.ID,
			Email:        u.Email,
			DisplayName:  u.DisplayName,
			IsAdmin:      u.IsAdmin,
			IsActive:     u.IsActive,
			CreatedAt:    u.CreatedAt,
			DeviceCount:  len(devices),
			SessionCount: len(sessions),
			PhotoCount:   photoCount,
		})
	}

	return &models.UserListResponse{
		Users:      responses,
		TotalCount: len(responses),
	}, nil
}

// GetUser returns a single user with statistics
func (s *AdminService) GetUser(ctx context.Context, userID string) (*models.AdminUserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, models.ErrUserNotFound
	}

	devices, _ := s.deviceRepo.GetAllForUser(ctx, userID)
	sessions, _ := s.sessionRepo.GetActiveForUser(ctx, userID)
	photoCount, _ := s.photoRepo.GetCountForUser(ctx, userID)

	return &models.AdminUserResponse{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		IsAdmin:      user.IsAdmin,
		IsActive:     user.IsActive,
		CreatedAt:    user.CreatedAt,
		DeviceCount:  len(devices),
		SessionCount: len(sessions),
		PhotoCount:   photoCount,
	}, nil
}

// CreateUser creates a new user and returns the user with API key
func (s *AdminService) CreateUser(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	// Check if email exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, models.ErrEmailExists
	}

	user, err := models.NewUser(req.Email, req.DisplayName, req.IsAdmin)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.Add(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser updates a user's details
func (s *AdminService) UpdateUser(ctx context.Context, userID string, req models.UpdateUserRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return models.ErrUserNotFound
	}

	// Check if email is changing and already exists
	if req.Email != user.Email {
		existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
		if existing != nil && existing.ID != userID {
			return models.ErrEmailExists
		}
	}

	user.Email = req.Email
	user.DisplayName = req.DisplayName
	user.IsAdmin = req.IsAdmin
	user.IsActive = req.IsActive

	return s.userRepo.Update(ctx, user)
}

// DeleteUser deletes a user (prevents self-deletion)
func (s *AdminService) DeleteUser(ctx context.Context, userID, adminUserID string) error {
	// Prevent self-deletion
	if userID == adminUserID {
		return fmt.Errorf("cannot delete your own account")
	}

	// Invalidate all sessions for the user
	s.sessionRepo.InvalidateAllForUser(ctx, userID)

	deleted, err := s.userRepo.Delete(ctx, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return models.ErrUserNotFound
	}
	return nil
}

// ResetAPIKey generates a new API key for a user
func (s *AdminService) ResetAPIKey(ctx context.Context, userID string) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", models.ErrUserNotFound
	}

	// Generate new API key
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	newAPIKey := hex.EncodeToString(bytes)
	newAPIKeyHash := models.HashAPIKey(newAPIKey)

	// Update user's API key hash
	if err := s.userRepo.UpdateAPIKeyHash(ctx, userID, newAPIKeyHash); err != nil {
		return "", fmt.Errorf("failed to update API key: %w", err)
	}

	// Invalidate all existing sessions (they're using the old key context)
	s.sessionRepo.InvalidateAllForUser(ctx, userID)

	return newAPIKey, nil
}

// GetUserDevices returns all devices for a user
func (s *AdminService) GetUserDevices(ctx context.Context, userID string) ([]*models.Device, error) {
	return s.deviceRepo.GetAllForUser(ctx, userID)
}

// DeleteDevice removes a device
func (s *AdminService) DeleteDevice(ctx context.Context, deviceID string) error {
	deleted, err := s.deviceRepo.Delete(ctx, deviceID)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("device not found")
	}
	return nil
}

// GetUserSessions returns all active sessions for a user
func (s *AdminService) GetUserSessions(ctx context.Context, userID string) ([]*models.WebSession, error) {
	return s.sessionRepo.GetActiveForUser(ctx, userID)
}

// InvalidateSession ends a specific session
func (s *AdminService) InvalidateSession(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Invalidate(ctx, sessionID)
}

// GetSystemStatus returns system health and statistics
func (s *AdminService) GetSystemStatus(ctx context.Context) (*models.SystemStatusResponse, error) {
	userCount, _ := s.userRepo.GetCount(ctx)
	photoCount, _ := s.photoRepo.GetCount(ctx)

	// Count all devices (sum across all users)
	users, _ := s.userRepo.GetAll(ctx)
	deviceCount := 0
	sessionCount := 0
	for _, u := range users {
		devices, _ := s.deviceRepo.GetAllForUser(ctx, u.ID)
		sessions, _ := s.sessionRepo.GetActiveForUser(ctx, u.ID)
		deviceCount += len(devices)
		sessionCount += len(sessions)
	}

	// Calculate storage size
	var totalSize int64
	filepath.Walk(s.storageBasePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Check Firebase configuration
	firebaseConfigured := false
	projectID := ""
	configPath := filepath.Join(s.storageBasePath, ".config", "firebase-service-account.json")
	if _, err := os.Stat(configPath); err == nil {
		firebaseConfigured = true
		// Could parse the JSON to get project ID if needed
	}

	// Calculate uptime
	uptime := time.Since(s.startTime)
	uptimeStr := fmt.Sprintf("%dd %dh %dm", int(uptime.Hours()/24), int(uptime.Hours())%24, int(uptime.Minutes())%60)

	// Determine database type
	dbType := "postgres"
	if os.Getenv("DATABASE_URL") == "" {
		dbType = "sqlite"
	}

	return &models.SystemStatusResponse{
		Version:            "2.0",
		BuildVersion:       s.buildVersion,
		BuildDate:          s.buildDate,
		ServerStartTime:    s.startTime.Format(time.RFC3339),
		ContainerBuildDate: s.containerBuildDate,
		Uptime:             uptimeStr,
		Database: models.DatabaseStatus{
			Type:      dbType,
			Connected: true,
		},
		Storage: models.StorageStatus{
			BasePath:    s.storageBasePath,
			TotalPhotos: int64(photoCount),
			TotalSizeMB: totalSize / (1024 * 1024),
		},
		Firebase: models.FirebaseStatus{
			Configured: firebaseConfigured,
			ProjectID:  projectID,
		},
		Stats: models.SystemStats{
			TotalUsers:     userCount,
			TotalPhotos:    photoCount,
			TotalDevices:   deviceCount,
			ActiveSessions: sessionCount,
		},
	}, nil
}

// GetSystemConfig returns current system configuration
func (s *AdminService) GetSystemConfig(ctx context.Context) (*models.SystemConfigResponse, error) {
	setupComplete, _ := s.setupRepo.IsSetupComplete(ctx)
	firebaseConfigured, _ := s.setupRepo.Get(ctx, "firebase_configured")

	return &models.SystemConfigResponse{
		SetupComplete:      setupComplete,
		FirebaseConfigured: firebaseConfigured == "true",
		PhotoStoragePath:   s.storageBasePath,
		MaxFileSizeMB:      50,
		AllowedExtensions:  []string{"jpg", "jpeg", "png", "gif", "webp", "heic", "heif"},
	}, nil
}

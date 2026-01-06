package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// SetupStatus represents the current setup state
type SetupStatus struct {
	IsComplete         bool `json:"isComplete"`
	DatabaseConfigured bool `json:"databaseConfigured"`
	FirebaseConfigured bool `json:"firebaseConfigured"`
	AdminCreated       bool `json:"adminCreated"`
}

// FirebaseConfig represents Firebase configuration
type FirebaseConfig struct {
	ProjectID       string `json:"project_id"`
	CredentialsPath string `json:"credentialsPath"`
}

// SetupService handles first-run setup operations
type SetupService struct {
	setupRepo   repository.SetupConfigRepo
	userRepo    repository.UserRepo
	configDir   string
}

// NewSetupService creates a new SetupService
func NewSetupService(setupRepo repository.SetupConfigRepo, userRepo repository.UserRepo, configDir string) *SetupService {
	return &SetupService{
		setupRepo:   setupRepo,
		userRepo:    userRepo,
		configDir:   configDir,
	}
}

// GetStatus returns the current setup status
func (s *SetupService) GetStatus(ctx context.Context) (*SetupStatus, error) {
	status := &SetupStatus{
		DatabaseConfigured: true, // If we got here, database is working
	}

	// Check if setup is marked complete
	complete, err := s.setupRepo.IsSetupComplete(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check setup status: %w", err)
	}
	status.IsComplete = complete

	// Check Firebase configuration
	firebaseValue, err := s.setupRepo.Get(ctx, repository.SetupKeyFirebaseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to check firebase config: %w", err)
	}
	status.FirebaseConfigured = firebaseValue == "true"

	// Check if admin user exists
	adminValue, err := s.setupRepo.Get(ctx, repository.SetupKeyAdminCreated)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}
	status.AdminCreated = adminValue == "true"

	return status, nil
}

// SaveFirebaseCredentials saves the uploaded Firebase service account JSON
func (s *SetupService) SaveFirebaseCredentials(ctx context.Context, reader io.Reader) (*FirebaseConfig, error) {
	// Read the JSON content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Parse to validate and extract project ID
	var creds struct {
		ProjectID string `json:"project_id"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(content, &creds); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	if creds.Type != "service_account" {
		return nil, fmt.Errorf("invalid credentials: expected service_account type, got %s", creds.Type)
	}

	if creds.ProjectID == "" {
		return nil, fmt.Errorf("invalid credentials: project_id is missing")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(s.configDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save credentials file
	credentialsPath := filepath.Join(s.configDir, "firebase-service-account.json")
	if err := os.WriteFile(credentialsPath, content, 0600); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	// Mark Firebase as configured in database
	if err := s.setupRepo.Set(ctx, repository.SetupKeyFirebaseConfig, "true"); err != nil {
		return nil, fmt.Errorf("failed to update setup config: %w", err)
	}

	return &FirebaseConfig{
		ProjectID:       creds.ProjectID,
		CredentialsPath: credentialsPath,
	}, nil
}

// GetFirebaseCredentialsPath returns the path to Firebase credentials if configured
func (s *SetupService) GetFirebaseCredentialsPath() string {
	path := filepath.Join(s.configDir, "firebase-service-account.json")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// CreateAdminRequest represents the request to create admin user
type CreateAdminRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

// CreateAdminResponse contains the new admin user and their API key
type CreateAdminResponse struct {
	User   models.UserResponse `json:"user"`
	APIKey string              `json:"apiKey"` // Only shown once!
}

// CreateAdminUser creates the first admin user
func (s *SetupService) CreateAdminUser(ctx context.Context, req CreateAdminRequest) (*CreateAdminResponse, error) {
	// Check if admin already exists
	adminValue, err := s.setupRepo.Get(ctx, repository.SetupKeyAdminCreated)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}
	if adminValue == "true" {
		return nil, fmt.Errorf("admin user already exists")
	}

	// Create admin user with generated API key
	user, err := models.NewUser(req.Email, req.DisplayName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Store the API key before it gets cleared
	apiKey := user.APIKey

	// Save to database
	if err := s.userRepo.Add(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// Mark admin as created
	if err := s.setupRepo.Set(ctx, repository.SetupKeyAdminCreated, "true"); err != nil {
		return nil, fmt.Errorf("failed to update setup config: %w", err)
	}

	return &CreateAdminResponse{
		User:   user.ToResponse(),
		APIKey: apiKey,
	}, nil
}

// CompleteSetup marks the setup as complete
func (s *SetupService) CompleteSetup(ctx context.Context) error {
	// Verify all prerequisites are met
	status, err := s.GetStatus(ctx)
	if err != nil {
		return err
	}

	if !status.FirebaseConfigured {
		return fmt.Errorf("firebase must be configured before completing setup")
	}
	if !status.AdminCreated {
		return fmt.Errorf("admin user must be created before completing setup")
	}

	// Mark setup as complete
	if err := s.setupRepo.Set(ctx, repository.SetupKeyComplete, "true"); err != nil {
		return fmt.Errorf("failed to mark setup as complete: %w", err)
	}

	return nil
}

// IsSetupRequired returns true if setup wizard should be shown
func (s *SetupService) IsSetupRequired(ctx context.Context) (bool, error) {
	complete, err := s.setupRepo.IsSetupComplete(ctx)
	if err != nil {
		return true, err
	}
	return !complete, nil
}

package services

import (
	"context"
	"fmt"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// AuthService orchestrates web authentication flow
type AuthService struct {
	userRepo        repository.UserRepo
	deviceRepo      repository.DeviceRepo
	authRequestRepo repository.AuthRequestRepo
	sessionRepo     repository.WebSessionRepo
	fcmService      *FCMService
	authTimeout     int // seconds
	sessionDuration int // hours
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userRepo repository.UserRepo,
	deviceRepo repository.DeviceRepo,
	authRequestRepo repository.AuthRequestRepo,
	sessionRepo repository.WebSessionRepo,
	fcmService *FCMService,
	authTimeout int,
	sessionDuration int,
) *AuthService {
	if authTimeout <= 0 {
		authTimeout = 60
	}
	if sessionDuration <= 0 {
		sessionDuration = 24
	}
	return &AuthService{
		userRepo:        userRepo,
		deviceRepo:      deviceRepo,
		authRequestRepo: authRequestRepo,
		sessionRepo:     sessionRepo,
		fcmService:      fcmService,
		authTimeout:     authTimeout,
		sessionDuration: sessionDuration,
	}
}

// InitiateAuthResult contains the result of initiating auth
type InitiateAuthResult struct {
	RequestID string `json:"requestId"`
	ExpiresAt string `json:"expiresAt"`
}

// InitiateAuth starts the push notification auth flow
func (s *AuthService) InitiateAuth(ctx context.Context, email, ipAddress, userAgent string) (*InitiateAuthResult, error) {
	// Look up user
	user, err := s.userRepo.GetByEmail(ctx, email)
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
	devices, err := s.deviceRepo.GetActiveForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no registered devices found")
	}

	// Create auth request
	authReq := models.NewAuthRequest(user.ID, ipAddress, userAgent, s.authTimeout)
	if err := s.authRequestRepo.Add(ctx, authReq); err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	// Send push notifications to all devices
	if s.fcmService != nil {
		tokens := make([]string, 0, len(devices))
		for _, d := range devices {
			tokens = append(tokens, d.FCMToken)
		}

		notification := AuthRequestNotification{
			RequestID: authReq.ID,
			Email:     user.Email,
			IPAddress: ipAddress,
			UserAgent: userAgent,
		}

		sent, _ := s.fcmService.SendAuthRequestToMultiple(ctx, tokens, notification)
		if sent == 0 {
			return nil, fmt.Errorf("failed to send push notification to any device")
		}
	}

	return &InitiateAuthResult{
		RequestID: authReq.ID,
		ExpiresAt: authReq.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// CheckAuthStatus checks the status of an auth request
func (s *AuthService) CheckAuthStatus(ctx context.Context, requestID string) (*models.AuthStatusResponse, error) {
	authReq, err := s.authRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth request: %w", err)
	}
	if authReq == nil {
		return nil, models.ErrAuthRequestNotFound
	}

	// Check if expired
	if authReq.IsExpired() && authReq.Status == models.AuthStatusPending {
		authReq.Status = models.AuthStatusExpired
		s.authRequestRepo.Update(ctx, authReq)
	}

	response := &models.AuthStatusResponse{
		Status:    authReq.Status,
		ExpiresAt: authReq.ExpiresAt,
	}

	// If approved, create session and include token
	if authReq.Status == models.AuthStatusApproved {
		// Check if session was already created for this auth request
		existingSessions, err := s.sessionRepo.GetActiveForUser(ctx, authReq.UserID)
		if err == nil {
			for _, sess := range existingSessions {
				if sess.AuthRequestID != nil && *sess.AuthRequestID == authReq.ID {
					response.SessionToken = sess.ID
					return response, nil
				}
			}
		}

		// Create new session
		session := models.NewWebSession(authReq.UserID, &authReq.ID, authReq.IPAddress, authReq.UserAgent, s.sessionDuration)
		if err := s.sessionRepo.Add(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		response.SessionToken = session.ID
	}

	return response, nil
}

// RespondToAuth handles approve/deny from mobile app
func (s *AuthService) RespondToAuth(ctx context.Context, requestID string, approved bool, deviceID string) error {
	authReq, err := s.authRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get auth request: %w", err)
	}
	if authReq == nil {
		return models.ErrAuthRequestNotFound
	}

	if authReq.Status != models.AuthStatusPending {
		return models.ErrAuthAlreadyResolved
	}

	if authReq.IsExpired() {
		authReq.Status = models.AuthStatusExpired
		s.authRequestRepo.Update(ctx, authReq)
		return models.ErrAuthRequestExpired
	}

	if approved {
		authReq.Approve(deviceID)
	} else {
		authReq.Deny(deviceID)
	}

	if err := s.authRequestRepo.Update(ctx, authReq); err != nil {
		return fmt.Errorf("failed to update auth request: %w", err)
	}

	return nil
}

// GetSession retrieves a session and validates it
func (s *AuthService) GetSession(ctx context.Context, sessionID string) (*models.WebSession, *models.User, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, models.ErrSessionNotFound
	}
	if !session.IsActive {
		return nil, nil, models.ErrSessionInactive
	}
	if session.IsExpired() {
		return nil, nil, models.ErrSessionExpired
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || !user.IsActive {
		return nil, nil, models.ErrUserNotFound
	}

	return session, user, nil
}

// Logout invalidates a session
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Invalidate(ctx, sessionID)
}

// LogoutAll invalidates all sessions for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return s.sessionRepo.InvalidateAllForUser(ctx, userID)
}

// GetUserByAPIKeyHash looks up a user by their API key hash
func (s *AuthService) GetUserByAPIKeyHash(ctx context.Context, keyHash string) (*models.User, error) {
	return s.userRepo.GetByAPIKeyHash(ctx, keyHash)
}

// CreateSessionForUser creates a web session directly for a user (admin login)
func (s *AuthService) CreateSessionForUser(ctx context.Context, userID, ipAddress, userAgent string) (*models.WebSession, error) {
	session := models.NewWebSession(userID, nil, ipAddress, userAgent, s.sessionDuration)
	if err := s.sessionRepo.Add(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return session, nil
}

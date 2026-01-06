package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2/google"
)

// FCMService handles Firebase Cloud Messaging operations using direct HTTP API
type FCMService struct {
	projectID   string
	credentials []byte
	httpClient  *http.Client
	token       string
	tokenExpiry time.Time
	tokenMu     sync.Mutex
}

// NewFCMService creates a new FCMService with the given credentials file
func NewFCMService(credentialsPath string) (*FCMService, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("firebase credentials path is required")
	}

	log.Printf("Initializing FCM with credentials from: %s", credentialsPath)

	// Check if file exists and is readable
	if _, err := os.Stat(credentialsPath); err != nil {
		return nil, fmt.Errorf("credentials file not accessible: %w", err)
	}

	// Read credentials
	credData, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(credData, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}
	log.Printf("Using Firebase project: %s", creds.ProjectID)

	svc := &FCMService{
		projectID:   creds.ProjectID,
		credentials: credData,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}

	// Test getting a token
	if _, err := svc.getAccessToken(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to get initial access token: %w", err)
	}
	log.Printf("Firebase Cloud Messaging initialized successfully (direct HTTP)")

	return svc, nil
}

// getAccessToken returns a valid OAuth2 access token, refreshing if needed
func (s *FCMService) getAccessToken(ctx context.Context) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	// Return cached token if still valid (with 5 min buffer)
	if s.token != "" && time.Now().Add(5*time.Minute).Before(s.tokenExpiry) {
		log.Printf("FCM: Using cached token (expires %v)", s.tokenExpiry)
		return s.token, nil
	}

	// Get new token using FindDefaultCredentials (uses GOOGLE_APPLICATION_CREDENTIALS)
	scopes := []string{"https://www.googleapis.com/auth/firebase.messaging"}

	// Try default credentials first (uses GOOGLE_APPLICATION_CREDENTIALS env var)
	creds, err := google.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		log.Printf("FCM: FindDefaultCredentials failed, falling back to file: %v", err)
		// Fallback to explicit credentials
		creds, err = google.CredentialsFromJSON(ctx, s.credentials, scopes...)
		if err != nil {
			return "", fmt.Errorf("failed to create credentials: %w", err)
		}
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	s.token = token.AccessToken
	s.tokenExpiry = token.Expiry

	// Log token info for debugging (first/last few chars only for security)
	tokenLen := len(s.token)
	if tokenLen > 20 {
		log.Printf("FCM: Obtained new access token (len=%d, prefix=%s..., suffix=...%s), expires at %v",
			tokenLen, s.token[:10], s.token[tokenLen-10:], s.tokenExpiry)
	} else {
		log.Printf("FCM: Obtained new access token (len=%d), expires at %v", tokenLen, s.tokenExpiry)
	}

	return s.token, nil
}

// AuthRequestNotification represents the data sent in an auth request push
type AuthRequestNotification struct {
	RequestID string `json:"requestId"`
	Email     string `json:"email"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

// FCM API message structures
type fcmMessage struct {
	Message fcmMessageBody `json:"message"`
}

type fcmMessageBody struct {
	Token        string            `json:"token"`
	Data         map[string]string `json:"data,omitempty"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Android      *fcmAndroid       `json:"android,omitempty"`
	APNS         *fcmAPNS          `json:"apns,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmAndroid struct {
	Priority     string                  `json:"priority,omitempty"`
	Notification *fcmAndroidNotification `json:"notification,omitempty"`
}

type fcmAndroidNotification struct {
	ClickAction string `json:"click_action,omitempty"`
	ChannelID   string `json:"channel_id,omitempty"`
}

type fcmAPNS struct {
	Headers map[string]string `json:"headers,omitempty"`
	Payload *fcmAPNSPayload   `json:"payload,omitempty"`
}

type fcmAPNSPayload struct {
	Aps *fcmAps `json:"aps,omitempty"`
}

type fcmAps struct {
	Alert            *fcmApsAlert `json:"alert,omitempty"`
	Sound            string       `json:"sound,omitempty"`
	ContentAvailable int          `json:"content-available,omitempty"`
	Category         string       `json:"category,omitempty"`
}

type fcmApsAlert struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SendAuthRequest sends a push notification for authentication approval
func (s *FCMService) SendAuthRequest(ctx context.Context, fcmToken string, notification AuthRequestNotification) error {
	token, err := s.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	body := fmt.Sprintf("Someone is trying to log in to your PhotoSync account from %s", notification.IPAddress)

	message := fcmMessage{
		Message: fcmMessageBody{
			Token: fcmToken,
			Data: map[string]string{
				"type":          "auth_request",
				"authRequestId": notification.RequestID,
				"email":         notification.Email,
				"ipAddress":     notification.IPAddress,
				"userAgent":     notification.UserAgent,
			},
			Notification: &fcmNotification{
				Title: "Login Request",
				Body:  body,
			},
			Android: &fcmAndroid{
				Priority: "high",
				Notification: &fcmAndroidNotification{
					ClickAction: "OPEN_AUTH_APPROVAL",
					ChannelID:   "auth_requests",
				},
			},
			APNS: &fcmAPNS{
				Headers: map[string]string{
					"apns-priority":  "10",
					"apns-push-type": "alert",
				},
				Payload: &fcmAPNSPayload{
					Aps: &fcmAps{
						Alert: &fcmApsAlert{
							Title: "Login Request",
							Body:  body,
						},
						Sound:            "default",
						ContentAvailable: 1,
						Category:         "AUTH_REQUEST",
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", s.projectID)
	log.Printf("FCM: Sending to URL: %s", url)
	log.Printf("FCM: Request body: %s", string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("FCM: Authorization header set (Bearer token len=%d)", len(token))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("FCM API error: status=%d, body=%s", resp.StatusCode, string(respBody))
		// Log additional debug info
		log.Printf("FCM: Debug - project_id=%s, fcm_token_prefix=%s...", s.projectID, fcmToken[:min(20, len(fcmToken))])
		return fmt.Errorf("FCM API error: %s", string(respBody))
	}

	log.Printf("FCM notification sent successfully: %s", string(respBody))
	return nil
}

// SendAuthRequestToMultiple sends the same notification to multiple tokens
func (s *FCMService) SendAuthRequestToMultiple(ctx context.Context, fcmTokens []string, notification AuthRequestNotification) (int, error) {
	if len(fcmTokens) == 0 {
		return 0, nil
	}

	successCount := 0
	for _, token := range fcmTokens {
		if err := s.SendAuthRequest(ctx, token, notification); err != nil {
			log.Printf("FCM send failed for token %s...: %v", token[:min(20, len(token))], err)
			continue
		}
		successCount++
	}

	return successCount, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

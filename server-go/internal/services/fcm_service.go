package services

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMService handles Firebase Cloud Messaging operations
type FCMService struct {
	client *messaging.Client
}

// NewFCMService creates a new FCMService with the given credentials file
func NewFCMService(credentialsPath string) (*FCMService, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("firebase credentials path is required")
	}

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Firebase messaging client: %w", err)
	}

	return &FCMService{client: client}, nil
}

// AuthRequestNotification represents the data sent in an auth request push
type AuthRequestNotification struct {
	RequestID string `json:"requestId"`
	Email     string `json:"email"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

// SendAuthRequest sends a push notification for authentication approval
func (s *FCMService) SendAuthRequest(ctx context.Context, fcmToken string, notification AuthRequestNotification) error {
	message := &messaging.Message{
		Token: fcmToken,
		Data: map[string]string{
			"type":      "auth_request",
			"requestId": notification.RequestID,
			"email":     notification.Email,
			"ipAddress": notification.IPAddress,
			"userAgent": notification.UserAgent,
		},
		Notification: &messaging.Notification{
			Title: "Login Request",
			Body:  fmt.Sprintf("Someone is trying to log in to your PhotoSync account from %s", notification.IPAddress),
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ClickAction: "OPEN_AUTH_APPROVAL",
				ChannelID:   "auth_requests",
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: "Login Request",
						Body:  fmt.Sprintf("Someone is trying to log in to your PhotoSync account from %s", notification.IPAddress),
					},
					Sound:            "default",
					ContentAvailable: true,
					Category:         "AUTH_REQUEST",
				},
			},
		},
	}

	_, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	return nil
}

// SendToMultiple sends the same notification to multiple tokens
func (s *FCMService) SendAuthRequestToMultiple(ctx context.Context, fcmTokens []string, notification AuthRequestNotification) (int, error) {
	if len(fcmTokens) == 0 {
		return 0, nil
	}

	successCount := 0
	for _, token := range fcmTokens {
		if err := s.SendAuthRequest(ctx, token, notification); err != nil {
			// Log error but continue with other tokens
			continue
		}
		successCount++
	}

	return successCount, nil
}

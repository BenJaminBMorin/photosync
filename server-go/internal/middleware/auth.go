package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

type contextKey string

const (
	UserContextKey    contextKey = "user"
	SessionContextKey contextKey = "session"
)

// GetUserFromContext retrieves the authenticated user from request context
func GetUserFromContext(ctx context.Context) *models.User {
	if user, ok := ctx.Value(UserContextKey).(*models.User); ok {
		return user
	}
	return nil
}

// GetSessionFromContext retrieves the web session from request context
func GetSessionFromContext(ctx context.Context) *models.WebSession {
	if session, ok := ctx.Value(SessionContextKey).(*models.WebSession); ok {
		return session
	}
	return nil
}

// APIKeyAuth creates middleware for API key authentication (legacy single-key mode)
func APIKeyAuth(apiKey, headerName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health endpoints
			path := r.URL.Path
			if path == "/health" || path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Only authenticate API routes
			if !strings.HasPrefix(path, "/api") {
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from header
			providedKey := r.Header.Get(headerName)
			if providedKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "API key is required."})
				return
			}

			// Constant-time comparison to prevent timing attacks
			if !constantTimeEquals(apiKey, providedKey) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key."})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserAPIKeyAuth creates middleware that looks up users by API key hash
func UserAPIKeyAuth(userRepo repository.UserRepo, headerName string, skipPaths []string) func(http.Handler) http.Handler {
	skipSet := make(map[string]bool)
	for _, p := range skipPaths {
		skipSet[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip auth for explicit paths
			if skipSet[path] {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for paths starting with skip prefixes
			for p := range skipSet {
				if strings.HasSuffix(p, "*") && strings.HasPrefix(path, strings.TrimSuffix(p, "*")) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Only authenticate API routes
			if !strings.HasPrefix(path, "/api") {
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from header
			providedKey := r.Header.Get(headerName)
			if providedKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "API key is required."})
				return
			}

			// Hash the provided key and look up user
			keyHash := models.HashAPIKey(providedKey)
			user, err := userRepo.GetByAPIKeyHash(r.Context(), keyHash)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error."})
				return
			}

			if user == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key."})
				return
			}

			if !user.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "User account is disabled."})
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SessionAuth creates middleware for web session authentication
func SessionAuth(sessionRepo repository.WebSessionRepo, userRepo repository.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session token from cookie
			cookie, err := r.Cookie("session_token")
			if err != nil || cookie.Value == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Session required."})
				return
			}

			// Look up session
			session, err := sessionRepo.GetByID(r.Context(), cookie.Value)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error."})
				return
			}

			if session == nil || !session.IsActive || session.IsExpired() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Session expired or invalid."})
				return
			}

			// Look up user
			user, err := userRepo.GetByID(r.Context(), session.UserID)
			if err != nil || user == nil || !user.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "User not found or disabled."})
				return
			}

			// Update last activity (async, don't wait)
			go sessionRepo.Touch(context.Background(), session.ID)

			// Add session and user to context
			ctx := context.WithValue(r.Context(), SessionContextKey, session)
			ctx = context.WithValue(ctx, UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// constantTimeEquals performs a constant-time string comparison
func constantTimeEquals(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

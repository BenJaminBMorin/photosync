package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/photosync/server/internal/repository"
)

// AdminAuth creates middleware requiring session auth + admin status
func AdminAuth(sessionRepo repository.WebSessionRepo, userRepo repository.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session token from cookie
			cookie, err := r.Cookie("session_token")
			if err != nil || cookie.Value == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required."})
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

			// CRITICAL: Verify admin status
			if !user.IsAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "Admin access required."})
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

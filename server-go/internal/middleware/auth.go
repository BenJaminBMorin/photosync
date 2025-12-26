package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// APIKeyAuth creates middleware for API key authentication
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

// constantTimeEquals performs a constant-time string comparison
func constantTimeEquals(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

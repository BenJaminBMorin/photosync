package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/photosync/server/internal/services"
)

// SetupRequired creates middleware that redirects to setup if not configured
func SetupRequired(setupService *services.SetupService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Always allow setup routes
			if strings.HasPrefix(path, "/setup") || strings.HasPrefix(path, "/api/setup") {
				next.ServeHTTP(w, r)
				return
			}

			// Always allow bootstrap and recovery endpoints (emergency access)
			if path == "/api/web/auth/bootstrap" ||
				path == "/api/web/auth/request-recovery" ||
				path == "/api/web/auth/recover" {
				next.ServeHTTP(w, r)
				return
			}

			// Always allow health checks
			if path == "/health" || path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Always allow static assets
			if strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/css/") ||
				strings.HasPrefix(path, "/js/") || strings.HasPrefix(path, "/images/") {
				next.ServeHTTP(w, r)
				return
			}

			// Always allow swagger
			if strings.HasPrefix(path, "/swagger") {
				next.ServeHTTP(w, r)
				return
			}

			// Check if setup is required
			required, err := setupService.IsSetupRequired(r.Context())
			if err != nil {
				// If we can't determine setup status, allow request but log error
				next.ServeHTTP(w, r)
				return
			}

			if required {
				// For API requests, return 503 Service Unavailable
				if strings.HasPrefix(path, "/api") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":    "Setup required",
						"message":  "Please complete the setup wizard at /setup",
						"setupUrl": "/setup",
					})
					return
				}

				// For web requests, redirect to setup
				http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

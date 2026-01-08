package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/photosync/server/internal/config"
	"github.com/photosync/server/internal/handlers"
	custommw "github.com/photosync/server/internal/middleware"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title PhotoSync API
// @version 2.0
// @description API for syncing photos from mobile devices to a NAS server with web gallery and push authentication
// @description
// @description ## Authentication
// @description - Mobile API: Use X-API-Key header with your personal API key
// @description - Web Gallery: Session-based authentication via push notifications
// @description
// @description ## Features
// @description - Upload photos with automatic duplicate detection via SHA256 hash
// @description - Photos organized by year/month folders
// @description - Multi-user support with per-user API keys
// @description - Web gallery with push notification login
// @description - Setup wizard for first-time configuration

// @contact.name PhotoSync Support
// @license.name MIT

// @host localhost:5050
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// Build-time variables set via ldflags
var (
	Version            = "dev"
	BuildDate          = "unknown"
	ContainerBuildDate = ""
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	var db *sql.DB
	var photoRepo repository.PhotoRepo
	var photoRepoPostgres *repository.PhotoRepositoryPostgres

	if cfg.UsePostgres() {
		log.Println("Using PostgreSQL database")
		db, err = repository.NewPostgresDB(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize PostgreSQL database: %v", err)
		}
		photoRepoPostgres = repository.NewPhotoRepositoryPostgres(db)
		photoRepo = photoRepoPostgres
	} else {
		log.Println("Using SQLite database")
		db, err = repository.NewSQLiteDB(cfg.DatabasePath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite database: %v", err)
		}
		photoRepo = repository.NewPhotoRepository(db)
	}
	defer db.Close()

	// Initialize all repositories
	userRepo := repository.NewUserRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	authRequestRepo := repository.NewAuthRequestRepository(db)
	deleteRequestRepo := repository.NewDeleteRequestRepository(db)
	sessionRepo := repository.NewWebSessionRepository(db)
	setupConfigRepo := repository.NewSetupConfigRepository(db)
	bootstrapKeyRepo := repository.NewBootstrapKeyRepository(db)
	recoveryTokenRepo := repository.NewRecoveryTokenRepository(db)
	inviteTokenRepo := repository.NewInviteTokenRepository(db)
	configOverrideRepo := repository.NewConfigOverrideRepository(db)
	smtpConfigRepo := repository.NewSMTPConfigRepository(db)

	// Collection repositories
	collectionRepo := repository.NewCollectionRepository(db)
	collectionPhotoRepo := repository.NewCollectionPhotoRepository(db)
	collectionShareRepo := repository.NewCollectionShareRepository(db)

	// Theme and user preferences repositories
	themeRepo := repository.NewThemeRepository(db)
	userPrefsRepo := repository.NewUserPreferencesRepository(db)

	// Sync state repository
	deviceSyncStateRepo := repository.NewDeviceSyncStateRepository(db)

	// Initialize services
	hashService := services.NewHashService()
	storageService, err := services.NewPhotoStorageService(
		cfg.PhotoStorage.BasePath,
		cfg.PhotoStorage.AllowedExtensions,
		cfg.PhotoStorage.MaxFileSizeMB,
	)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}

	// EXIF, thumbnail, and metadata services
	exifService := services.NewEXIFService()
	thumbnailService := services.NewThumbnailService(cfg.PhotoStorage.BasePath)
	metadataService := services.NewMetadataService(cfg.PhotoStorage.BasePath)

	// Maintenance service for background tasks
	maintenanceService := services.NewMaintenanceService(photoRepo, thumbnailService, cfg.PhotoStorage.BasePath)
	maintenanceService.Start()

	// Config directory for Firebase credentials etc
	configDir := filepath.Join(cfg.PhotoStorage.BasePath, ".config")

	// Setup service
	setupService := services.NewSetupService(setupConfigRepo, userRepo, configDir)

	// Encryption service for sensitive config (SMTP passwords etc)
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Println("WARNING: ENCRYPTION_KEY not set, using default key (not recommended for production)")
		encryptionKey = "photosync-default-encryption-key-change-me"
	}
	encryptionService, err := services.NewEncryptionService(encryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize encryption service: %v", err)
	}

	// SMTP service for sending emails
	smtpService := services.NewSMTPService(smtpConfigRepo, encryptionService)

	// Config service for runtime configuration management
	configService := services.NewConfigService(
		configOverrideRepo, smtpConfigRepo, setupConfigRepo,
		encryptionService, cfg,
	)

	// Bootstrap service for emergency admin access
	bootstrapService := services.NewBootstrapService(
		bootstrapKeyRepo, userRepo, setupConfigRepo, configDir,
	)

	// Generate bootstrap key if needed (first startup, no admin exists)
	if err := bootstrapService.GenerateBootstrapKeyIfNeeded(context.Background()); err != nil {
		log.Printf("Warning: Failed to generate bootstrap key: %v", err)
	}

	// Recovery service for email-based account recovery
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		// Default to localhost for development
		serverURL = "http://localhost" + cfg.ServerAddress
		log.Printf("WARNING: SERVER_URL not set, using default: %s", serverURL)
	}
	recoveryService := services.NewRecoveryService(
		recoveryTokenRepo, userRepo, smtpService, serverURL,
	)

	// FCM service (optional - only if Firebase is configured)
	var fcmService *services.FCMService
	firebaseCredPath := setupService.GetFirebaseCredentialsPath()
	if firebaseCredPath != "" {
		fcmService, err = services.NewFCMService(firebaseCredPath)
		if err != nil {
			log.Printf("Warning: Failed to initialize FCM service: %v", err)
		} else {
			log.Println("Firebase Cloud Messaging initialized")
		}
	}

	// Auth service
	authTimeout := 60     // 60 seconds for auth approval
	sessionDuration := 24 // 24 hours session
	authService := services.NewAuthService(
		userRepo, deviceRepo, authRequestRepo, sessionRepo,
		fcmService, authTimeout, sessionDuration,
	)

	// Delete service
	deleteTimeout := 60 // 60 seconds for delete approval
	deleteService := services.NewDeleteService(
		userRepo, deviceRepo, deleteRequestRepo, photoRepo,
		fcmService, deleteTimeout,
	)

	// Admin service
	adminService := services.NewAdminService(
		userRepo, deviceRepo, sessionRepo, photoRepo, setupConfigRepo,
		cfg.PhotoStorage.BasePath,
		Version, BuildDate, ContainerBuildDate,
	)

	// Theme service
	themeService := services.NewThemeService(themeRepo)

	// Seed system themes on startup
	if err := repository.SeedSystemThemes(context.Background(), themeRepo); err != nil {
		log.Printf("Warning: Failed to seed system themes: %v", err)
	} else {
		log.Println("System themes initialized")
	}

	// Collection service
	collectionService := services.NewCollectionService(
		collectionRepo, collectionPhotoRepo, collectionShareRepo,
		photoRepo, userRepo, themeService, userPrefsRepo,
	)

	// Determine web directory for static files and templates
	webDir := filepath.Join(getExecutableDir(), "web")
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		// Try current directory
		webDir = "web"
	}

	// Initialize handlers
	photoHandler := handlers.NewPhotoHandler(photoRepo, storageService, hashService, exifService, thumbnailService, metadataService)
	healthHandler := handlers.NewHealthHandler(setupConfigRepo)
	setupHandler := handlers.NewSetupHandler(setupService, configService, smtpService)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo)
	webAuthHandler := handlers.NewWebAuthHandler(authService, bootstrapService, recoveryService)
	webDeleteHandler := handlers.NewWebDeleteHandler(deleteService)
	adminHandler := handlers.NewAdminHandler(adminService)
	configHandler := handlers.NewConfigHandler(configService, smtpService)
	// Web gallery handler requires PostgreSQL for location features
	var webGalleryHandler *handlers.WebGalleryHandler
	if photoRepoPostgres != nil {
		webGalleryHandler = handlers.NewWebGalleryHandler(photoRepoPostgres, cfg.PhotoStorage.BasePath)
	}

	// Collection handler
	collectionHandler := handlers.NewCollectionHandler(collectionService)

	// Theme handler
	themeHandler := handlers.NewThemeHandler(themeService)

	// User handler
	userHandler := handlers.NewUserHandler(userPrefsRepo)

	// Invite handler
	inviteHandler := handlers.NewInviteHandler(inviteTokenRepo, userRepo, smtpService, serverURL)

	// Sync handler
	syncHandler := handlers.NewSyncHandler(photoRepo, deviceRepo, deviceSyncStateRepo, storageService)

	// Public gallery handler
	publicGalleryHandler := handlers.NewPublicGalleryHandler(
		collectionService, collectionRepo, collectionPhotoRepo,
		photoRepo, cfg.PhotoStorage.BasePath, webDir,
	)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Setup required middleware (redirects to /setup if not configured)
	r.Use(custommw.SetupRequired(setupService))

	// Static file server for web UI
	fileServer := http.FileServer(http.Dir(webDir))
	r.Handle("/css/*", fileServer)
	r.Handle("/js/*", fileServer)
	r.Handle("/images/*", fileServer)

	// Swagger UI (always accessible)
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Health check and version (no auth)
	r.Get("/health", healthHandler.HealthCheck)
	r.Get("/api/health", healthHandler.HealthCheck)
	r.Get("/api/info", healthHandler.GetAppInfo)
	r.Get("/api/version", handlers.VersionHandler)

	// Public theme routes (no auth required)
	r.Get("/api/themes", themeHandler.ListThemes)
	r.Get("/api/themes/{id}", themeHandler.GetTheme)
	r.Get("/api/themes/{id}/css", themeHandler.GetThemeCSS)

	// Public invite redemption (no auth required)
	r.Post("/api/invite/redeem", inviteHandler.HandleRedeemInvite)

	// Setup routes (no auth during setup)
	r.Get("/setup", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "setup", "index.html"))
	})
	r.Get("/setup/*", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "setup", "index.html"))
	})
	r.Route("/api/setup", func(r chi.Router) {
		r.Get("/status", setupHandler.GetStatus)
		r.Post("/firebase", setupHandler.UploadFirebaseCredentials)
		r.Post("/email", setupHandler.ConfigureEmail)
		r.Post("/email/test", setupHandler.TestEmail)
		r.Get("/validation", setupHandler.GetValidationStatus)
		r.Post("/admin", setupHandler.CreateAdmin)
		r.Post("/complete", setupHandler.CompleteSetup)
	})

	// Web authentication routes (no auth required for initiate/status/admin-login/bootstrap/recovery)
	r.Post("/api/web/auth/initiate", webAuthHandler.InitiateAuth)
	r.Get("/api/web/auth/status/{id}", webAuthHandler.CheckStatus)
	r.Post("/api/web/auth/admin-login", webAuthHandler.AdminLogin)
	r.Post("/api/web/auth/bootstrap", webAuthHandler.BootstrapLogin)
	r.Post("/api/web/auth/request-recovery", webAuthHandler.RequestRecovery)
	r.Post("/api/web/auth/recover", webAuthHandler.RecoverAccount)

	// API routes requiring API key authentication
	skipPaths := []string{
		"/health",
		"/api/health",
		"/api/info",
		"/api/setup/*",
		"/api/web/auth/initiate",
		"/api/web/auth/status/*",
	}

	r.Group(func(r chi.Router) {
		r.Use(custommw.UserAPIKeyAuth(userRepo, cfg.Security.APIKeyHeader, skipPaths))

		// Photo upload API (mobile)
		r.Route("/api/photos", func(r chi.Router) {
			r.Post("/upload", photoHandler.Upload)
			r.Post("/check", photoHandler.CheckHashes)
			r.Get("/", photoHandler.List)
			r.Get("/{id}", photoHandler.GetByID)
			r.Delete("/{id}", photoHandler.Delete)
		})

		// Device registration (mobile)
		r.Route("/api/devices", func(r chi.Router) {
			r.Post("/register", deviceHandler.RegisterDevice)
			r.Get("/", deviceHandler.ListDevices)
			r.Delete("/{id}", deviceHandler.DeleteDevice)
		})

		// Sync routes (mobile)
		r.Route("/api/sync", func(r chi.Router) {
			r.Get("/status", syncHandler.GetSyncStatus)
			r.Post("/photos", syncHandler.SyncPhotos)
			r.Get("/legacy-photos", syncHandler.GetLegacyPhotos)
			r.Post("/claim-legacy", syncHandler.ClaimLegacy)
			r.Get("/thumbnail/{id}", syncHandler.GetThumbnail)
			r.Get("/download/{hash}", syncHandler.DownloadPhotoByHash)
		})

		// Photo download (mobile)
		r.Get("/api/photos/{id}/download", syncHandler.DownloadPhoto)

		// Auth response from mobile
		r.Post("/api/web/auth/respond", webAuthHandler.RespondAuth)

		// Delete response from mobile
		r.Post("/api/web/delete/respond", webDeleteHandler.RespondDelete)
	})

	// Web routes requiring session authentication
	r.Group(func(r chi.Router) {
		r.Use(custommw.SessionAuth(sessionRepo, userRepo))

		r.Get("/api/web/session", webAuthHandler.GetSession)
		r.Post("/api/web/auth/logout", webAuthHandler.Logout)

		// User preferences routes
		r.Get("/api/users/me/preferences", userHandler.GetPreferences)
		r.Put("/api/users/me/preferences", userHandler.UpdatePreferences)

		// Delete request routes
		r.Post("/api/web/delete/initiate", webDeleteHandler.InitiateDelete)
		r.Get("/api/web/delete/status/{id}", webDeleteHandler.CheckStatus)

		if webGalleryHandler != nil {
			r.Route("/api/web/photos", func(r chi.Router) {
				r.Get("/", webGalleryHandler.ListPhotos)
				r.Get("/locations", webGalleryHandler.ListPhotosWithLocation)
				r.Get("/{id}/image", webGalleryHandler.ServeImage)
				r.Get("/{id}/thumbnail", webGalleryHandler.ServeThumbnail)
				r.Delete("/{id}", webGalleryHandler.DeletePhoto)
			})
		}

		// Collection management routes
		r.Route("/api/web/collections", func(r chi.Router) {
			r.Get("/", collectionHandler.ListCollections)
			r.Post("/", collectionHandler.CreateCollection)
			r.Get("/themes", collectionHandler.GetThemes)
			r.Get("/{id}", collectionHandler.GetCollection)
			r.Put("/{id}", collectionHandler.UpdateCollection)
			r.Delete("/{id}", collectionHandler.DeleteCollection)
			r.Put("/{id}/visibility", collectionHandler.UpdateVisibility)
			r.Post("/{id}/photos", collectionHandler.AddPhotos)
			r.Delete("/{id}/photos", collectionHandler.RemovePhotos)
			r.Put("/{id}/photos/reorder", collectionHandler.ReorderPhotos)
			r.Post("/{id}/shares", collectionHandler.ShareWithUsers)
			r.Delete("/{id}/shares/{userId}", collectionHandler.RemoveShare)
		})
	})

	// Admin routes requiring session auth + admin status
	r.Group(func(r chi.Router) {
		r.Use(custommw.AdminAuth(sessionRepo, userRepo))

		r.Route("/api/admin", func(r chi.Router) {
			// User management
			r.Get("/users", adminHandler.ListUsers)
			r.Post("/users", adminHandler.CreateUser)
			r.Get("/users/{id}", adminHandler.GetUser)
			r.Put("/users/{id}", adminHandler.UpdateUser)
			r.Delete("/users/{id}", adminHandler.DeleteUser)
			r.Post("/users/{id}/reset-api-key", adminHandler.ResetAPIKey)
			r.Post("/users/{id}/invite", inviteHandler.HandleGenerateInvite)

			// User's devices
			r.Get("/users/{id}/devices", adminHandler.GetUserDevices)
			r.Delete("/users/{id}/devices/{deviceId}", adminHandler.DeleteUserDevice)

			// User's sessions
			r.Get("/users/{id}/sessions", adminHandler.GetUserSessions)
			r.Delete("/users/{id}/sessions/{sessionId}", adminHandler.InvalidateUserSession)

			// System
			r.Get("/system/status", adminHandler.GetSystemStatus)
			r.Get("/system/config", adminHandler.GetSystemConfig)

			// App settings
			r.Get("/settings/app", adminHandler.GetAppSettings)
			r.Put("/settings/app", adminHandler.UpdateAppSettings)

			// Theme management
			r.Get("/themes", themeHandler.ListAllThemes)
			r.Get("/themes/{id}", themeHandler.GetThemeAdmin)
			r.Post("/themes", themeHandler.CreateTheme)
			r.Put("/themes/{id}", themeHandler.UpdateTheme)
			r.Delete("/themes/{id}", themeHandler.DeleteTheme)
			r.Get("/themes/{id}/preview", themeHandler.GetThemePreview)

			// Configuration management
			r.Get("/config", configHandler.GetConfig)
			r.Put("/config", configHandler.UpdateConfig)
			r.Get("/config/smtp", configHandler.GetSMTPConfig)
			r.Put("/config/smtp", configHandler.UpdateSMTPConfig)
			r.Post("/config/smtp/test", configHandler.TestSMTP)
			r.Get("/config/restart-status", configHandler.GetRestartStatus)

			// Maintenance service control
			r.Get("/maintenance/status", func(w http.ResponseWriter, r *http.Request) {
				status := maintenanceService.GetStatus()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(status)
			})

			r.Post("/maintenance/start", func(w http.ResponseWriter, r *http.Request) {
				maintenanceService.Start()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "started"})
			})

			r.Post("/maintenance/stop", func(w http.ResponseWriter, r *http.Request) {
				maintenanceService.Stop()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
			})

			r.Post("/maintenance/run", func(w http.ResponseWriter, r *http.Request) {
				maintenanceService.RunNow()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "triggered"})
			})

			// Thumbnail stats
			r.Get("/thumbnail-stats", func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()

				// Get total photo count
				totalCount, err := photoRepo.GetCount(ctx)
				if err != nil {
					http.Error(w, "Failed to get photo count: "+err.Error(), http.StatusInternalServerError)
					return
				}

				// Get photos without thumbnails (just count by getting a large batch)
				photosWithout, err := photoRepo.GetPhotosWithoutThumbnails(ctx, 10000)
				if err != nil {
					http.Error(w, "Failed to get photos without thumbnails: "+err.Error(), http.StatusInternalServerError)
					return
				}

				missingCount := len(photosWithout)
				withThumbs := totalCount - missingCount
				percentage := 0.0
				if totalCount > 0 {
					percentage = float64(withThumbs) / float64(totalCount) * 100
				}

				response := map[string]interface{}{
					"total":      totalCount,
					"withThumbs": withThumbs,
					"missing":    missingCount,
					"percentage": percentage,
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			})

			// Thumbnail regeneration
			r.Post("/regenerate-thumbnails", func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				batchSize := 50 // Process 50 at a time

				photos, err := photoRepo.GetPhotosWithoutThumbnails(ctx, batchSize)
				if err != nil {
					http.Error(w, "Failed to get photos: "+err.Error(), http.StatusInternalServerError)
					return
				}

				processed := 0
				failed := 0
				skipped := 0

				for _, photo := range photos {
					// Skip unsupported formats
					if !services.IsSupportedFormat(photo.StoredPath) || services.IsHEIC(photo.StoredPath) {
						skipped++
						continue
					}

					result, err := thumbnailService.RegenerateThumbnailsFromFile(photo.ID, photo.StoredPath)
					if err != nil {
						log.Printf("Failed to regenerate thumbnails for %s: %v", photo.ID, err)
						failed++
						continue
					}

					// Update database
					if err := photoRepo.UpdateThumbnails(ctx, photo.ID, result.SmallPath, result.MediumPath, result.LargePath); err != nil {
						log.Printf("Failed to update thumbnails in DB for %s: %v", photo.ID, err)
						failed++
						continue
					}

					processed++
				}

				// Get remaining count
				remaining, _ := photoRepo.GetPhotosWithoutThumbnails(ctx, 1)
				hasMore := len(remaining) > 0

				response := map[string]interface{}{
					"processed": processed,
					"failed":    failed,
					"skipped":   skipped,
					"hasMore":   hasMore,
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			})
		})
	})

	// Admin UI pages
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "admin", "index.html"))
	})
	r.Get("/admin/*", func(w http.ResponseWriter, r *http.Request) {
		path := chi.URLParam(r, "*")
		filePath := filepath.Join(webDir, "admin", path+".html")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Fall back to index.html for SPA routing
			http.ServeFile(w, r, filepath.Join(webDir, "admin", "index.html"))
			return
		}
		http.ServeFile(w, r, filePath)
	})

	// Public gallery routes (no auth required)
	r.Get("/gallery/{slug}", publicGalleryHandler.ViewGalleryBySlug)
	r.Get("/gallery/s/{token}", publicGalleryHandler.ViewGalleryByToken)
	r.Get("/gallery/photos/{photoId}/image", publicGalleryHandler.ServeGalleryImage)
	r.Get("/gallery/photos/{photoId}/thumbnail", publicGalleryHandler.ServeGalleryThumbnail)

	// Collections management page (requires session auth handled by JS)
	r.Get("/collections", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "collections.html"))
	})

	// Web UI pages
	r.Get("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "login.html"))
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
	})

	// Create server
	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Longer for uploads
		IdleTimeout:  60 * time.Second,
	}

	// Start background cleanup goroutine for expired tokens
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()

			// Expire old bootstrap keys
			if expired, err := bootstrapKeyRepo.ExpireOld(ctx); err != nil {
				log.Printf("ERROR: Failed to expire old bootstrap keys: %v", err)
			} else if expired > 0 {
				log.Printf("Expired %d old bootstrap keys", expired)
			}

			// Expire old recovery tokens
			if expired, err := recoveryTokenRepo.ExpireOld(ctx); err != nil {
				log.Printf("ERROR: Failed to expire old recovery tokens: %v", err)
			} else if expired > 0 {
				log.Printf("Expired %d old recovery tokens", expired)
			}
		}
	}()

	// Start server in goroutine
	go func() {
		log.Printf("PhotoSync Server starting on %s", cfg.ServerAddress)
		log.Printf("Photo storage path: %s", cfg.PhotoStorage.BasePath)
		log.Printf("Max file size: %dMB", cfg.PhotoStorage.MaxFileSizeMB)
		log.Printf("Web UI available at http://localhost%s", cfg.ServerAddress)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

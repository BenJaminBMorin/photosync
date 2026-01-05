package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	_ "github.com/photosync/server/docs"
)

// @title PhotoSync API
// @version 1.0
// @description API for syncing photos from mobile devices to a NAS server
// @description
// @description ## Authentication
// @description All endpoints (except /api/health) require an API key passed via the X-API-Key header.
// @description
// @description ## Features
// @description - Upload photos with automatic duplicate detection via SHA256 hash
// @description - Photos organized by year/month folders
// @description - Batch hash checking to avoid uploading duplicates
// @description - Paginated photo listing

// @contact.name PhotoSync Support
// @license.name MIT

// @host localhost:5050
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database and repository
	var photoRepo repository.PhotoRepo
	if cfg.UsePostgres() {
		log.Println("Using PostgreSQL database")
		db, err := repository.NewPostgresDB(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize PostgreSQL database: %v", err)
		}
		defer db.Close()
		photoRepo = repository.NewPhotoRepositoryPostgres(db)
	} else {
		log.Println("Using SQLite database")
		db, err := repository.NewSQLiteDB(cfg.DatabasePath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite database: %v", err)
		}
		defer db.Close()
		photoRepo = repository.NewPhotoRepository(db)
	}

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

	// Initialize handlers
	photoHandler := handlers.NewPhotoHandler(photoRepo, storageService, hashService)
	healthHandler := handlers.NewHealthHandler()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommw.APIKeyAuth(cfg.Security.APIKey, cfg.Security.APIKeyHeader))

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Routes
	r.Get("/health", healthHandler.HealthCheck)
	r.Get("/api/health", healthHandler.HealthCheck)

	r.Route("/api/photos", func(r chi.Router) {
		r.Post("/upload", photoHandler.Upload)
		r.Post("/check", photoHandler.CheckHashes)
		r.Get("/", photoHandler.List)
		r.Get("/{id}", photoHandler.GetByID)
		r.Delete("/{id}", photoHandler.Delete)
	})

	// Create server
	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Longer for uploads
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("PhotoSync Server (Go) starting on %s", cfg.ServerAddress)
		log.Printf("Photo storage path: %s", cfg.PhotoStorage.BasePath)
		log.Printf("Max file size: %dMB", cfg.PhotoStorage.MaxFileSizeMB)

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

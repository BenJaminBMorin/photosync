package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// CollectionService handles collection business logic
type CollectionService struct {
	collectionRepo      repository.CollectionRepo
	collectionPhotoRepo repository.CollectionPhotoRepo
	collectionShareRepo repository.CollectionShareRepo
	photoRepo           repository.PhotoRepo
	userRepo            repository.UserRepo
}

// NewCollectionService creates a new CollectionService
func NewCollectionService(
	collectionRepo repository.CollectionRepo,
	collectionPhotoRepo repository.CollectionPhotoRepo,
	collectionShareRepo repository.CollectionShareRepo,
	photoRepo repository.PhotoRepo,
	userRepo repository.UserRepo,
) *CollectionService {
	return &CollectionService{
		collectionRepo:      collectionRepo,
		collectionPhotoRepo: collectionPhotoRepo,
		collectionShareRepo: collectionShareRepo,
		photoRepo:           photoRepo,
		userRepo:            userRepo,
	}
}

// CreateCollection creates a new collection
func (s *CollectionService) CreateCollection(ctx context.Context, userID string, req *models.CreateCollectionRequest) (*models.Collection, error) {
	collection, err := models.NewCollection(userID, req.Name)
	if err != nil {
		return nil, err
	}

	// Set description if provided
	if req.Description != nil {
		collection.Description = req.Description
	}

	// Use custom slug if provided, otherwise keep generated one
	if req.Slug != nil && *req.Slug != "" {
		slug := s.sanitizeSlug(*req.Slug)
		exists, err := s.collectionRepo.SlugExists(ctx, slug, "")
		if err != nil {
			return nil, fmt.Errorf("failed to check slug: %w", err)
		}
		if exists {
			return nil, models.ErrCollectionSlugExists
		}
		collection.Slug = slug
	} else {
		// Ensure generated slug is unique
		for {
			exists, err := s.collectionRepo.SlugExists(ctx, collection.Slug, "")
			if err != nil {
				return nil, fmt.Errorf("failed to check slug: %w", err)
			}
			if !exists {
				break
			}
			// Regenerate slug
			collection.Slug = models.GenerateSlug(req.Name)
		}
	}

	// Set theme if provided
	if req.Theme != nil {
		if !models.IsValidTheme(*req.Theme) {
			return nil, models.ErrCollectionInvalidTheme
		}
		collection.Theme = models.CollectionTheme(*req.Theme)
	}

	// Set custom CSS if provided
	if req.CustomCSS != nil {
		collection.CustomCSS = req.CustomCSS
	}

	if err := s.collectionRepo.Add(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return collection, nil
}

// GetCollection retrieves a collection by ID with access control
func (s *CollectionService) GetCollection(ctx context.Context, collectionID, userID string) (*models.Collection, error) {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Check access
	if !s.canViewCollection(ctx, collection, userID) {
		return nil, models.ErrCollectionAccessDenied
	}

	collection.IsOwner = collection.UserID == userID
	return collection, nil
}

// GetCollectionBySlug retrieves a public collection by slug
func (s *CollectionService) GetCollectionBySlug(ctx context.Context, slug string) (*models.Collection, error) {
	collection, err := s.collectionRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Only public collections accessible by slug without auth
	if collection.Visibility != models.VisibilityPublic {
		return nil, models.ErrCollectionAccessDenied
	}

	return collection, nil
}

// GetCollectionBySecretToken retrieves a collection by secret token
func (s *CollectionService) GetCollectionBySecretToken(ctx context.Context, token string) (*models.Collection, error) {
	collection, err := s.collectionRepo.GetBySecretToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	return collection, nil
}

// ListCollections returns collections owned by and shared with the user
func (s *CollectionService) ListCollections(ctx context.Context, userID string) (*models.CollectionListResponse, error) {
	owned, err := s.collectionRepo.GetAllForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get owned collections: %w", err)
	}

	shared, err := s.collectionRepo.GetSharedWithUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared collections: %w", err)
	}

	// Convert to summaries
	ownedSummaries := make([]*models.CollectionSummary, 0, len(owned))
	for _, c := range owned {
		ownedSummaries = append(ownedSummaries, s.toSummary(c))
	}

	sharedSummaries := make([]*models.CollectionSummary, 0, len(shared))
	for _, c := range shared {
		sharedSummaries = append(sharedSummaries, s.toSummary(c))
	}

	return &models.CollectionListResponse{
		Owned:  ownedSummaries,
		Shared: sharedSummaries,
	}, nil
}

// UpdateCollection updates a collection
func (s *CollectionService) UpdateCollection(ctx context.Context, collectionID, userID string, req *models.UpdateCollectionRequest) (*models.Collection, error) {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Only owner can edit
	if collection.UserID != userID {
		return nil, models.ErrCollectionAccessDenied
	}

	// Update fields
	if req.Name != nil {
		collection.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		collection.Description = req.Description
	}
	if req.Slug != nil && *req.Slug != "" {
		slug := s.sanitizeSlug(*req.Slug)
		exists, err := s.collectionRepo.SlugExists(ctx, slug, collectionID)
		if err != nil {
			return nil, fmt.Errorf("failed to check slug: %w", err)
		}
		if exists {
			return nil, models.ErrCollectionSlugExists
		}
		collection.Slug = slug
	}
	if req.Theme != nil {
		if !models.IsValidTheme(*req.Theme) {
			return nil, models.ErrCollectionInvalidTheme
		}
		collection.Theme = models.CollectionTheme(*req.Theme)
	}
	if req.CustomCSS != nil {
		collection.CustomCSS = req.CustomCSS
	}
	if req.CoverPhotoID != nil {
		// Verify the photo is in the collection
		if *req.CoverPhotoID != "" {
			inCollection, err := s.collectionPhotoRepo.IsPhotoInCollection(ctx, collectionID, *req.CoverPhotoID)
			if err != nil {
				return nil, fmt.Errorf("failed to verify cover photo: %w", err)
			}
			if !inCollection {
				return nil, fmt.Errorf("cover photo must be in the collection")
			}
		}
		collection.CoverPhotoID = req.CoverPhotoID
	}

	collection.UpdatedAt = time.Now().UTC()

	if err := s.collectionRepo.Update(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	return collection, nil
}

// UpdateVisibility changes collection visibility
func (s *CollectionService) UpdateVisibility(ctx context.Context, collectionID, userID string, visibility string) (*models.Collection, error) {
	if !models.IsValidVisibility(visibility) {
		return nil, models.ErrCollectionInvalidVisibility
	}

	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Only owner can change visibility
	if collection.UserID != userID {
		return nil, models.ErrCollectionAccessDenied
	}

	collection.SetVisibility(models.CollectionVisibility(visibility))

	if err := s.collectionRepo.Update(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to update visibility: %w", err)
	}

	return collection, nil
}

// DeleteCollection deletes a collection
func (s *CollectionService) DeleteCollection(ctx context.Context, collectionID, userID string) error {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return models.ErrCollectionNotFound
	}

	// Only owner can delete
	if collection.UserID != userID {
		return models.ErrCollectionAccessDenied
	}

	if err := s.collectionRepo.Delete(ctx, collectionID); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

// AddPhotos adds photos to a collection
func (s *CollectionService) AddPhotos(ctx context.Context, collectionID, userID string, photoIDs []string) error {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return models.ErrCollectionNotFound
	}

	// Only owner can add photos
	if collection.UserID != userID {
		return models.ErrCollectionAccessDenied
	}

	// Verify user owns all photos
	for _, photoID := range photoIDs {
		photo, err := s.photoRepo.GetByID(ctx, photoID)
		if err != nil {
			return fmt.Errorf("failed to verify photo: %w", err)
		}
		if photo == nil {
			return fmt.Errorf("photo not found: %s", photoID)
		}
		if photo.UserID == nil || *photo.UserID != userID {
			return models.ErrCollectionPhotoNotOwned
		}
	}

	if err := s.collectionPhotoRepo.AddMultiple(ctx, collectionID, photoIDs); err != nil {
		return fmt.Errorf("failed to add photos: %w", err)
	}

	// Update collection timestamp
	collection.UpdatedAt = time.Now().UTC()
	s.collectionRepo.Update(ctx, collection)

	return nil
}

// RemovePhotos removes photos from a collection
func (s *CollectionService) RemovePhotos(ctx context.Context, collectionID, userID string, photoIDs []string) error {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return models.ErrCollectionNotFound
	}

	// Only owner can remove photos
	if collection.UserID != userID {
		return models.ErrCollectionAccessDenied
	}

	if err := s.collectionPhotoRepo.RemoveMultiple(ctx, collectionID, photoIDs); err != nil {
		return fmt.Errorf("failed to remove photos: %w", err)
	}

	// Update collection timestamp
	collection.UpdatedAt = time.Now().UTC()
	s.collectionRepo.Update(ctx, collection)

	return nil
}

// ReorderPhotos reorders photos in a collection
func (s *CollectionService) ReorderPhotos(ctx context.Context, collectionID, userID string, photoIDs []string) error {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return models.ErrCollectionNotFound
	}

	// Only owner can reorder
	if collection.UserID != userID {
		return models.ErrCollectionAccessDenied
	}

	if err := s.collectionPhotoRepo.Reorder(ctx, collectionID, photoIDs); err != nil {
		return fmt.Errorf("failed to reorder photos: %w", err)
	}

	return nil
}

// GetPhotos returns photos in a collection
func (s *CollectionService) GetPhotos(ctx context.Context, collectionID, userID string) ([]*models.Photo, error) {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Check access
	if !s.canViewCollection(ctx, collection, userID) {
		return nil, models.ErrCollectionAccessDenied
	}

	photos, err := s.collectionPhotoRepo.GetPhotosForCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get photos: %w", err)
	}

	return photos, nil
}

// GetPhotosPublic returns photos for public/secret link access
func (s *CollectionService) GetPhotosPublic(ctx context.Context, collectionID string) ([]*models.Photo, error) {
	photos, err := s.collectionPhotoRepo.GetPhotosForCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get photos: %w", err)
	}
	return photos, nil
}

// ShareWithUsers shares a collection with users by email
func (s *CollectionService) ShareWithUsers(ctx context.Context, collectionID, userID string, emails []string) ([]string, error) {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Only owner can share
	if collection.UserID != userID {
		return nil, models.ErrCollectionAccessDenied
	}

	var failedEmails []string
	for _, email := range emails {
		user, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil || user == nil {
			failedEmails = append(failedEmails, email)
			continue
		}

		// Don't share with self
		if user.ID == userID {
			continue
		}

		share := models.NewCollectionShare(collectionID, user.ID)
		if err := s.collectionShareRepo.Add(ctx, share); err != nil {
			failedEmails = append(failedEmails, email)
		}
	}

	return failedEmails, nil
}

// RemoveShare removes a share from a collection
func (s *CollectionService) RemoveShare(ctx context.Context, collectionID, ownerID, targetUserID string) error {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return models.ErrCollectionNotFound
	}

	// Only owner can remove shares
	if collection.UserID != ownerID {
		return models.ErrCollectionAccessDenied
	}

	if err := s.collectionShareRepo.Remove(ctx, collectionID, targetUserID); err != nil {
		return fmt.Errorf("failed to remove share: %w", err)
	}

	return nil
}

// GetShares returns the users a collection is shared with
func (s *CollectionService) GetShares(ctx context.Context, collectionID, userID string) ([]*models.CollectionShareWithUser, error) {
	collection, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if collection == nil {
		return nil, models.ErrCollectionNotFound
	}

	// Only owner can see shares
	if collection.UserID != userID {
		return nil, models.ErrCollectionAccessDenied
	}

	shares, err := s.collectionShareRepo.GetSharesWithUsers(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}

	return shares, nil
}

// GetThemeCSS returns the CSS for a theme
func (s *CollectionService) GetThemeCSS(theme models.CollectionTheme) string {
	switch theme {
	case models.ThemeDark:
		return `
			:root {
				--bg-color: #0f172a;
				--card-color: #1e293b;
				--text-color: #f1f5f9;
				--text-muted: #94a3b8;
				--accent-color: #667eea;
				--border-color: #334155;
			}
		`
	case models.ThemeLight:
		return `
			:root {
				--bg-color: #ffffff;
				--card-color: #f3f4f6;
				--text-color: #111827;
				--text-muted: #6b7280;
				--accent-color: #4f46e5;
				--border-color: #e5e7eb;
			}
		`
	case models.ThemeMinimal:
		return `
			:root {
				--bg-color: #fafafa;
				--card-color: transparent;
				--text-color: #171717;
				--text-muted: #737373;
				--accent-color: #000000;
				--border-color: transparent;
			}
		`
	case models.ThemeGallery:
		return `
			:root {
				--bg-color: #1c1c1c;
				--card-color: #2a2a2a;
				--text-color: #ffffff;
				--text-muted: #a3a3a3;
				--accent-color: #d4af37;
				--border-color: #3a3a3a;
				--frame-width: 8px;
			}
		`
	case models.ThemeMagazine:
		return `
			:root {
				--bg-color: #f5f5f4;
				--card-color: #ffffff;
				--text-color: #1c1917;
				--text-muted: #78716c;
				--accent-color: #dc2626;
				--border-color: #e7e5e4;
				--font-family: Georgia, serif;
			}
		`
	default:
		return s.GetThemeCSS(models.ThemeDark)
	}
}

// Helper methods

func (s *CollectionService) canViewCollection(ctx context.Context, collection *models.Collection, userID string) bool {
	// Owner can always view
	if collection.UserID == userID {
		return true
	}

	// Public collections can be viewed by anyone
	if collection.Visibility == models.VisibilityPublic {
		return true
	}

	// Check if shared with user
	if collection.Visibility == models.VisibilityShared || collection.Visibility == models.VisibilitySecretLink {
		shared, err := s.collectionShareRepo.IsSharedWithUser(ctx, collection.ID, userID)
		if err == nil && shared {
			return true
		}
	}

	return false
}

func (s *CollectionService) sanitizeSlug(slug string) string {
	// Convert to lowercase
	slug = strings.ToLower(slug)

	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 100 {
		slug = slug[:100]
	}

	return slug
}

func (s *CollectionService) toSummary(c *models.Collection) *models.CollectionSummary {
	return &models.CollectionSummary{
		ID:           c.ID,
		Name:         c.Name,
		Slug:         c.Slug,
		Theme:        c.Theme,
		Visibility:   c.Visibility,
		PhotoCount:   c.PhotoCount,
		CoverPhotoID: c.CoverPhotoID,
		IsOwner:      c.IsOwner,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
}

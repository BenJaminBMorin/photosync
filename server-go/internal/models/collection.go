package models

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CollectionVisibility represents access levels for a collection
type CollectionVisibility string

const (
	VisibilityPrivate    CollectionVisibility = "private"     // Only owner can see
	VisibilityShared     CollectionVisibility = "shared"      // Owner + explicitly shared users
	VisibilitySecretLink CollectionVisibility = "secret_link" // Anyone with the secret token
	VisibilityPublic     CollectionVisibility = "public"      // Anyone can see via slug
)

// CollectionTheme represents predefined visual themes
type CollectionTheme string

const (
	ThemeDark     CollectionTheme = "dark"
	ThemeLight    CollectionTheme = "light"
	ThemeMinimal  CollectionTheme = "minimal"
	ThemeGallery  CollectionTheme = "gallery"
	ThemeMagazine CollectionTheme = "magazine"
)

// IsValidVisibility checks if a visibility value is valid
func IsValidVisibility(v string) bool {
	switch CollectionVisibility(v) {
	case VisibilityPrivate, VisibilityShared, VisibilitySecretLink, VisibilityPublic:
		return true
	}
	return false
}

// IsValidTheme checks if a theme value is valid
func IsValidTheme(t string) bool {
	switch CollectionTheme(t) {
	case ThemeDark, ThemeLight, ThemeMinimal, ThemeGallery, ThemeMagazine:
		return true
	}
	return false
}

// Collection represents a photo collection/gallery
type Collection struct {
	ID           string               `json:"id"`
	UserID       string               `json:"userId"`
	Name         string               `json:"name"`
	Description  *string              `json:"description,omitempty"`
	Slug         string               `json:"slug"`
	Theme        CollectionTheme      `json:"theme"`
	CustomCSS    *string              `json:"customCss,omitempty"`
	Visibility   CollectionVisibility `json:"visibility"`
	SecretToken  *string              `json:"secretToken,omitempty"`
	CoverPhotoID *string              `json:"coverPhotoId,omitempty"`
	CreatedAt    time.Time            `json:"createdAt"`
	UpdatedAt    time.Time            `json:"updatedAt"`

	// Computed fields (not stored in DB directly)
	PhotoCount int  `json:"photoCount,omitempty"`
	IsOwner    bool `json:"isOwner,omitempty"`
}

// NewCollection creates a new collection with generated ID and slug
func NewCollection(userID, name string) (*Collection, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrCollectionUserRequired
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrCollectionNameRequired
	}

	now := time.Now().UTC()
	slug := GenerateSlug(name)

	return &Collection{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       strings.TrimSpace(name),
		Slug:       slug,
		Theme:      ThemeDark,
		Visibility: VisibilityPrivate,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// GenerateSlug creates a URL-friendly slug from a name
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	// Add random suffix for uniqueness
	suffix := make([]byte, 4)
	rand.Read(suffix)
	slug = slug + "-" + hex.EncodeToString(suffix)

	return slug
}

// GenerateSecretToken creates a secure random token for secret link sharing
func GenerateSecretToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SetVisibility updates visibility and manages secret token
func (c *Collection) SetVisibility(visibility CollectionVisibility) {
	c.Visibility = visibility
	c.UpdatedAt = time.Now().UTC()

	// Generate secret token if switching to secret_link
	if visibility == VisibilitySecretLink && c.SecretToken == nil {
		token := GenerateSecretToken()
		c.SecretToken = &token
	}
}

// CanView checks if a user can view this collection
func (c *Collection) CanView(userID string) bool {
	// Owner can always view
	if c.UserID == userID {
		return true
	}

	// Public collections can be viewed by anyone
	if c.Visibility == VisibilityPublic {
		return true
	}

	// For other visibility levels, additional checks are needed
	// (shared users, secret tokens) - handled at service level
	return false
}

// CanEdit checks if a user can edit this collection
func (c *Collection) CanEdit(userID string) bool {
	return c.UserID == userID
}

// Collection errors
type CollectionError struct {
	Message string
}

func (e CollectionError) Error() string {
	return e.Message
}

var (
	ErrCollectionNotFound       = CollectionError{"collection not found"}
	ErrCollectionNameRequired   = CollectionError{"collection name is required"}
	ErrCollectionUserRequired   = CollectionError{"user ID is required"}
	ErrCollectionSlugExists     = CollectionError{"collection slug already exists"}
	ErrCollectionAccessDenied   = CollectionError{"access denied to collection"}
	ErrCollectionPhotoNotOwned  = CollectionError{"you can only add your own photos to a collection"}
	ErrCollectionInvalidTheme   = CollectionError{"invalid collection theme"}
	ErrCollectionInvalidVisibility = CollectionError{"invalid collection visibility"}
)

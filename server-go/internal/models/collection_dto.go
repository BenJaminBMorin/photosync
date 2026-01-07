package models

// CreateCollectionRequest is the request body for creating a collection
type CreateCollectionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Slug        *string `json:"slug,omitempty"` // Optional custom slug
	Theme       *string `json:"theme,omitempty"`
	CustomCSS   *string `json:"customCss,omitempty"`
}

// UpdateCollectionRequest is the request body for updating a collection
type UpdateCollectionRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Slug         *string `json:"slug,omitempty"`
	Theme        *string `json:"theme,omitempty"`
	CustomCSS    *string `json:"customCss,omitempty"`
	CoverPhotoID *string `json:"coverPhotoId,omitempty"`
}

// UpdateVisibilityRequest changes collection visibility
type UpdateVisibilityRequest struct {
	Visibility string `json:"visibility"`
}

// AddPhotosRequest adds photos to a collection
type AddPhotosRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

// RemovePhotosRequest removes photos from a collection
type RemovePhotosRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

// ReorderPhotosRequest reorders photos in a collection
type ReorderPhotosRequest struct {
	PhotoIDs []string `json:"photoIds"` // IDs in new order
}

// ShareCollectionRequest shares a collection with users
type ShareCollectionRequest struct {
	Emails []string `json:"emails"` // User emails to share with
}

// CollectionResponse is the API response for a single collection
type CollectionResponse struct {
	Collection     *Collection                `json:"collection"`
	Photos         []*CollectionPhotoWithDetails `json:"photos,omitempty"`
	Shares         []*CollectionShareWithUser    `json:"shares,omitempty"`
	ShareURL       string                     `json:"shareUrl,omitempty"`
	SecretURL      string                     `json:"secretUrl,omitempty"`
}

// CollectionListResponse is the API response for listing collections
type CollectionListResponse struct {
	Owned  []*CollectionSummary `json:"owned"`
	Shared []*CollectionSummary `json:"shared"`
}

// CollectionSummary is a brief view of a collection for lists
type CollectionSummary struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Slug         string               `json:"slug"`
	Theme        CollectionTheme      `json:"theme"`
	Visibility   CollectionVisibility `json:"visibility"`
	PhotoCount   int                  `json:"photoCount"`
	CoverPhotoID *string              `json:"coverPhotoId,omitempty"`
	CoverThumb   *string              `json:"coverThumb,omitempty"`
	IsOwner      bool                 `json:"isOwner"`
	CreatedAt    string               `json:"createdAt"`
	UpdatedAt    string               `json:"updatedAt"`
}

// ThemeInfo describes a theme for the theme selector
type ThemeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PreviewCSS  string `json:"previewCss"` // CSS variables preview
}

// ThemesResponse is the response for available themes
type ThemesResponse struct {
	Themes []ThemeInfo `json:"themes"`
}

// GetAvailableThemes returns the list of predefined themes
func GetAvailableThemes() []ThemeInfo {
	return []ThemeInfo{
		{
			ID:          string(ThemeDark),
			Name:        "Dark",
			Description: "Elegant night mode with deep blue background",
			PreviewCSS:  "--bg-color: #0f172a; --card-color: #1e293b; --accent-color: #667eea;",
		},
		{
			ID:          string(ThemeLight),
			Name:        "Light",
			Description: "Clean white background with modern styling",
			PreviewCSS:  "--bg-color: #ffffff; --card-color: #f3f4f6; --accent-color: #4f46e5;",
		},
		{
			ID:          string(ThemeMinimal),
			Name:        "Minimal",
			Description: "Ultra clean, borderless design",
			PreviewCSS:  "--bg-color: #fafafa; --card-color: transparent; --accent-color: #000000;",
		},
		{
			ID:          string(ThemeGallery),
			Name:        "Gallery",
			Description: "Classic gallery with elegant frames",
			PreviewCSS:  "--bg-color: #1c1c1c; --card-color: #2a2a2a; --accent-color: #d4af37;",
		},
		{
			ID:          string(ThemeMagazine),
			Name:        "Magazine",
			Description: "Editorial layout with serif typography",
			PreviewCSS:  "--bg-color: #f5f5f4; --card-color: #ffffff; --accent-color: #dc2626;",
		},
	}
}

// PublicGalleryData is the data passed to the public gallery template
type PublicGalleryData struct {
	Collection *Collection
	Photos     []*Photo
	ThemeCSS   string
	CustomCSS  string
	BaseURL    string
}

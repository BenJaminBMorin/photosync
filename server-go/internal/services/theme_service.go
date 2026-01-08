package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// ThemeService handles theme business logic and CSS generation
type ThemeService struct {
	themeRepo repository.ThemeRepository
	cache     *ThemeCache
}

// NewThemeService creates a new theme service
func NewThemeService(themeRepo repository.ThemeRepository) *ThemeService {
	return &ThemeService{
		themeRepo: themeRepo,
		cache:     NewThemeCache(1 * time.Hour), // 1-hour cache TTL
	}
}

// GetAll retrieves all themes
func (s *ThemeService) GetAll(ctx context.Context) ([]*models.Theme, error) {
	return s.themeRepo.GetAll(ctx)
}

// GetByID retrieves a theme by ID
func (s *ThemeService) GetByID(ctx context.Context, id string) (*models.Theme, error) {
	return s.themeRepo.GetByID(ctx, id)
}

// Create creates a new custom theme
func (s *ThemeService) Create(ctx context.Context, theme *models.Theme) error {
	// Validate theme
	if err := theme.Validate(); err != nil {
		return err
	}

	// Ensure it's not a system theme
	theme.IsSystem = false

	// Set timestamps
	now := time.Now()
	theme.CreatedAt = now
	theme.UpdatedAt = now

	// Create in repository
	if err := s.themeRepo.Create(ctx, theme); err != nil {
		return err
	}

	return nil
}

// Update updates an existing theme
func (s *ThemeService) Update(ctx context.Context, theme *models.Theme) error {
	// Validate theme
	if err := theme.Validate(); err != nil {
		return err
	}

	// Update in repository
	if err := s.themeRepo.Update(ctx, theme); err != nil {
		return err
	}

	// Clear cache for this theme
	s.cache.Delete(theme.ID)

	return nil
}

// Delete deletes a theme
func (s *ThemeService) Delete(ctx context.Context, id string) error {
	// Delete from repository
	if err := s.themeRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Clear cache
	s.cache.Delete(id)

	return nil
}

// GetSystemThemes retrieves only system themes
func (s *ThemeService) GetSystemThemes(ctx context.Context) ([]*models.Theme, error) {
	return s.themeRepo.GetSystemThemes(ctx)
}

// GenerateCSS generates CSS from theme properties with caching
func (s *ThemeService) GenerateCSS(ctx context.Context, themeID string) (string, error) {
	// Check cache first
	if css, found := s.cache.Get(themeID); found {
		return css, nil
	}

	// Fetch theme from repository
	theme, err := s.themeRepo.GetByID(ctx, themeID)
	if err != nil {
		return "", err
	}

	// Generate CSS
	css := generateCSSFromTheme(theme)

	// Cache for 1 hour
	s.cache.Set(themeID, css, 1*time.Hour)

	return css, nil
}

// ClearCache clears the entire CSS cache
func (s *ThemeService) ClearCache() {
	s.cache.Clear()
}

// generateCSSFromTheme converts ThemeProperties to CSS variables
func generateCSSFromTheme(theme *models.Theme) string {
	var css strings.Builder

	css.WriteString(":root {\n")

	// Color properties
	p := theme.Properties

	// Backgrounds
	css.WriteString(fmt.Sprintf("  --bg-primary: %s;\n", p.Colors.Backgrounds.Primary))
	css.WriteString(fmt.Sprintf("  --bg-secondary: %s;\n", p.Colors.Backgrounds.Secondary))
	css.WriteString(fmt.Sprintf("  --bg-tertiary: %s;\n", p.Colors.Backgrounds.Tertiary))
	css.WriteString(fmt.Sprintf("  --bg-elevated: %s;\n", p.Colors.Backgrounds.Elevated))
	css.WriteString(fmt.Sprintf("  --bg-overlay: %s;\n", p.Colors.Backgrounds.Overlay))
	// Use elevated as hover variant for tertiary (works for both light and dark themes)
	css.WriteString(fmt.Sprintf("  --bg-tertiary-hover: %s;\n", p.Colors.Backgrounds.Elevated))
	// Add semi-transparent secondary for dropdowns/overlays
	css.WriteString(fmt.Sprintf("  --bg-secondary-alpha: %s;\n", p.Colors.Backgrounds.Overlay))

	// Text colors
	css.WriteString(fmt.Sprintf("  --text-primary: %s;\n", p.Colors.Text.Primary))
	css.WriteString(fmt.Sprintf("  --text-secondary: %s;\n", p.Colors.Text.Secondary))
	css.WriteString(fmt.Sprintf("  --text-muted: %s;\n", p.Colors.Text.Muted))
	css.WriteString(fmt.Sprintf("  --text-disabled: %s;\n", p.Colors.Text.Disabled))
	css.WriteString(fmt.Sprintf("  --text-inverse: %s;\n", p.Colors.Text.Inverse))

	// Accent colors
	css.WriteString(fmt.Sprintf("  --accent-primary: %s;\n", p.Colors.Accents.Primary))
	css.WriteString(fmt.Sprintf("  --accent-secondary: %s;\n", p.Colors.Accents.Secondary))
	css.WriteString(fmt.Sprintf("  --accent-hover: %s;\n", p.Colors.Accents.Hover))
	css.WriteString(fmt.Sprintf("  --accent-active: %s;\n", p.Colors.Accents.Active))

	// Semantic colors
	css.WriteString(fmt.Sprintf("  --color-success: %s;\n", p.Colors.Semantic.Success))
	css.WriteString(fmt.Sprintf("  --color-warning: %s;\n", p.Colors.Semantic.Warning))
	css.WriteString(fmt.Sprintf("  --color-error: %s;\n", p.Colors.Semantic.Error))
	css.WriteString("  --color-error-dark: #dc2626;\n") // Darker variant for hover
	css.WriteString(fmt.Sprintf("  --color-info: %s;\n", p.Colors.Semantic.Info))

	// Border colors
	css.WriteString(fmt.Sprintf("  --border-color: %s;\n", p.Colors.Borders.Default))
	css.WriteString(fmt.Sprintf("  --border-color-hover: %s;\n", p.Colors.Borders.Hover))
	css.WriteString(fmt.Sprintf("  --border-focus: %s;\n", p.Colors.Borders.Focus))

	// Typography - Font families
	css.WriteString(fmt.Sprintf("  --font-family-base: %s;\n", p.Typography.FontFamilies.Base))
	css.WriteString(fmt.Sprintf("  --font-family-heading: %s;\n", p.Typography.FontFamilies.Heading))
	css.WriteString(fmt.Sprintf("  --font-family-accent: %s;\n", p.Typography.FontFamilies.Accent))
	css.WriteString(fmt.Sprintf("  --font-family-mono: %s;\n", p.Typography.FontFamilies.Mono))

	// Font sizes
	css.WriteString(fmt.Sprintf("  --font-size-xs: %s;\n", p.Typography.FontSizes.XS))
	css.WriteString(fmt.Sprintf("  --font-size-sm: %s;\n", p.Typography.FontSizes.SM))
	css.WriteString(fmt.Sprintf("  --font-size-base: %s;\n", p.Typography.FontSizes.Base))
	css.WriteString(fmt.Sprintf("  --font-size-lg: %s;\n", p.Typography.FontSizes.LG))
	css.WriteString(fmt.Sprintf("  --font-size-xl: %s;\n", p.Typography.FontSizes.XL))
	css.WriteString(fmt.Sprintf("  --font-size-2xl: %s;\n", p.Typography.FontSizes.XL2))
	css.WriteString(fmt.Sprintf("  --font-size-3xl: %s;\n", p.Typography.FontSizes.XL3))
	css.WriteString(fmt.Sprintf("  --font-size-4xl: %s;\n", p.Typography.FontSizes.XL4))

	// Font weights
	css.WriteString(fmt.Sprintf("  --font-weight-light: %s;\n", p.Typography.FontWeights.Light))
	css.WriteString(fmt.Sprintf("  --font-weight-normal: %s;\n", p.Typography.FontWeights.Normal))
	css.WriteString(fmt.Sprintf("  --font-weight-medium: %s;\n", p.Typography.FontWeights.Medium))
	css.WriteString(fmt.Sprintf("  --font-weight-semibold: %s;\n", p.Typography.FontWeights.Semibold))
	css.WriteString(fmt.Sprintf("  --font-weight-bold: %s;\n", p.Typography.FontWeights.Bold))

	// Line heights
	css.WriteString(fmt.Sprintf("  --line-height-tight: %s;\n", p.Typography.LineHeights.Tight))
	css.WriteString(fmt.Sprintf("  --line-height-normal: %s;\n", p.Typography.LineHeights.Normal))
	css.WriteString(fmt.Sprintf("  --line-height-relaxed: %s;\n", p.Typography.LineHeights.Relaxed))
	css.WriteString(fmt.Sprintf("  --line-height-loose: %s;\n", p.Typography.LineHeights.Loose))

	// Letter spacing
	css.WriteString(fmt.Sprintf("  --letter-spacing-tight: %s;\n", p.Typography.LetterSpacing.Tight))
	css.WriteString(fmt.Sprintf("  --letter-spacing-normal: %s;\n", p.Typography.LetterSpacing.Normal))
	css.WriteString(fmt.Sprintf("  --letter-spacing-wide: %s;\n", p.Typography.LetterSpacing.Wide))

	// Layout - Spacing
	css.WriteString(fmt.Sprintf("  --spacing-xs: %s;\n", p.Layout.Spacing.XS))
	css.WriteString(fmt.Sprintf("  --spacing-sm: %s;\n", p.Layout.Spacing.SM))
	css.WriteString(fmt.Sprintf("  --spacing-md: %s;\n", p.Layout.Spacing.MD))
	css.WriteString(fmt.Sprintf("  --spacing-lg: %s;\n", p.Layout.Spacing.LG))
	css.WriteString(fmt.Sprintf("  --spacing-xl: %s;\n", p.Layout.Spacing.XL))
	css.WriteString(fmt.Sprintf("  --spacing-2xl: %s;\n", p.Layout.Spacing.XL2))

	// Border radius
	css.WriteString(fmt.Sprintf("  --border-radius-sm: %s;\n", p.Layout.BorderRadius.SM))
	css.WriteString(fmt.Sprintf("  --border-radius-md: %s;\n", p.Layout.BorderRadius.MD))
	css.WriteString(fmt.Sprintf("  --border-radius-lg: %s;\n", p.Layout.BorderRadius.LG))
	css.WriteString(fmt.Sprintf("  --border-radius-xl: %s;\n", p.Layout.BorderRadius.XL))
	css.WriteString(fmt.Sprintf("  --border-radius-2xl: %s;\n", p.Layout.BorderRadius.XL2))
	css.WriteString(fmt.Sprintf("  --border-radius-full: %s;\n", p.Layout.BorderRadius.Full))

	// Grid
	css.WriteString(fmt.Sprintf("  --grid-cols-min: %s;\n", p.Layout.Grid.ColsMin))
	css.WriteString(fmt.Sprintf("  --grid-gap: %s;\n", p.Layout.Grid.Gap))
	css.WriteString(fmt.Sprintf("  --grid-card-padding: %s;\n", p.Layout.Grid.CardPadding))
	css.WriteString(fmt.Sprintf("  --grid-card-aspect-ratio: %s;\n", p.Layout.Grid.CardAspectRatio))

	// Max widths
	css.WriteString(fmt.Sprintf("  --max-width-container: %s;\n", p.Layout.MaxWidths.Container))
	css.WriteString(fmt.Sprintf("  --max-width-modal: %s;\n", p.Layout.MaxWidths.Modal))

	// Effects - Shadows
	css.WriteString(fmt.Sprintf("  --shadow-sm: %s;\n", p.Effects.Shadows.SM))
	css.WriteString(fmt.Sprintf("  --shadow-md: %s;\n", p.Effects.Shadows.MD))
	css.WriteString(fmt.Sprintf("  --shadow-lg: %s;\n", p.Effects.Shadows.LG))
	css.WriteString(fmt.Sprintf("  --shadow-xl: %s;\n", p.Effects.Shadows.XL))
	// Legacy shadow-color for compatibility
	css.WriteString("  --shadow-color: rgba(0, 0, 0, 0.4);\n")

	// Transitions
	css.WriteString(fmt.Sprintf("  --transition-fast: %s;\n", p.Effects.Transitions.Fast))
	css.WriteString(fmt.Sprintf("  --transition-normal: %s;\n", p.Effects.Transitions.Normal))
	css.WriteString(fmt.Sprintf("  --transition-slow: %s;\n", p.Effects.Transitions.Slow))

	// Blur
	css.WriteString(fmt.Sprintf("  --blur-sm: %s;\n", p.Effects.Blur.SM))
	css.WriteString(fmt.Sprintf("  --blur-md: %s;\n", p.Effects.Blur.MD))
	css.WriteString(fmt.Sprintf("  --blur-lg: %s;\n", p.Effects.Blur.LG))

	// Opacity
	css.WriteString(fmt.Sprintf("  --opacity-disabled: %s;\n", p.Effects.Opacity.Disabled))
	css.WriteString(fmt.Sprintf("  --opacity-muted: %s;\n", p.Effects.Opacity.Muted))

	// Animations
	if p.Animations.Enabled {
		css.WriteString("  --animations-enabled: 1;\n")
	} else {
		css.WriteString("  --animations-enabled: 0;\n")
	}
	css.WriteString(fmt.Sprintf("  --animation-hover-scale: %s;\n", p.Animations.HoverScale))
	css.WriteString(fmt.Sprintf("  --animation-press-scale: %s;\n", p.Animations.PressScale))

	// Transform helpers used in frontend
	css.WriteString("  --hover-scale: translateY(-2px);\n")

	css.WriteString("}\n")

	return css.String()
}

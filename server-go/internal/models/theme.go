package models

import (
	"fmt"
	"time"
)

// Theme represents a comprehensive theme definition
type Theme struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	IsSystem    bool             `json:"isSystem"`
	CreatedBy   *string          `json:"createdBy,omitempty"`
	Properties  ThemeProperties  `json:"properties"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// ThemeProperties contains all visual properties for a theme
type ThemeProperties struct {
	Colors     ColorProperties      `json:"colors"`
	Typography TypographyProperties `json:"typography"`
	Layout     LayoutProperties     `json:"layout"`
	Effects    EffectProperties     `json:"effects"`
	Animations AnimationProperties  `json:"animations"`
}

// ColorProperties defines all color variables for a theme
type ColorProperties struct {
	Backgrounds BackgroundColors `json:"backgrounds"`
	Text        TextColors       `json:"text"`
	Accents     AccentColors     `json:"accents"`
	Semantic    SemanticColors   `json:"semantic"`
	Borders     BorderColors     `json:"borders"`
}

// BackgroundColors defines background color variables
type BackgroundColors struct {
	Primary   string `json:"primary" example:"#0f172a"`
	Secondary string `json:"secondary" example:"#1e293b"`
	Tertiary  string `json:"tertiary" example:"#334155"`
	Elevated  string `json:"elevated" example:"#1e293b"`
	Overlay   string `json:"overlay" example:"rgba(15, 23, 42, 0.95)"`
}

// TextColors defines text color variables
type TextColors struct {
	Primary  string `json:"primary" example:"#ffffff"`
	Secondary string `json:"secondary" example:"#e2e8f0"`
	Muted    string `json:"muted" example:"#94a3b8"`
	Disabled string `json:"disabled" example:"#64748b"`
	Inverse  string `json:"inverse" example:"#0f172a"`
}

// AccentColors defines accent and brand color variables
type AccentColors struct {
	Primary   string `json:"primary" example:"#667eea"`
	Secondary string `json:"secondary" example:"#764ba2"`
	Hover     string `json:"hover" example:"#5a67d8"`
	Active    string `json:"active" example:"#4c51bf"`
}

// SemanticColors defines semantic color variables for status indicators
type SemanticColors struct {
	Success string `json:"success" example:"#10b981"`
	Warning string `json:"warning" example:"#f59e0b"`
	Error   string `json:"error" example:"#ef4444"`
	Info    string `json:"info" example:"#3b82f6"`
}

// BorderColors defines border color variables
type BorderColors struct {
	Default string `json:"default" example:"rgba(100, 116, 139, 0.2)"`
	Hover   string `json:"hover" example:"rgba(100, 116, 139, 0.4)"`
	Focus   string `json:"focus" example:"rgba(102, 126, 234, 0.5)"`
}

// TypographyProperties defines all typography variables for a theme
type TypographyProperties struct {
	FontFamilies  FontFamilies  `json:"fontFamilies"`
	FontSizes     FontSizes     `json:"fontSizes"`
	FontWeights   FontWeights   `json:"fontWeights"`
	LineHeights   LineHeights   `json:"lineHeights"`
	LetterSpacing LetterSpacing `json:"letterSpacing"`
}

// FontFamilies defines font family variables
type FontFamilies struct {
	Base    string `json:"base" example:"-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"`
	Heading string `json:"heading" example:"inherit"`
	Accent  string `json:"accent" example:"inherit"`
	Mono    string `json:"mono" example:"monospace"`
}

// FontSizes defines font size variables
type FontSizes struct {
	XS   string `json:"xs" example:"12px"`
	SM   string `json:"sm" example:"14px"`
	Base string `json:"base" example:"16px"`
	LG   string `json:"lg" example:"18px"`
	XL   string `json:"xl" example:"20px"`
	XL2  string `json:"2xl" example:"24px"`
	XL3  string `json:"3xl" example:"28px"`
	XL4  string `json:"4xl" example:"32px"`
}

// FontWeights defines font weight variables
type FontWeights struct {
	Light    string `json:"light" example:"300"`
	Normal   string `json:"normal" example:"400"`
	Medium   string `json:"medium" example:"500"`
	Semibold string `json:"semibold" example:"600"`
	Bold     string `json:"bold" example:"700"`
}

// LineHeights defines line height variables
type LineHeights struct {
	Tight   string `json:"tight" example:"1.2"`
	Normal  string `json:"normal" example:"1.5"`
	Relaxed string `json:"relaxed" example:"1.75"`
	Loose   string `json:"loose" example:"2"`
}

// LetterSpacing defines letter spacing variables
type LetterSpacing struct {
	Tight  string `json:"tight" example:"-0.02em"`
	Normal string `json:"normal" example:"0"`
	Wide   string `json:"wide" example:"0.02em"`
}

// LayoutProperties defines all layout variables for a theme
type LayoutProperties struct {
	Spacing    Spacing    `json:"spacing"`
	BorderRadius BorderRadius `json:"borderRadius"`
	Grid       Grid       `json:"grid"`
	MaxWidths  MaxWidths  `json:"maxWidths"`
}

// Spacing defines spacing scale variables
type Spacing struct {
	XS  string `json:"xs" example:"4px"`
	SM  string `json:"sm" example:"8px"`
	MD  string `json:"md" example:"16px"`
	LG  string `json:"lg" example:"24px"`
	XL  string `json:"xl" example:"32px"`
	XL2 string `json:"2xl" example:"48px"`
}

// BorderRadius defines border radius variables
type BorderRadius struct {
	SM   string `json:"sm" example:"2px"`
	MD   string `json:"md" example:"4px"`
	LG   string `json:"lg" example:"8px"`
	XL   string `json:"xl" example:"12px"`
	XL2  string `json:"2xl" example:"16px"`
	Full string `json:"full" example:"9999px"`
}

// Grid defines grid layout variables
type Grid struct {
	ColsMin        string `json:"colsMin" example:"200px"`
	Gap            string `json:"gap" example:"16px"`
	CardPadding    string `json:"cardPadding" example:"16px"`
	CardAspectRatio string `json:"cardAspectRatio" example:"1"`
}

// MaxWidths defines maximum width variables
type MaxWidths struct {
	Container string `json:"container" example:"1400px"`
	Modal     string `json:"modal" example:"600px"`
}

// EffectProperties defines all visual effect variables for a theme
type EffectProperties struct {
	Shadows     Shadows     `json:"shadows"`
	Transitions Transitions `json:"transitions"`
	Blur        Blur        `json:"blur"`
	Opacity     Opacity     `json:"opacity"`
}

// Shadows defines box shadow variables
type Shadows struct {
	SM string `json:"sm" example:"0 1px 2px rgba(0, 0, 0, 0.05)"`
	MD string `json:"md" example:"0 4px 6px rgba(0, 0, 0, 0.1)"`
	LG string `json:"lg" example:"0 10px 20px rgba(0, 0, 0, 0.15)"`
	XL string `json:"xl" example:"0 20px 40px rgba(0, 0, 0, 0.2)"`
}

// Transitions defines transition timing variables
type Transitions struct {
	Fast   string `json:"fast" example:"0.1s cubic-bezier(0.4, 0, 0.2, 1)"`
	Normal string `json:"normal" example:"0.3s cubic-bezier(0.4, 0, 0.2, 1)"`
	Slow   string `json:"slow" example:"0.5s cubic-bezier(0.4, 0, 0.2, 1)"`
}

// Blur defines blur effect variables
type Blur struct {
	SM string `json:"sm" example:"4px"`
	MD string `json:"md" example:"8px"`
	LG string `json:"lg" example:"16px"`
}

// Opacity defines opacity variables
type Opacity struct {
	Disabled string `json:"disabled" example:"0.5"`
	Muted    string `json:"muted" example:"0.7"`
}

// AnimationProperties defines animation-related variables
type AnimationProperties struct {
	Enabled    bool   `json:"enabled" example:"true"`
	HoverScale string `json:"hoverScale" example:"1.02"`
	PressScale string `json:"pressScale" example:"0.98"`
}

// ValidateTheme checks if a theme has all required properties
func (t *Theme) Validate() error {
	if t.ID == "" {
		return ErrInvalidThemeID
	}
	if t.Name == "" {
		return ErrInvalidThemeName
	}
	// Add more validation as needed
	return nil
}

// ToThemeInfo converts a Theme to a lightweight ThemeInfo for public listing
func (t *Theme) ToThemeInfo() ThemeInfo {
	description := ""
	if t.Description != nil {
		description = *t.Description
	}
	return ThemeInfo{
		ID:          t.ID,
		Name:        t.Name,
		Description: description,
		PreviewCSS:  fmt.Sprintf("--bg-primary: %s; --accent-primary: %s;", t.Properties.Colors.Backgrounds.Primary, t.Properties.Colors.Accents.Primary),
	}
}

// Common theme-related errors
var (
	ErrInvalidThemeID   = fmt.Errorf("theme ID is required")
	ErrInvalidThemeName = fmt.Errorf("theme name is required")
	ErrThemeNotFound    = fmt.Errorf("theme not found")
	ErrSystemThemeEdit  = fmt.Errorf("system themes cannot be modified")
)

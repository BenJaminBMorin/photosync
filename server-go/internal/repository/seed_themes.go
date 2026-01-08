package repository

import (
	"context"
	"time"

	"github.com/photosync/server/internal/models"
)

// SeedSystemThemes creates the 5 default system themes if they don't exist
func SeedSystemThemes(ctx context.Context, repo ThemeRepository) error {
	themes := getSystemThemes()

	for _, theme := range themes {
		// Check if theme already exists
		existing, err := repo.GetByID(ctx, theme.ID)
		if err == nil && existing != nil {
			// Theme exists, skip
			continue
		}

		// Create theme
		if err := repo.Create(ctx, theme); err != nil {
			return err
		}
	}

	return nil
}

func getSystemThemes() []*models.Theme {
	now := time.Now()

	return []*models.Theme{
		getDarkTheme(now),
		getLightTheme(now),
		getMinimalTheme(now),
		getGalleryTheme(now),
		getMagazineTheme(now),
	}
}

func getDarkTheme(now time.Time) *models.Theme {
	desc := "Classic dark theme with deep blues and purple accents"
	return &models.Theme{
		ID:          "dark",
		Name:        "Dark",
		Description: &desc,
		IsSystem:    true,
		CreatedBy:   nil,
		Properties: models.ThemeProperties{
			Colors: models.ColorProperties{
				Backgrounds: models.BackgroundColors{
					Primary:   "#0f172a",
					Secondary: "#1e293b",
					Tertiary:  "#334155",
					Elevated:  "#1e293b",
					Overlay:   "rgba(15, 23, 42, 0.95)",
				},
				Text: models.TextColors{
					Primary:   "#ffffff",
					Secondary: "#e2e8f0",
					Muted:     "#94a3b8",
					Disabled:  "#64748b",
					Inverse:   "#0f172a",
				},
				Accents: models.AccentColors{
					Primary:   "#667eea",
					Secondary: "#764ba2",
					Hover:     "#5a67d8",
					Active:    "#4c51bf",
				},
				Semantic: models.SemanticColors{
					Success: "#10b981",
					Warning: "#f59e0b",
					Error:   "#ef4444",
					Info:    "#3b82f6",
				},
				Borders: models.BorderColors{
					Default: "rgba(100, 116, 139, 0.2)",
					Hover:   "rgba(100, 116, 139, 0.4)",
					Focus:   "rgba(102, 126, 234, 0.5)",
				},
			},
			Typography: models.TypographyProperties{
				FontFamilies: models.FontFamilies{
					Base:    "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
					Heading: "inherit",
					Accent:  "inherit",
					Mono:    "monospace",
				},
				FontSizes: models.FontSizes{
					XS:   "12px",
					SM:   "14px",
					Base: "16px",
					LG:   "18px",
					XL:   "20px",
					XL2:  "24px",
					XL3:  "28px",
					XL4:  "32px",
				},
				FontWeights: models.FontWeights{
					Light:    "300",
					Normal:   "400",
					Medium:   "500",
					Semibold: "600",
					Bold:     "700",
				},
				LineHeights: models.LineHeights{
					Tight:   "1.2",
					Normal:  "1.5",
					Relaxed: "1.75",
					Loose:   "2",
				},
				LetterSpacing: models.LetterSpacing{
					Tight:  "-0.02em",
					Normal: "0",
					Wide:   "0.02em",
				},
			},
			Layout: models.LayoutProperties{
				Spacing: models.Spacing{
					XS:  "4px",
					SM:  "8px",
					MD:  "16px",
					LG:  "24px",
					XL:  "32px",
					XL2: "48px",
				},
				BorderRadius: models.BorderRadius{
					SM:   "2px",
					MD:   "4px",
					LG:   "8px",
					XL:   "12px",
					XL2:  "16px",
					Full: "9999px",
				},
				Grid: models.Grid{
					ColsMin:         "200px",
					Gap:             "16px",
					CardPadding:     "16px",
					CardAspectRatio: "1",
				},
				MaxWidths: models.MaxWidths{
					Container: "1400px",
					Modal:     "600px",
				},
			},
			Effects: models.EffectProperties{
				Shadows: models.Shadows{
					SM: "0 1px 2px rgba(0, 0, 0, 0.05)",
					MD: "0 4px 6px rgba(0, 0, 0, 0.1)",
					LG: "0 10px 20px rgba(0, 0, 0, 0.15)",
					XL: "0 20px 40px rgba(0, 0, 0, 0.2)",
				},
				Transitions: models.Transitions{
					Fast:   "0.1s cubic-bezier(0.4, 0, 0.2, 1)",
					Normal: "0.3s cubic-bezier(0.4, 0, 0.2, 1)",
					Slow:   "0.5s cubic-bezier(0.4, 0, 0.2, 1)",
				},
				Blur: models.Blur{
					SM: "4px",
					MD: "8px",
					LG: "16px",
				},
				Opacity: models.Opacity{
					Disabled: "0.5",
					Muted:    "0.7",
				},
			},
			Animations: models.AnimationProperties{
				Enabled:    true,
				HoverScale: "1.02",
				PressScale: "0.98",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func getLightTheme(now time.Time) *models.Theme {
	desc := "Clean light theme with vibrant colors"
	return &models.Theme{
		ID:          "light",
		Name:        "Light",
		Description: &desc,
		IsSystem:    true,
		CreatedBy:   nil,
		Properties: models.ThemeProperties{
			Colors: models.ColorProperties{
				Backgrounds: models.BackgroundColors{
					Primary:   "#ffffff",
					Secondary: "#f8fafc",
					Tertiary:  "#f1f5f9",
					Elevated:  "#ffffff",
					Overlay:   "rgba(255, 255, 255, 0.95)",
				},
				Text: models.TextColors{
					Primary:   "#0f172a",
					Secondary: "#334155",
					Muted:     "#64748b",
					Disabled:  "#94a3b8",
					Inverse:   "#ffffff",
				},
				Accents: models.AccentColors{
					Primary:   "#667eea",
					Secondary: "#764ba2",
					Hover:     "#5a67d8",
					Active:    "#4c51bf",
				},
				Semantic: models.SemanticColors{
					Success: "#10b981",
					Warning: "#f59e0b",
					Error:   "#ef4444",
					Info:    "#3b82f6",
				},
				Borders: models.BorderColors{
					Default: "rgba(15, 23, 42, 0.1)",
					Hover:   "rgba(15, 23, 42, 0.2)",
					Focus:   "rgba(102, 126, 234, 0.5)",
				},
			},
			Typography: models.TypographyProperties{
				FontFamilies: models.FontFamilies{
					Base:    "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
					Heading: "inherit",
					Accent:  "inherit",
					Mono:    "monospace",
				},
				FontSizes: models.FontSizes{
					XS:   "12px",
					SM:   "14px",
					Base: "16px",
					LG:   "18px",
					XL:   "20px",
					XL2:  "24px",
					XL3:  "28px",
					XL4:  "32px",
				},
				FontWeights: models.FontWeights{
					Light:    "300",
					Normal:   "400",
					Medium:   "500",
					Semibold: "600",
					Bold:     "700",
				},
				LineHeights: models.LineHeights{
					Tight:   "1.2",
					Normal:  "1.5",
					Relaxed: "1.75",
					Loose:   "2",
				},
				LetterSpacing: models.LetterSpacing{
					Tight:  "-0.02em",
					Normal: "0",
					Wide:   "0.02em",
				},
			},
			Layout: models.LayoutProperties{
				Spacing: models.Spacing{
					XS:  "4px",
					SM:  "8px",
					MD:  "16px",
					LG:  "24px",
					XL:  "32px",
					XL2: "48px",
				},
				BorderRadius: models.BorderRadius{
					SM:   "2px",
					MD:   "4px",
					LG:   "8px",
					XL:   "12px",
					XL2:  "16px",
					Full: "9999px",
				},
				Grid: models.Grid{
					ColsMin:         "200px",
					Gap:             "16px",
					CardPadding:     "16px",
					CardAspectRatio: "1",
				},
				MaxWidths: models.MaxWidths{
					Container: "1400px",
					Modal:     "600px",
				},
			},
			Effects: models.EffectProperties{
				Shadows: models.Shadows{
					SM: "0 1px 2px rgba(0, 0, 0, 0.05)",
					MD: "0 4px 6px rgba(0, 0, 0, 0.1)",
					LG: "0 10px 20px rgba(0, 0, 0, 0.15)",
					XL: "0 20px 40px rgba(0, 0, 0, 0.2)",
				},
				Transitions: models.Transitions{
					Fast:   "0.1s cubic-bezier(0.4, 0, 0.2, 1)",
					Normal: "0.3s cubic-bezier(0.4, 0, 0.2, 1)",
					Slow:   "0.5s cubic-bezier(0.4, 0, 0.2, 1)",
				},
				Blur: models.Blur{
					SM: "4px",
					MD: "8px",
					LG: "16px",
				},
				Opacity: models.Opacity{
					Disabled: "0.5",
					Muted:    "0.7",
				},
			},
			Animations: models.AnimationProperties{
				Enabled:    true,
				HoverScale: "1.02",
				PressScale: "0.98",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func getMinimalTheme(now time.Time) *models.Theme {
	desc := "Minimalist design with neutral colors and clean typography"
	return &models.Theme{
		ID:          "minimal",
		Name:        "Minimal",
		Description: &desc,
		IsSystem:    true,
		CreatedBy:   nil,
		Properties: models.ThemeProperties{
			Colors: models.ColorProperties{
				Backgrounds: models.BackgroundColors{
					Primary:   "#fafafa",
					Secondary: "#f5f5f5",
					Tertiary:  "#eeeeee",
					Elevated:  "#ffffff",
					Overlay:   "rgba(250, 250, 250, 0.95)",
				},
				Text: models.TextColors{
					Primary:   "#212121",
					Secondary: "#424242",
					Muted:     "#757575",
					Disabled:  "#9e9e9e",
					Inverse:   "#ffffff",
				},
				Accents: models.AccentColors{
					Primary:   "#424242",
					Secondary: "#616161",
					Hover:     "#212121",
					Active:    "#000000",
				},
				Semantic: models.SemanticColors{
					Success: "#4caf50",
					Warning: "#ff9800",
					Error:   "#f44336",
					Info:    "#2196f3",
				},
				Borders: models.BorderColors{
					Default: "rgba(0, 0, 0, 0.08)",
					Hover:   "rgba(0, 0, 0, 0.16)",
					Focus:   "rgba(0, 0, 0, 0.24)",
				},
			},
			Typography: models.TypographyProperties{
				FontFamilies: models.FontFamilies{
					Base:    "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
					Heading: "inherit",
					Accent:  "inherit",
					Mono:    "'Courier New', monospace",
				},
				FontSizes: models.FontSizes{
					XS:   "12px",
					SM:   "14px",
					Base: "15px",
					LG:   "17px",
					XL:   "19px",
					XL2:  "22px",
					XL3:  "26px",
					XL4:  "30px",
				},
				FontWeights: models.FontWeights{
					Light:    "300",
					Normal:   "400",
					Medium:   "400",
					Semibold: "500",
					Bold:     "600",
				},
				LineHeights: models.LineHeights{
					Tight:   "1.3",
					Normal:  "1.6",
					Relaxed: "1.8",
					Loose:   "2",
				},
				LetterSpacing: models.LetterSpacing{
					Tight:  "0",
					Normal: "0.01em",
					Wide:   "0.03em",
				},
			},
			Layout: models.LayoutProperties{
				Spacing: models.Spacing{
					XS:  "4px",
					SM:  "8px",
					MD:  "16px",
					LG:  "32px",
					XL:  "48px",
					XL2: "64px",
				},
				BorderRadius: models.BorderRadius{
					SM:   "0px",
					MD:   "2px",
					LG:   "4px",
					XL:   "8px",
					XL2:  "12px",
					Full: "9999px",
				},
				Grid: models.Grid{
					ColsMin:         "220px",
					Gap:             "24px",
					CardPadding:     "0px",
					CardAspectRatio: "1",
				},
				MaxWidths: models.MaxWidths{
					Container: "1200px",
					Modal:     "500px",
				},
			},
			Effects: models.EffectProperties{
				Shadows: models.Shadows{
					SM: "0 1px 2px rgba(0, 0, 0, 0.04)",
					MD: "0 2px 4px rgba(0, 0, 0, 0.06)",
					LG: "0 4px 8px rgba(0, 0, 0, 0.08)",
					XL: "0 8px 16px rgba(0, 0, 0, 0.1)",
				},
				Transitions: models.Transitions{
					Fast:   "0.1s ease",
					Normal: "0.2s ease",
					Slow:   "0.4s ease",
				},
				Blur: models.Blur{
					SM: "2px",
					MD: "4px",
					LG: "8px",
				},
				Opacity: models.Opacity{
					Disabled: "0.4",
					Muted:    "0.6",
				},
			},
			Animations: models.AnimationProperties{
				Enabled:    false,
				HoverScale: "1",
				PressScale: "1",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func getGalleryTheme(now time.Time) *models.Theme {
	desc := "Museum-style gallery with elegant presentation"
	return &models.Theme{
		ID:          "gallery",
		Name:        "Gallery",
		Description: &desc,
		IsSystem:    true,
		CreatedBy:   nil,
		Properties: models.ThemeProperties{
			Colors: models.ColorProperties{
				Backgrounds: models.BackgroundColors{
					Primary:   "#1a1a1a",
					Secondary: "#2a2a2a",
					Tertiary:  "#3a3a3a",
					Elevated:  "#2a2a2a",
					Overlay:   "rgba(26, 26, 26, 0.98)",
				},
				Text: models.TextColors{
					Primary:   "#f5f5f5",
					Secondary: "#d4d4d4",
					Muted:     "#a3a3a3",
					Disabled:  "#737373",
					Inverse:   "#1a1a1a",
				},
				Accents: models.AccentColors{
					Primary:   "#d4af37",
					Secondary: "#c9a227",
					Hover:     "#e5c148",
					Active:    "#b89b26",
				},
				Semantic: models.SemanticColors{
					Success: "#22c55e",
					Warning: "#eab308",
					Error:   "#dc2626",
					Info:    "#3b82f6",
				},
				Borders: models.BorderColors{
					Default: "rgba(212, 175, 55, 0.2)",
					Hover:   "rgba(212, 175, 55, 0.4)",
					Focus:   "rgba(212, 175, 55, 0.6)",
				},
			},
			Typography: models.TypographyProperties{
				FontFamilies: models.FontFamilies{
					Base:    "'Playfair Display', Georgia, serif",
					Heading: "'Playfair Display', Georgia, serif",
					Accent:  "'Cinzel', serif",
					Mono:    "monospace",
				},
				FontSizes: models.FontSizes{
					XS:   "12px",
					SM:   "14px",
					Base: "16px",
					LG:   "19px",
					XL:   "22px",
					XL2:  "28px",
					XL3:  "34px",
					XL4:  "40px",
				},
				FontWeights: models.FontWeights{
					Light:    "300",
					Normal:   "400",
					Medium:   "500",
					Semibold: "600",
					Bold:     "700",
				},
				LineHeights: models.LineHeights{
					Tight:   "1.3",
					Normal:  "1.6",
					Relaxed: "1.8",
					Loose:   "2.2",
				},
				LetterSpacing: models.LetterSpacing{
					Tight:  "-0.01em",
					Normal: "0.01em",
					Wide:   "0.05em",
				},
			},
			Layout: models.LayoutProperties{
				Spacing: models.Spacing{
					XS:  "8px",
					SM:  "16px",
					MD:  "24px",
					LG:  "48px",
					XL:  "64px",
					XL2: "96px",
				},
				BorderRadius: models.BorderRadius{
					SM:   "0px",
					MD:   "0px",
					LG:   "2px",
					XL:   "4px",
					XL2:  "8px",
					Full: "9999px",
				},
				Grid: models.Grid{
					ColsMin:         "280px",
					Gap:             "32px",
					CardPadding:     "20px",
					CardAspectRatio: "0.8",
				},
				MaxWidths: models.MaxWidths{
					Container: "1600px",
					Modal:     "700px",
				},
			},
			Effects: models.EffectProperties{
				Shadows: models.Shadows{
					SM: "0 2px 4px rgba(0, 0, 0, 0.2)",
					MD: "0 8px 16px rgba(0, 0, 0, 0.3)",
					LG: "0 16px 32px rgba(0, 0, 0, 0.4)",
					XL: "0 32px 64px rgba(0, 0, 0, 0.5)",
				},
				Transitions: models.Transitions{
					Fast:   "0.2s cubic-bezier(0.4, 0, 0.2, 1)",
					Normal: "0.4s cubic-bezier(0.4, 0, 0.2, 1)",
					Slow:   "0.6s cubic-bezier(0.4, 0, 0.2, 1)",
				},
				Blur: models.Blur{
					SM: "6px",
					MD: "12px",
					LG: "24px",
				},
				Opacity: models.Opacity{
					Disabled: "0.5",
					Muted:    "0.8",
				},
			},
			Animations: models.AnimationProperties{
				Enabled:    true,
				HoverScale: "1.05",
				PressScale: "0.95",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func getMagazineTheme(now time.Time) *models.Theme {
	desc := "Editorial magazine layout with bold typography"
	return &models.Theme{
		ID:          "magazine",
		Name:        "Magazine",
		Description: &desc,
		IsSystem:    true,
		CreatedBy:   nil,
		Properties: models.ThemeProperties{
			Colors: models.ColorProperties{
				Backgrounds: models.BackgroundColors{
					Primary:   "#ffffff",
					Secondary: "#fafafa",
					Tertiary:  "#f0f0f0",
					Elevated:  "#ffffff",
					Overlay:   "rgba(255, 255, 255, 0.98)",
				},
				Text: models.TextColors{
					Primary:   "#1a1a1a",
					Secondary: "#4a4a4a",
					Muted:     "#7a7a7a",
					Disabled:  "#aaaaaa",
					Inverse:   "#ffffff",
				},
				Accents: models.AccentColors{
					Primary:   "#e63946",
					Secondary: "#d62828",
					Hover:     "#f14c57",
					Active:    "#c71f2f",
				},
				Semantic: models.SemanticColors{
					Success: "#06d6a0",
					Warning: "#ffb703",
					Error:   "#e63946",
					Info:    "#118ab2",
				},
				Borders: models.BorderColors{
					Default: "rgba(26, 26, 26, 0.12)",
					Hover:   "rgba(26, 26, 26, 0.24)",
					Focus:   "rgba(230, 57, 70, 0.5)",
				},
			},
			Typography: models.TypographyProperties{
				FontFamilies: models.FontFamilies{
					Base:    "'Inter', -apple-system, BlinkMacSystemFont, sans-serif",
					Heading: "'Bebas Neue', 'Arial Black', sans-serif",
					Accent:  "'Oswald', sans-serif",
					Mono:    "'Roboto Mono', monospace",
				},
				FontSizes: models.FontSizes{
					XS:   "11px",
					SM:   "13px",
					Base: "15px",
					LG:   "18px",
					XL:   "22px",
					XL2:  "28px",
					XL3:  "36px",
					XL4:  "48px",
				},
				FontWeights: models.FontWeights{
					Light:    "300",
					Normal:   "400",
					Medium:   "500",
					Semibold: "600",
					Bold:     "800",
				},
				LineHeights: models.LineHeights{
					Tight:   "1.1",
					Normal:  "1.4",
					Relaxed: "1.6",
					Loose:   "1.8",
				},
				LetterSpacing: models.LetterSpacing{
					Tight:  "-0.03em",
					Normal: "0",
					Wide:   "0.08em",
				},
			},
			Layout: models.LayoutProperties{
				Spacing: models.Spacing{
					XS:  "4px",
					SM:  "12px",
					MD:  "20px",
					LG:  "32px",
					XL:  "48px",
					XL2: "72px",
				},
				BorderRadius: models.BorderRadius{
					SM:   "0px",
					MD:   "0px",
					LG:   "4px",
					XL:   "8px",
					XL2:  "12px",
					Full: "9999px",
				},
				Grid: models.Grid{
					ColsMin:         "250px",
					Gap:             "20px",
					CardPadding:     "0px",
					CardAspectRatio: "0.75",
				},
				MaxWidths: models.MaxWidths{
					Container: "1400px",
					Modal:     "650px",
				},
			},
			Effects: models.EffectProperties{
				Shadows: models.Shadows{
					SM: "0 1px 3px rgba(0, 0, 0, 0.08)",
					MD: "0 4px 12px rgba(0, 0, 0, 0.12)",
					LG: "0 12px 24px rgba(0, 0, 0, 0.16)",
					XL: "0 24px 48px rgba(0, 0, 0, 0.2)",
				},
				Transitions: models.Transitions{
					Fast:   "0.15s ease-out",
					Normal: "0.25s ease-out",
					Slow:   "0.4s ease-out",
				},
				Blur: models.Blur{
					SM: "3px",
					MD: "6px",
					LG: "12px",
				},
				Opacity: models.Opacity{
					Disabled: "0.4",
					Muted:    "0.7",
				},
			},
			Animations: models.AnimationProperties{
				Enabled:    true,
				HoverScale: "1.03",
				PressScale: "0.97",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

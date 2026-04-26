// Package server - Project-wide theme detection and switching.
// See AI.md PART 16 for theme specifications.
// Supports dark (default), light, and auto (system preference) themes.
package server

// ThemePalette represents a theme option
type ThemePalette string

const (
	// ThemePaletteDark is the default dark theme
	ThemePaletteDark ThemePalette = "dark"
	// ThemePaletteLight is the light theme
	ThemePaletteLight ThemePalette = "light"
	// ThemePaletteAuto follows system preference
	ThemePaletteAuto ThemePalette = "auto"
)

// ValidThemePalettes contains all valid theme options
var ValidThemePalettes = map[string]ThemePalette{
	"dark":  ThemePaletteDark,
	"light": ThemePaletteLight,
	"auto":  ThemePaletteAuto,
}

// IsValidThemePalette checks if a theme name is valid
func IsValidThemePalette(name string) bool {
	_, ok := ValidThemePalettes[name]
	return ok
}

// DefaultThemePalette returns the default theme (dark)
func DefaultThemePalette() ThemePalette {
	return ThemePaletteDark
}

// ThemeConfig holds theme-related configuration
type ThemeConfig struct {
	// Default theme for new visitors
	DefaultPalette ThemePalette
	// Allow users to change theme
	AllowUserTheme bool
}

// DefaultThemeConfig returns the default theme configuration
func DefaultThemeConfig() ThemeConfig {
	return ThemeConfig{
		DefaultPalette: ThemePaletteDark,
		AllowUserTheme: true,
	}
}

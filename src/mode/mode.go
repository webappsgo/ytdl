// Package mode manages application runtime modes (production/development).
// See AI.md PART 6 for complete mode specifications.
package mode

import (
	"os"
	"strings"
)

// AppMode represents the application runtime mode
type AppMode int

const (
	// AppModeProduction is the default mode with optimized settings
	AppModeProduction AppMode = iota
	// AppModeDevelopment enables verbose logging and debug endpoints
	AppModeDevelopment
)

var currentAppMode = AppModeProduction

// SetAppMode sets the application mode from a string value
func SetAppMode(modeStr string) {
	switch strings.ToLower(strings.TrimSpace(modeStr)) {
	case "development", "dev":
		currentAppMode = AppModeDevelopment
	default:
		currentAppMode = AppModeProduction
	}
}

// GetAppMode returns the current application mode
func GetAppMode() AppMode {
	return currentAppMode
}

// IsAppModeProd returns true if running in production mode
func IsAppModeProd() bool {
	return currentAppMode == AppModeProduction
}

// IsAppModeDev returns true if running in development mode
func IsAppModeDev() bool {
	return currentAppMode == AppModeDevelopment
}

// String returns the string representation of the mode
func (m AppMode) String() string {
	switch m {
	case AppModeDevelopment:
		return "development"
	default:
		return "production"
	}
}

// DetectAppMode determines mode from environment, falling back to the provided default.
// Priority: YTDL_MODE env var > provided default
func DetectAppMode(defaultMode string) string {
	if envMode := os.Getenv("YTDL_MODE"); envMode != "" {
		return envMode
	}
	if defaultMode != "" {
		return defaultMode
	}
	return "production"
}

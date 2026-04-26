// Package server - CORS middleware.
// See AI.md PART 14 for CORS specifications.
// Handles preflight OPTIONS requests and sets CORS headers.
package server

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	// Allowed origins ("*" for any, or comma-separated list)
	AllowedOrigins string
	// Allowed methods
	AllowedMethods string
	// Allowed headers
	AllowedHeaders string
	// Max age for preflight cache (seconds)
	MaxAge string
	// Allow credentials (cookies)
	AllowCredentials bool
}

// DefaultCORSConfig returns sensible CORS defaults
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   "*",
		AllowedMethods:   "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowedHeaders:   "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With",
		MaxAge:           "86400",
		AllowCredentials: true,
	}
}

// CORSMiddleware adds CORS headers to responses and handles preflight.
// See AI.md PART 14 for CORS requirements.
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine allowed origin
			allowedOrigin := cfg.AllowedOrigins
			if allowedOrigin != "*" && origin != "" {
				// Check if origin is in allowed list
				allowed := false
				for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
					if strings.TrimSpace(o) == origin {
						allowed = true
						break
					}
				}
				if allowed {
					allowedOrigin = origin
				} else {
					allowedOrigin = ""
				}
			}

			// When allowing credentials, cannot use "*"
			if cfg.AllowCredentials && allowedOrigin == "*" && origin != "" {
				allowedOrigin = origin
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}

			w.Header().Set("Access-Control-Allow-Methods", cfg.AllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", cfg.AllowedHeaders)
			w.Header().Set("Access-Control-Max-Age", cfg.MaxAge)

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Expose custom headers to JS
			w.Header().Set("Access-Control-Expose-Headers", "X-CSRF-Token, X-Request-Id")

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

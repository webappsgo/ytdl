// Package server - HTTP middleware for security, path validation, and headers.
// See AI.md PART 5 for middleware order and path security.
// See AI.md PART 9 for cache headers.
// See AI.md PART 11 for security headers and HSTS.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"path"
	"strings"
)

// PathSecurityMiddleware normalizes paths and blocks traversal attempts.
// This MUST be first in the middleware chain - before auth, before routing.
// See AI.md PART 5 for path security specification.
func PathSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		original := r.URL.Path

		// Check both raw path and URL-decoded for traversal
		rawPath := r.URL.RawPath
		if rawPath == "" {
			rawPath = r.URL.Path
		}

		// Block path traversal attempts (encoded and decoded)
		if strings.Contains(original, "..") ||
			strings.Contains(rawPath, "..") ||
			strings.Contains(strings.ToLower(rawPath), "%2e") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Normalize the path
		cleaned := path.Clean(original)

		// Ensure leading slash
		if !strings.HasPrefix(cleaned, "/") {
			cleaned = "/" + cleaned
		}

		// Preserve trailing slash for directory paths
		if original != "/" && strings.HasSuffix(original, "/") && !strings.HasSuffix(cleaned, "/") {
			cleaned += "/"
		}

		// Update request
		r.URL.Path = cleaned

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to all responses.
// See AI.md PART 11 for security header requirements.
// Includes HSTS when request is over TLS.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Permissions policy
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// Content Security Policy
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' https://unpkg.com; "+
				"style-src 'self' 'unsafe-inline' https://unpkg.com; "+
				"img-src 'self' data: https:; "+
				"connect-src 'self' ws: wss:; "+
				"font-src 'self'")

		// HSTS when over TLS (PART 11)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// CacheControlMiddleware sets appropriate cache headers per resource type.
// See AI.md PART 9 for caching specifications.
func CacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path

		// Static assets: cache aggressively
		if strings.HasPrefix(urlPath, "/static/") {
			w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
			next.ServeHTTP(w, r)
			return
		}

		// API responses: no cache by default
		if strings.HasPrefix(urlPath, "/api/") {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			next.ServeHTTP(w, r)
			return
		}

		// HTML pages: revalidate
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")

		next.ServeHTTP(w, r)
	})
}

// ETagMiddleware generates ETag headers for cacheable responses.
// See AI.md PART 9 for ETag specifications.
func ETagMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only for GET requests on static resources
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// Capture response to compute ETag
		rw := &etagResponseWriter{ResponseWriter: w, body: make([]byte, 0)}
		next.ServeHTTP(rw, r)

		if len(rw.body) > 0 {
			hash := sha256.Sum256(rw.body)
			etag := `"` + hex.EncodeToString(hash[:8]) + `"`

			// Check If-None-Match
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("ETag", etag)
			w.Write(rw.body)
		}
	})
}

type etagResponseWriter struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (rw *etagResponseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return len(b), nil
}

func (rw *etagResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	if code != http.StatusOK {
		rw.ResponseWriter.WriteHeader(code)
	}
}

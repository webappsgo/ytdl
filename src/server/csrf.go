// Package server - CSRF protection middleware.
// See AI.md PART 1: "CSRF tokens for all state-changing forms".
// Generates and validates CSRF tokens on POST/PUT/PATCH/DELETE requests.
package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	csrfTokenLength = 32
	csrfCookieName  = "ytdl_csrf"
	csrfHeaderName  = "X-CSRF-Token"
	csrfFormField   = "csrf_token"
	csrfMaxAge      = 12 * time.Hour
)

// CSRFStore holds active CSRF tokens
type CSRFStore struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
}

// NewCSRFStore creates a new token store
func NewCSRFStore() *CSRFStore {
	s := &CSRFStore{tokens: make(map[string]time.Time)}
	go s.cleanupLoop()
	return s
}

// GenerateToken creates a new CSRF token
func (s *CSRFStore) GenerateToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	s.mu.Lock()
	s.tokens[token] = time.Now().Add(csrfMaxAge)
	s.mu.Unlock()

	return token, nil
}

// ValidateToken checks if a token is valid
func (s *CSRFStore) ValidateToken(token string) bool {
	s.mu.RLock()
	expiry, exists := s.tokens[token]
	s.mu.RUnlock()

	if !exists || time.Now().After(expiry) {
		return false
	}
	return true
}

func (s *CSRFStore) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for token, expiry := range s.tokens {
			if now.After(expiry) {
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}

// CSRFMiddleware validates CSRF tokens on state-changing requests.
// Skips API requests with Bearer token (already authenticated).
// Sets CSRF token cookie on every response for forms to use.
func CSRFMiddleware(store *CSRFStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip safe methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				// Set CSRF token cookie for forms
				setCSRFCookie(w, store)
				next.ServeHTTP(w, r)
				return
			}

			// Skip if Bearer token present (API auth, not form)
			if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
				next.ServeHTTP(w, r)
				return
			}

			// Skip JSON API requests (they use Bearer tokens or are stateless)
			contentType := r.Header.Get("Content-Type")
			if contentType == "application/json" {
				next.ServeHTTP(w, r)
				return
			}

			// Validate CSRF token for form submissions
			token := r.Header.Get(csrfHeaderName)
			if token == "" {
				token = r.FormValue(csrfFormField)
			}
			if token == "" {
				// Check cookie
				if cookie, err := r.Cookie(csrfCookieName); err == nil {
					token = cookie.Value
				}
			}

			if token == "" || !store.ValidateToken(token) {
				http.Error(w, "CSRF token missing or invalid", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func setCSRFCookie(w http.ResponseWriter, store *CSRFStore) {
	token, err := store.GenerateToken()
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(csrfMaxAge.Seconds()),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	// Also set as header for JS to read
	w.Header().Set("X-CSRF-Token", token)
}

// ConstantTimeCompare does constant-time string comparison
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

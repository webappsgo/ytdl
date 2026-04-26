// Package handler - robots.txt, security.txt, and well-known endpoints.
// See AI.md for server endpoint requirements.
package handler

import (
	"fmt"
	"net/http"
	"time"
)

// HandleRobotsTxt serves robots.txt
func HandleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprint(w, `User-agent: *
Allow: /
Disallow: /admin
Disallow: /api/
Disallow: /auth/
Disallow: /ws
`)
}

// HandleSecurityTxt serves .well-known/security.txt
func HandleSecurityTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	expires := time.Now().AddDate(1, 0, 0).Format("2006-01-02T15:04:05Z")
	fmt.Fprintf(w, `Contact: https://github.com/casapps/ytdl/issues
Expires: %s
Preferred-Languages: en, es
`, expires)
}

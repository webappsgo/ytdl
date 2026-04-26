// Package handler - Sharing, RSS/podcast feed, webhooks, and integration endpoints.
// See IDEA.md for feature specifications.
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/casapps/ytdl/src/server/store"
	"github.com/go-chi/chi/v5"
)

// SharingHandler handles sharing, RSS, and webhook features
type SharingHandler struct {
	store       *store.Store
	version     string
	officialSite string
}

// NewSharingHandler creates a new sharing handler
func NewSharingHandler(st *store.Store, version, officialSite string) *SharingHandler {
	return &SharingHandler{store: st, version: version, officialSite: officialSite}
}

// HandleCreateShareLink handles POST /api/v1/downloads/{id}/share
// Generates a shareable direct download link with optional expiration
func (h *SharingHandler) HandleCreateShareLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	var req struct {
		ExpiresInHours int `json:"expires_in_hours"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Generate share token
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	shareToken := hex.EncodeToString(tokenBytes)

	// Store share link in database
	var expiresAt *time.Time
	if req.ExpiresInHours > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &exp
	}

	_, err = h.store.DB().Exec(
		`INSERT INTO share_links (download_id, token, expires_at) VALUES (?, ?, ?)`,
		id, shareToken, expiresAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to create share link", Code: "CREATE_FAILED"})
		return
	}

	shareURL := fmt.Sprintf("/dl/%s", shareToken)
	if h.officialSite != "" {
		shareURL = fmt.Sprintf("%s/dl/%s", h.officialSite, shareToken)
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Data: map[string]interface{}{
			"share_url":  shareURL,
			"token":      shareToken,
			"expires_at": expiresAt,
		},
	})
}

// HandleShareDownload handles GET /dl/{token}
// Serves the file for a shared download link
func (h *SharingHandler) HandleShareDownload(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	var downloadID int64
	var expiresAt *time.Time

	err := h.store.DB().QueryRow(
		`SELECT download_id, expires_at FROM share_links WHERE token = ?`, token,
	).Scan(&downloadID, &expiresAt)

	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	// Check expiration
	if expiresAt != nil && time.Now().After(*expiresAt) {
		http.Error(w, "Link has expired", http.StatusGone)
		return
	}

	// Get download and serve file
	download, err := h.store.GetDownloadByID(downloadID)
	if err != nil || download == nil || download.FilePath == "" {
		http.Error(w, "File not available", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, download.FilePath)
}

// HandleRSSFeed handles GET /api/v1/feed/rss
// Generates RSS/podcast feed from audio downloads
func (h *SharingHandler) HandleRSSFeed(w http.ResponseWriter, r *http.Request) {
	downloads, _, err := h.store.ListDownloads("completed", 50, 0)
	if err != nil {
		http.Error(w, "Failed to generate feed", http.StatusInternalServerError)
		return
	}

	type RSSEnclosure struct {
		XMLName xml.Name `xml:"enclosure"`
		URL     string   `xml:"url,attr"`
		Length  int64    `xml:"length,attr"`
		Type    string   `xml:"type,attr"`
	}

	type RSSItem struct {
		XMLName     xml.Name       `xml:"item"`
		Title       string         `xml:"title"`
		Link        string         `xml:"link"`
		Description string         `xml:"description"`
		PubDate     string         `xml:"pubDate"`
		Enclosure   *RSSEnclosure  `xml:"enclosure,omitempty"`
	}

	type RSSChannel struct {
		XMLName     xml.Name  `xml:"channel"`
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []RSSItem `xml:"item"`
	}

	type RSS struct {
		XMLName xml.Name   `xml:"rss"`
		Version string     `xml:"version,attr"`
		Channel RSSChannel `xml:"channel"`
	}

	baseURL := h.officialSite
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s", r.Host)
	}

	var items []RSSItem
	for _, d := range downloads {
		// Only include audio files in podcast feed
		isAudio := d.Format == "mp3" || d.Format == "m4a" || d.Format == "opus" || d.Format == "flac"
		if !isAudio {
			continue
		}

		mimeType := "audio/mpeg"
		switch d.Format {
		case "m4a":
			mimeType = "audio/mp4"
		case "opus":
			mimeType = "audio/ogg"
		case "flac":
			mimeType = "audio/flac"
		}

		item := RSSItem{
			Title:       d.Title,
			Link:        fmt.Sprintf("%s/api/v1/downloads/%d/file", baseURL, d.ID),
			Description: d.Description,
		}
		if d.CompletedAt != nil {
			item.PubDate = d.CompletedAt.Format(time.RFC1123Z)
		}
		item.Enclosure = &RSSEnclosure{
			URL:    fmt.Sprintf("%s/api/v1/downloads/%d/file", baseURL, d.ID),
			Length: d.FileSize,
			Type:   mimeType,
		}
		items = append(items, item)
	}

	feed := RSS{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       "ytdl Downloads",
			Link:        baseURL,
			Description: "Media downloads from ytdl",
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(feed)
}

// HandleBrowserExtensionSubmit handles POST /api/v1/ext/download
// API endpoint for browser extensions to submit URLs
func (h *SharingHandler) HandleBrowserExtensionSubmit(w http.ResponseWriter, r *http.Request) {
	// Same as regular submit but accepts simplified format
	var req struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "URL is required", Code: "MISSING_URL"})
		return
	}
	if req.Format == "" {
		req.Format = "mp4"
	}

	d := &store.Download{
		URL:        req.URL,
		Format:     req.Format,
		Quality:    "1080",
		Bitrate:    "320k",
		Priority:   store.PriorityNormal,
		MaxRetries: 3,
	}

	id, err := h.store.CreateDownload(d)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to submit", Code: "SUBMIT_FAILED"})
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Data: map[string]interface{}{
			"id":     id,
			"status": "queued",
		},
	})
}

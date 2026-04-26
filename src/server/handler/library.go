// Package handler - Media library, collections, presets, and metadata API endpoints.
// See IDEA.md for feature specifications.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/casapps/ytdl/src/server/store"
	"github.com/go-chi/chi/v5"
)

// LibraryHandler handles media library, collections, presets, and metadata
type LibraryHandler struct {
	store *store.Store
}

// NewLibraryHandler creates a new library handler
func NewLibraryHandler(st *store.Store) *LibraryHandler {
	return &LibraryHandler{store: st}
}

// HandleBrowseLibrary handles GET /api/v1/library
// Browse downloaded media with search, filter, sort
func (h *LibraryHandler) HandleBrowseLibrary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	format := r.URL.Query().Get("format")
	site := r.URL.Query().Get("site")
	sortBy := r.URL.Query().Get("sort")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	// Build query
	sqlQuery := `SELECT id, url, title, source_site, channel_name, thumbnail_url,
	                    duration, format, quality, file_size, file_path, created_at, completed_at
	             FROM downloads WHERE status = 'completed'`
	var args []interface{}

	if query != "" {
		sqlQuery += ` AND (title LIKE ? OR channel_name LIKE ?)`
		searchTerm := "%" + query + "%"
		args = append(args, searchTerm, searchTerm)
	}
	if format != "" {
		sqlQuery += ` AND format = ?`
		args = append(args, format)
	}
	if site != "" {
		sqlQuery += ` AND source_site = ?`
		args = append(args, site)
	}

	// Sort
	switch sortBy {
	case "title":
		sqlQuery += ` ORDER BY title ASC`
	case "size":
		sqlQuery += ` ORDER BY file_size DESC`
	case "oldest":
		sqlQuery += ` ORDER BY completed_at ASC`
	default:
		sqlQuery += ` ORDER BY completed_at DESC`
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM downloads WHERE status = 'completed'`
	var total int
	h.store.DB().QueryRow(countQuery).Scan(&total)

	sqlQuery += ` LIMIT ? OFFSET ?`
	args = append(args, perPage, offset)

	rows, err := h.store.DB().Query(sqlQuery, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Query failed", Code: "QUERY_FAILED"})
		return
	}
	defer rows.Close()

	type LibraryItem struct {
		ID           int64   `json:"id"`
		URL          string  `json:"url"`
		Title        string  `json:"title"`
		SourceSite   string  `json:"source_site"`
		ChannelName  string  `json:"channel_name"`
		ThumbnailURL string  `json:"thumbnail_url"`
		Duration     int     `json:"duration"`
		Format       string  `json:"format"`
		Quality      string  `json:"quality"`
		FileSize     int64   `json:"file_size"`
		FilePath     string  `json:"file_path"`
		CreatedAt    string  `json:"created_at"`
		CompletedAt  *string `json:"completed_at"`
	}

	var items []LibraryItem
	for rows.Next() {
		var item LibraryItem
		if err := rows.Scan(&item.ID, &item.URL, &item.Title, &item.SourceSite,
			&item.ChannelName, &item.ThumbnailURL, &item.Duration, &item.Format,
			&item.Quality, &item.FileSize, &item.FilePath, &item.CreatedAt, &item.CompletedAt); err != nil {
			continue
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: items, Total: total, Page: page, PerPage: perPage})
}

// HandleListCollections handles GET /api/v1/collections
func (h *LibraryHandler) HandleListCollections(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.DB().Query(
		`SELECT id, name, description, type, rules_json, cover_image_path, created_at
		 FROM collections ORDER BY name ASC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Query failed", Code: "QUERY_FAILED"})
		return
	}
	defer rows.Close()

	type Collection struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		Type           string `json:"type"`
		RulesJSON      string `json:"rules_json,omitempty"`
		CoverImagePath string `json:"cover_image_path,omitempty"`
		CreatedAt      string `json:"created_at"`
	}

	var collections []Collection
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Type, &c.RulesJSON, &c.CoverImagePath, &c.CreatedAt); err != nil {
			continue
		}
		collections = append(collections, c)
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: collections, Total: len(collections)})
}

// HandleCreateCollection handles POST /api/v1/collections
func (h *LibraryHandler) HandleCreateCollection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
		RulesJSON   string `json:"rules_json"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Name is required", Code: "MISSING_NAME"})
		return
	}
	if req.Type == "" {
		req.Type = "manual"
	}

	result, err := h.store.DB().Exec(
		`INSERT INTO collections (name, description, type, rules_json) VALUES (?, ?, ?, ?)`,
		req.Name, req.Description, req.Type, req.RulesJSON,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to create collection", Code: "CREATE_FAILED"})
		return
	}

	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, APIResponse{Data: map[string]int64{"id": id}})
}

// HandleDeleteCollection handles DELETE /api/v1/collections/{id}
func (h *LibraryHandler) HandleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	h.store.DB().Exec(`DELETE FROM collections WHERE id = ?`, id)
	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "deleted"}})
}

// HandleAddToCollection handles POST /api/v1/collections/{id}/items
func (h *LibraryHandler) HandleAddToCollection(w http.ResponseWriter, r *http.Request) {
	collectionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	var req struct {
		DownloadID int64 `json:"download_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	_, err = h.store.DB().Exec(
		`INSERT OR IGNORE INTO collection_items (collection_id, download_id) VALUES (?, ?)`,
		collectionID, req.DownloadID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to add item", Code: "ADD_FAILED"})
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse{Data: map[string]string{"status": "added"}})
}

// HandleListPresets handles GET /api/v1/presets
func (h *LibraryHandler) HandleListPresets(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.DB().Query(
		`SELECT id, name, format, quality, bitrate, audio_only, subtitle_languages,
		        embed_subtitles, embed_lyrics, normalize_audio, trim_silence,
		        output_template, is_default, created_at
		 FROM download_presets ORDER BY name ASC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Query failed", Code: "QUERY_FAILED"})
		return
	}
	defer rows.Close()

	type Preset struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		Format            string `json:"format"`
		Quality           string `json:"quality"`
		Bitrate           string `json:"bitrate"`
		AudioOnly         bool   `json:"audio_only"`
		SubtitleLanguages string `json:"subtitle_languages"`
		EmbedSubtitles    bool   `json:"embed_subtitles"`
		EmbedLyrics       bool   `json:"embed_lyrics"`
		NormalizeAudio    bool   `json:"normalize_audio"`
		TrimSilence       bool   `json:"trim_silence"`
		OutputTemplate    string `json:"output_template"`
		IsDefault         bool   `json:"is_default"`
		CreatedAt         string `json:"created_at"`
	}

	var presets []Preset
	for rows.Next() {
		var p Preset
		if err := rows.Scan(&p.ID, &p.Name, &p.Format, &p.Quality, &p.Bitrate,
			&p.AudioOnly, &p.SubtitleLanguages, &p.EmbedSubtitles, &p.EmbedLyrics,
			&p.NormalizeAudio, &p.TrimSilence, &p.OutputTemplate, &p.IsDefault, &p.CreatedAt); err != nil {
			continue
		}
		presets = append(presets, p)
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: presets, Total: len(presets)})
}

// HandleCreatePreset handles POST /api/v1/presets
func (h *LibraryHandler) HandleCreatePreset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string `json:"name"`
		Format            string `json:"format"`
		Quality           string `json:"quality"`
		Bitrate           string `json:"bitrate"`
		AudioOnly         bool   `json:"audio_only"`
		SubtitleLanguages string `json:"subtitle_languages"`
		EmbedSubtitles    bool   `json:"embed_subtitles"`
		EmbedLyrics       bool   `json:"embed_lyrics"`
		NormalizeAudio    bool   `json:"normalize_audio"`
		TrimSilence       bool   `json:"trim_silence"`
		OutputTemplate    string `json:"output_template"`
		IsDefault         bool   `json:"is_default"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Name is required", Code: "MISSING_NAME"})
		return
	}

	// If setting as default, unset other defaults
	if req.IsDefault {
		h.store.DB().Exec(`UPDATE download_presets SET is_default = 0`)
	}

	result, err := h.store.DB().Exec(
		`INSERT INTO download_presets (name, format, quality, bitrate, audio_only,
		 subtitle_languages, embed_subtitles, embed_lyrics, normalize_audio,
		 trim_silence, output_template, is_default) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Format, req.Quality, req.Bitrate, req.AudioOnly,
		req.SubtitleLanguages, req.EmbedSubtitles, req.EmbedLyrics, req.NormalizeAudio,
		req.TrimSilence, req.OutputTemplate, req.IsDefault,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to create preset", Code: "CREATE_FAILED"})
		return
	}

	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, APIResponse{Data: map[string]int64{"id": id}})
}

// HandleDeletePreset handles DELETE /api/v1/presets/{id}
func (h *LibraryHandler) HandleDeletePreset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	h.store.DB().Exec(`DELETE FROM download_presets WHERE id = ?`, id)
	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "deleted"}})
}

// HandleGetMetadata handles GET /api/v1/downloads/{id}/metadata
func (h *LibraryHandler) HandleGetMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	type Metadata struct {
		ID             int64  `json:"id"`
		DownloadID     int64  `json:"download_id"`
		Title          string `json:"title"`
		Artist         string `json:"artist"`
		Album          string `json:"album"`
		Year           string `json:"year"`
		Genre          string `json:"genre"`
		TrackNumber    string `json:"track_number"`
		CoverArtPath   string `json:"cover_art_path"`
		LyricsSynced   string `json:"lyrics_synced"`
		LyricsUnsynced string `json:"lyrics_unsynced"`
		LyricsLanguage string `json:"lyrics_language"`
	}

	var m Metadata
	err = h.store.DB().QueryRow(
		`SELECT id, download_id, title, artist, album, year, genre, track_number,
		        cover_art_path, lyrics_synced, lyrics_unsynced, lyrics_language
		 FROM media_metadata WHERE download_id = ?`, id,
	).Scan(&m.ID, &m.DownloadID, &m.Title, &m.Artist, &m.Album, &m.Year,
		&m.Genre, &m.TrackNumber, &m.CoverArtPath, &m.LyricsSynced,
		&m.LyricsUnsynced, &m.LyricsLanguage)

	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Error: "Metadata not found", Code: "NOT_FOUND"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: m})
}

// HandleUpdateMetadata handles PUT /api/v1/downloads/{id}/metadata
func (h *LibraryHandler) HandleUpdateMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	var req struct {
		Title          string `json:"title"`
		Artist         string `json:"artist"`
		Album          string `json:"album"`
		Year           string `json:"year"`
		Genre          string `json:"genre"`
		TrackNumber    string `json:"track_number"`
		LyricsSynced   string `json:"lyrics_synced"`
		LyricsUnsynced string `json:"lyrics_unsynced"`
		LyricsLanguage string `json:"lyrics_language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	// Upsert metadata
	_, err = h.store.DB().Exec(
		`INSERT INTO media_metadata (download_id, title, artist, album, year, genre, track_number,
		 lyrics_synced, lyrics_unsynced, lyrics_language)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(download_id) DO UPDATE SET
		 title=excluded.title, artist=excluded.artist, album=excluded.album,
		 year=excluded.year, genre=excluded.genre, track_number=excluded.track_number,
		 lyrics_synced=excluded.lyrics_synced, lyrics_unsynced=excluded.lyrics_unsynced,
		 lyrics_language=excluded.lyrics_language`,
		id, req.Title, req.Artist, req.Album, req.Year, req.Genre, req.TrackNumber,
		req.LyricsSynced, req.LyricsUnsynced, req.LyricsLanguage,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to update metadata", Code: "UPDATE_FAILED"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "updated"}})
}

// HandleListWatchRules handles GET /api/v1/watch-rules
func (h *LibraryHandler) HandleListWatchRules(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.DB().Query(
		`SELECT id, name, url, check_interval, last_checked_at, action, preset_id, enabled, created_at
		 FROM watch_rules ORDER BY name ASC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Query failed", Code: "QUERY_FAILED"})
		return
	}
	defer rows.Close()

	type WatchRule struct {
		ID            int64   `json:"id"`
		Name          string  `json:"name"`
		URL           string  `json:"url"`
		CheckInterval string  `json:"check_interval"`
		LastCheckedAt *string `json:"last_checked_at"`
		Action        string  `json:"action"`
		PresetID      *int64  `json:"preset_id"`
		Enabled       bool    `json:"enabled"`
		CreatedAt     string  `json:"created_at"`
	}

	var rules []WatchRule
	for rows.Next() {
		var r WatchRule
		if err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.CheckInterval, &r.LastCheckedAt,
			&r.Action, &r.PresetID, &r.Enabled, &r.CreatedAt); err != nil {
			continue
		}
		rules = append(rules, r)
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: rules, Total: len(rules)})
}

// HandleCreateWatchRule handles POST /api/v1/watch-rules
func (h *LibraryHandler) HandleCreateWatchRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		CheckInterval string `json:"check_interval"`
		Action        string `json:"action"`
		PresetID      *int64 `json:"preset_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "URL is required", Code: "MISSING_URL"})
		return
	}
	if req.Name == "" {
		req.Name = req.URL
	}
	if req.CheckInterval == "" {
		req.CheckInterval = "6h"
	}
	if req.Action == "" {
		req.Action = "auto_download"
	}

	result, err := h.store.DB().Exec(
		`INSERT INTO watch_rules (name, url, check_interval, action, preset_id) VALUES (?, ?, ?, ?, ?)`,
		req.Name, req.URL, req.CheckInterval, req.Action, req.PresetID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to create watch rule", Code: "CREATE_FAILED"})
		return
	}

	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, APIResponse{Data: map[string]int64{"id": id}})
}

// HandleDeleteWatchRule handles DELETE /api/v1/watch-rules/{id}
func (h *LibraryHandler) HandleDeleteWatchRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid ID", Code: "INVALID_ID"})
		return
	}

	h.store.DB().Exec(`DELETE FROM watch_rules WHERE id = ?`, id)
	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "deleted"}})
}

// HandleGetAnalytics handles GET /api/v1/analytics
func (h *LibraryHandler) HandleGetAnalytics(w http.ResponseWriter, r *http.Request) {
	type AnalyticsData struct {
		TotalDownloads    int            `json:"total_downloads"`
		TotalSize         int64          `json:"total_size_bytes"`
		CompletedCount    int            `json:"completed_count"`
		FailedCount       int            `json:"failed_count"`
		SiteBreakdown     map[string]int `json:"site_breakdown"`
		FormatBreakdown   map[string]int `json:"format_breakdown"`
	}

	data := AnalyticsData{
		SiteBreakdown:   make(map[string]int),
		FormatBreakdown: make(map[string]int),
	}

	h.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads`).Scan(&data.TotalDownloads)
	h.store.DB().QueryRow(`SELECT COALESCE(SUM(file_size), 0) FROM downloads WHERE status = 'completed'`).Scan(&data.TotalSize)
	h.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads WHERE status = 'completed'`).Scan(&data.CompletedCount)
	h.store.DB().QueryRow(`SELECT COUNT(*) FROM downloads WHERE status = 'failed'`).Scan(&data.FailedCount)

	// Site breakdown
	rows, err := h.store.DB().Query(
		`SELECT source_site, COUNT(*) as cnt FROM downloads WHERE source_site != ''
		 GROUP BY source_site ORDER BY cnt DESC LIMIT 20`,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var site string
			var count int
			if rows.Scan(&site, &count) == nil {
				data.SiteBreakdown[site] = count
			}
		}
	}

	// Format breakdown
	rows2, err := h.store.DB().Query(
		`SELECT format, COUNT(*) as cnt FROM downloads
		 GROUP BY format ORDER BY cnt DESC`,
	)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var format string
			var count int
			if rows2.Scan(&format, &count) == nil {
				data.FormatBreakdown[format] = count
			}
		}
	}

	writeJSON(w, http.StatusOK, APIResponse{Data: data})
}

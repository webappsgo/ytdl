// Package store implements data access layer with SQLite.
// See AI.md PART 10 for database specifications.
// Uses modernc.org/sqlite (pure Go, CGO_ENABLED=0 compatible).
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store provides data access operations
type Store struct {
	db *sql.DB
}

// NewStore creates a new store with SQLite database
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Connection pool settings for SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}

	// Create tables on startup (idempotent)
	if err := store.ensureSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensuring schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection for advanced queries
func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) ensureSchema() error {
	for _, stmt := range createStatements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	for _, stmt := range schemaUpdates {
		if _, err := s.db.Exec(stmt); err != nil {
			if !isColumnExistsError(err) {
				return fmt.Errorf("schema update: %w", err)
			}
		}
	}

	return nil
}

var createStatements = []string{
	`CREATE TABLE IF NOT EXISTS downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		source_site TEXT NOT NULL DEFAULT '',
		channel_name TEXT NOT NULL DEFAULT '',
		channel_url TEXT NOT NULL DEFAULT '',
		thumbnail_url TEXT NOT NULL DEFAULT '',
		duration INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'queued',
		format TEXT NOT NULL DEFAULT 'mp4',
		quality TEXT NOT NULL DEFAULT '1080',
		bitrate TEXT NOT NULL DEFAULT '320k',
		priority TEXT NOT NULL DEFAULT 'normal',
		progress_percent REAL NOT NULL DEFAULT 0,
		file_size INTEGER NOT NULL DEFAULT 0,
		file_path TEXT NOT NULL DEFAULT '',
		thumbnail_path TEXT NOT NULL DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		retry_count INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		proxy_config TEXT NOT NULL DEFAULT '',
		preset_id INTEGER,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		completed_at TIMESTAMP,
		expires_at TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS playlist_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_id INTEGER NOT NULL,
		item_index INTEGER NOT NULL DEFAULT 0,
		url TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'queued',
		file_path TEXT NOT NULL DEFAULT '',
		thumbnail_path TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (download_id) REFERENCES downloads(id) ON DELETE CASCADE
	)`,

	`CREATE TABLE IF NOT EXISTS download_presets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		format TEXT NOT NULL DEFAULT 'mp4',
		quality TEXT NOT NULL DEFAULT '1080',
		bitrate TEXT NOT NULL DEFAULT '320k',
		audio_only INTEGER NOT NULL DEFAULT 0,
		subtitle_languages TEXT NOT NULL DEFAULT 'en,es',
		embed_subtitles INTEGER NOT NULL DEFAULT 1,
		embed_lyrics INTEGER NOT NULL DEFAULT 1,
		normalize_audio INTEGER NOT NULL DEFAULT 0,
		trim_silence INTEGER NOT NULL DEFAULT 0,
		output_template TEXT NOT NULL DEFAULT '',
		is_default INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS watch_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		check_interval TEXT NOT NULL DEFAULT '6h',
		last_checked_at TIMESTAMP,
		action TEXT NOT NULL DEFAULT 'auto_download',
		preset_id INTEGER,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (preset_id) REFERENCES download_presets(id)
	)`,

	`CREATE TABLE IF NOT EXISTS collections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL DEFAULT 'manual',
		rules_json TEXT NOT NULL DEFAULT '{}',
		cover_image_path TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS collection_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		collection_id INTEGER NOT NULL,
		download_id INTEGER NOT NULL,
		sort_order INTEGER NOT NULL DEFAULT 0,
		added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
		FOREIGN KEY (download_id) REFERENCES downloads(id) ON DELETE CASCADE,
		UNIQUE(collection_id, download_id)
	)`,

	`CREATE TABLE IF NOT EXISTS media_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_id INTEGER NOT NULL UNIQUE,
		title TEXT NOT NULL DEFAULT '',
		artist TEXT NOT NULL DEFAULT '',
		album TEXT NOT NULL DEFAULT '',
		year TEXT NOT NULL DEFAULT '',
		genre TEXT NOT NULL DEFAULT '',
		track_number TEXT NOT NULL DEFAULT '',
		cover_art_path TEXT NOT NULL DEFAULT '',
		lyrics_synced TEXT NOT NULL DEFAULT '',
		lyrics_unsynced TEXT NOT NULL DEFAULT '',
		lyrics_language TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (download_id) REFERENCES downloads(id) ON DELETE CASCADE
	)`,

	`CREATE TABLE IF NOT EXISTS schedule_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		start_time TEXT NOT NULL DEFAULT '00:00',
		end_time TEXT NOT NULL DEFAULT '23:59',
		days_of_week TEXT NOT NULL DEFAULT '0,1,2,3,4,5,6',
		speed_limit INTEGER NOT NULL DEFAULT 0,
		pause_downloads INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS analytics_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		download_id INTEGER,
		site TEXT NOT NULL DEFAULT '',
		format TEXT NOT NULL DEFAULT '',
		file_size INTEGER NOT NULL DEFAULT 0,
		duration_seconds INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS share_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (download_id) REFERENCES downloads(id) ON DELETE CASCADE
	)`,

	`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		admin_id INTEGER NOT NULL DEFAULT 0,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	// Indexes for performance
	`CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status)`,
	`CREATE INDEX IF NOT EXISTS idx_downloads_created ON downloads(created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_downloads_expires ON downloads(expires_at)`,
	`CREATE INDEX IF NOT EXISTS idx_playlist_items_download ON playlist_items(download_id)`,
	`CREATE INDEX IF NOT EXISTS idx_collection_items_collection ON collection_items(collection_id)`,
	`CREATE INDEX IF NOT EXISTS idx_analytics_events_type ON analytics_events(event_type)`,
	`CREATE INDEX IF NOT EXISTS idx_analytics_events_created ON analytics_events(created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_share_links_token ON share_links(token)`,
}

// Schema updates for future versions (idempotent)
var schemaUpdates = []string{}

func isColumnExistsError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "Duplicate column name")
}

// Download status constants
const (
	StatusQueued      = "queued"
	StatusDownloading = "downloading"
	StatusProcessing  = "processing"
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusCancelled   = "cancelled"
	StatusPaused      = "paused"
)

// Priority constants
const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// Download represents a download record
type Download struct {
	ID              int64      `json:"id"`
	URL             string     `json:"url"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	SourceSite      string     `json:"source_site"`
	ChannelName     string     `json:"channel_name"`
	ChannelURL      string     `json:"channel_url"`
	ThumbnailURL    string     `json:"thumbnail_url"`
	Duration        int        `json:"duration"`
	Status          string     `json:"status"`
	Format          string     `json:"format"`
	Quality         string     `json:"quality"`
	Bitrate         string     `json:"bitrate"`
	Priority        string     `json:"priority"`
	ProgressPercent float64    `json:"progress_percent"`
	FileSize        int64      `json:"file_size"`
	FilePath        string     `json:"file_path"`
	ThumbnailPath   string     `json:"thumbnail_path"`
	ErrorMessage    string     `json:"error_message"`
	RetryCount      int        `json:"retry_count"`
	MaxRetries      int        `json:"max_retries"`
	ProxyConfig     string     `json:"proxy_config,omitempty"`
	PresetID        *int64     `json:"preset_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

// CreateDownload inserts a new download record
func (s *Store) CreateDownload(d *Download) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO downloads (url, title, format, quality, bitrate, priority, preset_id, proxy_config, max_retries)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.URL, d.Title, d.Format, d.Quality, d.Bitrate, d.Priority, d.PresetID, d.ProxyConfig, d.MaxRetries,
	)
	if err != nil {
		return 0, fmt.Errorf("creating download: %w", err)
	}
	return result.LastInsertId()
}

// GetDownloadByID retrieves a download by ID
func (s *Store) GetDownloadByID(id int64) (*Download, error) {
	d := &Download{}
	err := s.db.QueryRow(
		`SELECT id, url, title, description, source_site, channel_name, channel_url,
		        thumbnail_url, duration, status, format, quality, bitrate, priority,
		        progress_percent, file_size, file_path, thumbnail_path, error_message,
		        retry_count, max_retries, proxy_config, preset_id,
		        created_at, started_at, completed_at, expires_at
		 FROM downloads WHERE id = ?`, id,
	).Scan(
		&d.ID, &d.URL, &d.Title, &d.Description, &d.SourceSite, &d.ChannelName, &d.ChannelURL,
		&d.ThumbnailURL, &d.Duration, &d.Status, &d.Format, &d.Quality, &d.Bitrate, &d.Priority,
		&d.ProgressPercent, &d.FileSize, &d.FilePath, &d.ThumbnailPath, &d.ErrorMessage,
		&d.RetryCount, &d.MaxRetries, &d.ProxyConfig, &d.PresetID,
		&d.CreatedAt, &d.StartedAt, &d.CompletedAt, &d.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting download: %w", err)
	}
	return d, nil
}

// ListDownloads retrieves downloads with optional status filter and pagination
func (s *Store) ListDownloads(status string, limit, offset int) ([]*Download, int, error) {
	// Count total
	var countQuery string
	var countArgs []interface{}
	if status != "" {
		countQuery = `SELECT COUNT(*) FROM downloads WHERE status = ?`
		countArgs = append(countArgs, status)
	} else {
		countQuery = `SELECT COUNT(*) FROM downloads`
	}

	var total int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting downloads: %w", err)
	}

	// Query records
	var query string
	var args []interface{}
	if status != "" {
		query = `SELECT id, url, title, description, source_site, channel_name, channel_url,
		                thumbnail_url, duration, status, format, quality, bitrate, priority,
		                progress_percent, file_size, file_path, thumbnail_path, error_message,
		                retry_count, max_retries, proxy_config, preset_id,
		                created_at, started_at, completed_at, expires_at
		         FROM downloads WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = append(args, status, limit, offset)
	} else {
		query = `SELECT id, url, title, description, source_site, channel_name, channel_url,
		                thumbnail_url, duration, status, format, quality, bitrate, priority,
		                progress_percent, file_size, file_path, thumbnail_path, error_message,
		                retry_count, max_retries, proxy_config, preset_id,
		                created_at, started_at, completed_at, expires_at
		         FROM downloads ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing downloads: %w", err)
	}
	defer rows.Close()

	var downloads []*Download
	for rows.Next() {
		d := &Download{}
		if err := rows.Scan(
			&d.ID, &d.URL, &d.Title, &d.Description, &d.SourceSite, &d.ChannelName, &d.ChannelURL,
			&d.ThumbnailURL, &d.Duration, &d.Status, &d.Format, &d.Quality, &d.Bitrate, &d.Priority,
			&d.ProgressPercent, &d.FileSize, &d.FilePath, &d.ThumbnailPath, &d.ErrorMessage,
			&d.RetryCount, &d.MaxRetries, &d.ProxyConfig, &d.PresetID,
			&d.CreatedAt, &d.StartedAt, &d.CompletedAt, &d.ExpiresAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning download: %w", err)
		}
		downloads = append(downloads, d)
	}

	return downloads, total, nil
}

// UpdateDownloadStatus updates the status of a download
func (s *Store) UpdateDownloadStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE downloads SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("updating download status: %w", err)
	}
	return nil
}

// UpdateDownloadProgress updates the progress of a download
func (s *Store) UpdateDownloadProgress(id int64, progress float64, fileSize int64) error {
	_, err := s.db.Exec(
		`UPDATE downloads SET progress_percent = ?, file_size = ? WHERE id = ?`,
		progress, fileSize, id,
	)
	if err != nil {
		return fmt.Errorf("updating download progress: %w", err)
	}
	return nil
}

// UpdateDownloadMetadata updates metadata fields after yt-dlp extraction
func (s *Store) UpdateDownloadMetadata(id int64, title, description, sourceSite, channelName, channelURL, thumbnailURL string, duration int) error {
	_, err := s.db.Exec(
		`UPDATE downloads SET title = ?, description = ?, source_site = ?, channel_name = ?,
		 channel_url = ?, thumbnail_url = ?, duration = ? WHERE id = ?`,
		title, description, sourceSite, channelName, channelURL, thumbnailURL, duration, id,
	)
	if err != nil {
		return fmt.Errorf("updating download metadata: %w", err)
	}
	return nil
}

// CompleteDownload marks a download as completed
func (s *Store) CompleteDownload(id int64, filePath string, fileSize int64, expiresAt *time.Time) error {
	now := time.Now()
	_, err := s.db.Exec(
		`UPDATE downloads SET status = ?, file_path = ?, file_size = ?, progress_percent = 100,
		 completed_at = ?, expires_at = ? WHERE id = ?`,
		StatusCompleted, filePath, fileSize, now, expiresAt, id,
	)
	if err != nil {
		return fmt.Errorf("completing download: %w", err)
	}
	return nil
}

// FailDownload marks a download as failed
func (s *Store) FailDownload(id int64, errorMessage string) error {
	_, err := s.db.Exec(
		`UPDATE downloads SET status = ?, error_message = ?, retry_count = retry_count + 1 WHERE id = ?`,
		StatusFailed, errorMessage, id,
	)
	if err != nil {
		return fmt.Errorf("failing download: %w", err)
	}
	return nil
}

// DeleteDownload removes a download record
func (s *Store) DeleteDownload(id int64) error {
	_, err := s.db.Exec(`DELETE FROM downloads WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting download: %w", err)
	}
	return nil
}

// GetNextQueuedDownload gets the next download to process (priority order)
func (s *Store) GetNextQueuedDownload() (*Download, error) {
	d := &Download{}
	err := s.db.QueryRow(
		`SELECT id, url, title, format, quality, bitrate, priority, proxy_config, preset_id, max_retries
		 FROM downloads WHERE status = ?
		 ORDER BY
		   CASE priority WHEN 'high' THEN 0 WHEN 'normal' THEN 1 WHEN 'low' THEN 2 END,
		   created_at ASC
		 LIMIT 1`, StatusQueued,
	).Scan(&d.ID, &d.URL, &d.Title, &d.Format, &d.Quality, &d.Bitrate, &d.Priority, &d.ProxyConfig, &d.PresetID, &d.MaxRetries)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting next queued download: %w", err)
	}
	return d, nil
}

// GetExpiredDownloads returns downloads past their expiration time
func (s *Store) GetExpiredDownloads() ([]*Download, error) {
	rows, err := s.db.Query(
		`SELECT id, file_path FROM downloads
		 WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?`,
		time.Now(), StatusCompleted,
	)
	if err != nil {
		return nil, fmt.Errorf("getting expired downloads: %w", err)
	}
	defer rows.Close()

	var downloads []*Download
	for rows.Next() {
		d := &Download{}
		if err := rows.Scan(&d.ID, &d.FilePath); err != nil {
			return nil, fmt.Errorf("scanning expired download: %w", err)
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}

// GetRetryableDownloads returns failed downloads that can be retried
func (s *Store) GetRetryableDownloads() ([]*Download, error) {
	rows, err := s.db.Query(
		`SELECT id, url, format, quality, bitrate, priority, proxy_config, retry_count, max_retries
		 FROM downloads WHERE status = ? AND retry_count < max_retries`,
		StatusFailed,
	)
	if err != nil {
		return nil, fmt.Errorf("getting retryable downloads: %w", err)
	}
	defer rows.Close()

	var downloads []*Download
	for rows.Next() {
		d := &Download{}
		if err := rows.Scan(&d.ID, &d.URL, &d.Format, &d.Quality, &d.Bitrate, &d.Priority, &d.ProxyConfig, &d.RetryCount, &d.MaxRetries); err != nil {
			return nil, fmt.Errorf("scanning retryable download: %w", err)
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}

// RecoverInterruptedDownloads resets downloading status to queued on startup (crash recovery)
func (s *Store) RecoverInterruptedDownloads() (int64, error) {
	result, err := s.db.Exec(
		`UPDATE downloads SET status = ?, progress_percent = 0 WHERE status = ?`,
		StatusQueued, StatusDownloading,
	)
	if err != nil {
		return 0, fmt.Errorf("recovering interrupted downloads: %w", err)
	}
	return result.RowsAffected()
}

// CreateAnalyticsEvent records an analytics event
func (s *Store) CreateAnalyticsEvent(eventType string, downloadID *int64, site, format string, fileSize int64, durationSeconds int) error {
	_, err := s.db.Exec(
		`INSERT INTO analytics_events (event_type, download_id, site, format, file_size, duration_seconds)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		eventType, downloadID, site, format, fileSize, durationSeconds,
	)
	if err != nil {
		return fmt.Errorf("creating analytics event: %w", err)
	}
	return nil
}

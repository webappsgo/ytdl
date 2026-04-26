// Package config - Configuration file watcher for live reload.
// See AI.md PART 5: "Changes apply immediately without restart".
// Uses polling-based approach (no external dependencies).
// Only port/address changes require restart (log warning).
package config

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// ConfigWatcher watches the config file for changes and reloads
type ConfigWatcher struct {
	path         string
	lastModTime  time.Time
	lastSize     int64
	cfg          *ServerConfig
	mu           sync.RWMutex
	onReload     func(*ServerConfig)
	stopCh       chan struct{}
	pollInterval time.Duration
}

// NewConfigWatcher creates a watcher for the config file
func NewConfigWatcher(path string, cfg *ServerConfig, onReload func(*ServerConfig)) *ConfigWatcher {
	return &ConfigWatcher{
		path:         path,
		cfg:          cfg,
		onReload:     onReload,
		stopCh:       make(chan struct{}),
		pollInterval: 5 * time.Second,
	}
}

// Start begins watching the config file
func (w *ConfigWatcher) Start() {
	// Record initial state
	if info, err := os.Stat(w.path); err == nil {
		w.lastModTime = info.ModTime()
		w.lastSize = info.Size()
	}

	go w.pollLoop()
	log.Printf("Config watcher started for %s", w.path)
}

// Stop stops the watcher
func (w *ConfigWatcher) Stop() {
	close(w.stopCh)
}

// GetConfig returns the current config (thread-safe)
func (w *ConfigWatcher) GetConfig() *ServerConfig {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg
}

func (w *ConfigWatcher) pollLoop() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkForChanges()
		}
	}
}

func (w *ConfigWatcher) checkForChanges() {
	info, err := os.Stat(w.path)
	if err != nil {
		return
	}

	// Check if file changed
	if info.ModTime().Equal(w.lastModTime) && info.Size() == w.lastSize {
		return
	}

	w.lastModTime = info.ModTime()
	w.lastSize = info.Size()

	log.Println("Config file changed, reloading...")

	// Load new config
	newCfg := defaultConfig()
	if err := loadFromFile(w.path, newCfg); err != nil {
		log.Printf("Warning: failed to reload config: %v", err)
		return
	}

	// Check for changes that require restart
	w.mu.RLock()
	oldAddress := w.cfg.Server.Address
	oldPort := w.cfg.Server.Port
	w.mu.RUnlock()

	if newCfg.Server.Address != oldAddress || newCfg.Server.Port != oldPort {
		log.Printf("WARNING: address/port changed (%s:%d -> %s:%d) - restart required to apply",
			oldAddress, oldPort, newCfg.Server.Address, newCfg.Server.Port)
	}

	// Validate
	if err := validateConfig(newCfg); err != nil {
		log.Printf("Warning: new config invalid, keeping current: %v", err)
		return
	}

	// Apply new config
	w.mu.Lock()
	w.cfg = newCfg
	w.mu.Unlock()

	// Notify callback
	if w.onReload != nil {
		w.onReload(newCfg)
	}

	log.Println("Config reloaded successfully")
}

// GenerateDefaultConfig creates a default server.yml with comments above settings
func GenerateDefaultConfig(path string) error {
	// Comments ABOVE settings, never inline per AI.md
	defaultYAML := `# ytdl server configuration
# See documentation for all options

# Server settings
server:
  # Application mode: production or development
  mode: production
  # Listen address (0.0.0.0 for all interfaces)
  address: 0.0.0.0
  # Listen port (0 = auto-assign random 64xxx port)
  port: 0
  # URL path prefix
  baseurl: /
  # Admin panel URL path
  admin_path: admin
  # Debug mode (verbose logging)
  debug: false
  # Database settings
  database:
    # Driver: sqlite, libsql, pgx, mysql, sqlserver, mongodb
    driver: sqlite

# Download settings
download:
  # Number of concurrent download workers
  workers: 2
  # Maximum download queue size
  max_queue_size: 100
  # File retention in hours (0 = keep forever)
  retention_hours: 72
  # Maximum file size in bytes (0 = unlimited)
  max_file_size: 0
  # Global speed limit in bytes/sec (0 = unlimited)
  speed_limit: 0
  # Path to yt-dlp binary
  ytdlp_path: yt-dlp
  # Path to ffmpeg binary
  ffmpeg_path: ffmpeg
  # Output path template
  output_template: "{site}/{channel}/{title}.{ext}"
  # Default audio bitrate (CBR)
  default_audio_bitrate: 320k
  # Default video quality
  default_video_quality: "1080"
  # Default subtitle languages (auto-download)
  default_subtitle_languages:
    - en
    - es
`

	if err := os.MkdirAll(fmt.Sprintf("%s", path[:len(path)-len("/server.yml")]), 0700); err != nil {
		// Try parent directory
	}

	return os.WriteFile(path, []byte(defaultYAML), 0600)
}
